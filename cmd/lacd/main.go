package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/gorilla/mux"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc/credentials"

	"github.com/ArcaneCryptoAS/lassets-client/util"

	"github.com/boltdb/bolt"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"google.golang.org/grpc"

	"github.com/ArcaneCryptoAS/lassets-client/build"
	"github.com/ArcaneCryptoAS/lassets-client/larpc"
)

var (
	contractsBucket = []byte("contracts")
	defaultDBName   = "laclient.db"
)

var (
	defaultClientPort         = 10456
	defaultRestPort           = 8081
	defaultClientDir          = util.CleanAndExpandPath("~/.lac")
	defaultNetwork            = "regtest"
	defaultNetAddress         = "localhost:10456"
	defaultPriceserverAddress = "http://127.0.0.1:3001"

	defaultLndDir     = util.CleanAndExpandPath("~/.lnd")
	defaultLndRPCPort = "localhost:10011"

	serverAddress = "localhost:10455"
)

var (
	// here we store the latest prices
	price = map[string]float64{
		"USD": 0.00,
		"NOK": 0.00,
	}
)

// define possible flag names here
const (
	flag_port                = "port"
	flag_rest_port           = "restport"
	flag_laddir              = "laddir"
	flag_network             = "network"
	flag_rebalancefrequency  = "rebalancefrequency"
	flag_lnddir              = "lnddir"
	flag_netaddress          = "netaddress"
	flag_lndrpchost          = "lndrpchost"
	flag_priceserver_address = "priceserver_address"
	flag_serveraddress       = "serveraddress"
	flag_insecureserver      = "insecureserver"
)

var log = logrus.New()

func main() {
	app := cli.NewApp()
	app.Name = "ladclient"
	app.Version = build.Version()
	app.Usage = "client daemon for Lightning Assets"
	app.Flags = []cli.Flag{
		// lightning asset daemon flags
		cli.IntFlag{
			Name:  flag_port,
			Value: defaultClientPort,
			Usage: "port to run lightning asset grpc daemon on",
		},
		cli.IntFlag{
			Name:  flag_rest_port,
			Value: defaultRestPort,
			Usage: "port to run lightning asset rest server",
		},
		cli.StringFlag{
			Name:  flag_laddir,
			Usage: "the location of lad dir",
			Value: defaultClientDir,
		},
		cli.StringFlag{
			Name:  flag_network,
			Usage: "which bitcoin network to run on, regtest | testnet | mainnet",
			Value: defaultNetwork,
		},
		cli.IntFlag{
			Name:  flag_rebalancefrequency,
			Usage: "how often to rebalance channels",
		},
		cli.StringFlag{
			Name:  flag_netaddress,
			Usage: "our ip-address, that other hosts can reach us at",
			Value: defaultNetAddress,
		},
		cli.StringFlag{
			Name:  flag_priceserver_address,
			Usage: "host:port the price server is running on",
			Value: defaultPriceserverAddress,
		},
		cli.StringFlag{
			Name:  flag_serveraddress,
			Usage: "the host:port the asset server is running on",
			Value: serverAddress,
		},
		cli.BoolFlag{
			Name:  flag_insecureserver,
			Usage: "whether the connection to the server should use TLS or not",
		},

		// flags specific to connecting to lnd
		cli.StringFlag{
			Name:  flag_lnddir,
			Usage: "the full path to lnd directory",
			Value: defaultLndDir,
		},
		cli.StringFlag{
			Name:  flag_lndrpchost,
			Usage: "host:port of lnd daemon",
			Value: defaultLndRPCPort,
		},
	}
	app.Action = runClient

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("[lad]: %v", err)
	}
}

func runClient(c *cli.Context) error {
	// create lightning asset client dir, used for saving tls and db files
	ladDir := c.String(flag_laddir)
	if _, err := os.Stat(ladDir); os.IsNotExist(err) {
		os.Mkdir(ladDir, os.ModePerm) // 0777 permission
	}

	db, err := bolt.Open(path.Join(ladDir, defaultDBName), 0600, &bolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("could not open database: %w", err)
	}
	defer db.Close()

	err = createBucketsIfNotExist(db)
	if err != nil {
		return err
	}

	// connect to lnd
	lncli, err := util.ConnectToLnd(
		c.String(flag_lnddir),
		c.String(flag_lndrpchost),
		c.String(flag_network),
	)
	if err != nil {
		return fmt.Errorf("could not connect to lnd: %w", err)
	}

	// create channel that new contracts and new payments are sent to
	contractCh := make(chan larpc.ClientContract)
	paymentCh := make(chan larpc.Payment)

	ladServer, cleanup, err := newServerConnection(c.String(
		flag_serveraddress), c.Bool(flag_insecureserver), "")
	if err != nil {
		return fmt.Errorf("could not connect to asset server: %w", err)
	}
	defer cleanup()

	// TODO: Add bitmex websocket that listens to the price

	assetServer := AssetClient{
		lncli:          lncli,
		db:             db,
		port:           c.Int(flag_port),
		netAddress:     c.String(flag_netaddress),
		server:         ladServer,

		contractCh: contractCh,
		paymentsCh: paymentCh,
	}

	// create grpc server that listens to grpc requests
	grpcServer := grpc.NewServer()
	larpc.RegisterAssetClientServer(grpcServer, &assetServer)

	// start webserver that uses normal http / http2, used for communicating with front-end
	go func() {
		wrappedGrpc := grpcweb.WrapServer(grpcServer)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrappedGrpc.ServeHTTP(w, r)
		})

		router := mux.NewRouter()
		router.Use(headerMiddleware)
		router.PathPrefix("/").Handler(handler)

		log.Infoln("rest server listening on port", c.Int(flag_rest_port))
		res := http.ListenAndServe(fmt.Sprintf(":%d", c.Int(flag_rest_port)), router)
		log.Fatal(res)
	}()

	// create and run the grpc daemon
	port := c.Int(flag_port)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("could not listen: %w", err)
	}

	log.Infof("grpc server listening on port %d", port)
	// the thread will hang on this next line
	err = grpcServer.Serve(lis)
	if err != nil {
		return fmt.Errorf("could not serve: %w", err)
	}

	return nil
}

func headerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Headers", "x-grpc-web")
		w.Header().Add("Access-Control-Allow-Headers", "content-type")
		next.ServeHTTP(w, r)
	})
}

func createBucketsIfNotExist(db *bolt.DB) error {
	// create bucket if it doesnt exist
	return db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(contractsBucket)
		if err != nil {
			return fmt.Errorf("could not create bucket: %w", err)
		}
		// add additional buckets here
		return nil
	})
}

type grpcServerConnection struct {
	server larpc.AssetServerClient
	conn   *grpc.ClientConn
}

// newServerConnection opens a connection to the swap server.
func newServerConnection(address string, insecure bool,
	tlsPath string) (*grpcServerConnection, func(), error) {

	// Create a dial options array.
	var opts []grpc.DialOption

	// There are three options to connect to a swap server, either insecure,
	// using a self-signed certificate or with a certificate signed by a
	// public CA.
	switch {
	case insecure:
		opts = append(opts, grpc.WithInsecure())

	case tlsPath != "":
		// Load the specified TLS certificate and build
		// transport credentials
		creds, err := credentials.NewClientTLSFromFile(tlsPath, "")
		if err != nil {
			return nil, nil, err
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))

	default:
		creds := credentials.NewTLS(&tls.Config{})
		opts = append(opts, grpc.WithTransportCredentials(creds))
	}

	serverConn, err := grpc.Dial(address, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to connect to RPC server: %v",
			err)
	}

	server := larpc.NewAssetServerClient(serverConn)

	cleanUp := func() {
		serverConn.Close()
	}

	conn := &grpcServerConnection{
		server: server,
		conn:   serverConn,
	}

	return conn, cleanUp, nil
}
