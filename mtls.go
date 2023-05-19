package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

var (
	// ErrAddCaCert is returned when the CA certificate is not added to the pool.
	ErrAddCACert = errors.New("failed to add CA cert to the pool")
)

// MtlsConfig returns a new tls.Config for the given DSN.
func MtlsConfig(certFileContent, keyFileContent, caCertFileContent string) (*tls.Config, error) {
	const (
		mtlsCertPath               = "mtlscert"
		mtlsKeyPath                = "mtlskey"
		mtlsCACertPath             = "mtlscacert"
		filePerm       fs.FileMode = 0600
	)

	certNonEscaped := strings.ReplaceAll(certFileContent, `\n`, "\n")
	keyNonEscaped := strings.ReplaceAll(keyFileContent, `\n`, "\n")
	caCertNonEscaped := strings.ReplaceAll(caCertFileContent, `\n`, "\n")

	if err := os.WriteFile(mtlsCertPath, []byte(certNonEscaped), filePerm); err != nil {
		return nil, fmt.Errorf("write cert file: %w", err)
	}
	if err := os.WriteFile(mtlsKeyPath, []byte(keyNonEscaped), filePerm); err != nil {
		return nil, fmt.Errorf("write key file: %w", err)
	}
	if err := os.WriteFile(mtlsCACertPath, []byte(caCertNonEscaped), filePerm); err != nil {
		return nil, fmt.Errorf("write ca cert file: %w", err)
	}
	return config(mtlsCertPath, mtlsKeyPath, mtlsCACertPath)
}

func config(certFilePath, keyFilePath, caCertPath string) (*tls.Config, error) {
	var cert tls.Certificate
	var err error
	cert, err = tls.LoadX509KeyPair(certFilePath, keyFilePath)
	if err != nil {
		return nil, fmt.Errorf("load x509 key pair: %w", err)
	}

	// nolint: gosec
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("read cacert: %w", err)
	}
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		return nil, fmt.Errorf("append certs from pem: %w", ErrAddCACert)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		RootCAs:      caCertPool,
	}, nil
}
