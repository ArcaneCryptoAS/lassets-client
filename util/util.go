package util

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/macaroons"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	macaroon2 "gopkg.in/macaroon.v2"
)

var log = logrus.New()

// ConnectToLnd connects to lnd-host using a tls.cert, admin.macaroon, and an net-address
func ConnectToLnd(lndDir, lndHost, network string) (lnrpc.LightningClient, error) {
	tlsPath := CleanAndExpandPath(fmt.Sprintf("%s/tls.cert", lndDir))
	macaroonPath := CleanAndExpandPath(fmt.Sprintf("%s/data/chain/bitcoin/%s/admin.macaroon", lndDir, network))

	tlsCreds, err := credentials.NewClientTLSFromFile(tlsPath, "")
	if err != nil {
		return nil, fmt.Errorf("could not extract tls cert: %w", err)
	}

	macaroonBytes, err := ioutil.ReadFile(macaroonPath)
	if err != nil {
		return nil, fmt.Errorf("could not extract macaroon: %w", err)
	}

	macaroon := &macaroon2.Macaroon{}
	if err = macaroon.UnmarshalBinary(macaroonBytes); err != nil {
		return nil, fmt.Errorf("could not unmarshal macaroonBytes: %w", err)
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(tlsCreds),
		grpc.WithBlock(),
		grpc.WithPerRPCCredentials(macaroons.NewMacaroonCredential(macaroon)),
	}
	ctx := context.Background()
	withTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(withTimeout, lndHost, opts...)
	if err != nil {
		log.WithFields(logrus.Fields{
			"tlsPath":      tlsPath,
			"macaroonPath": macaroonPath,
			"lndHost":      lndHost,
		}).Error("could not connect to lnd")
		return nil, fmt.Errorf("could not dial lnd: %w", err)
	}

	lncli := lnrpc.NewLightningClient(conn)

	return lncli, nil
}

// CleanAndExpandPath expands environment variables and leading ~ in the
// passed path, cleans the result, and returns it.
// This function is taken from https://github.com/btcsuite/btcd
func CleanAndExpandPath(path string) string {
	if path == "" {
		return ""
	}

	// Expand initial ~ to OS specific home directory.
	if strings.HasPrefix(path, "~") {
		var homeDir string
		user, err := user.Current()
		if err == nil {
			homeDir = user.HomeDir
		} else {
			homeDir = os.Getenv("HOME")
		}

		path = strings.Replace(path, "~", homeDir, 1)
	}

	return filepath.Clean(os.ExpandEnv(path))
}
