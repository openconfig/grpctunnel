package tunnel

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// ServerTLSCredsOpts returns a slice of ServerOption with TLS
// credential obtained from certFile and keyFile.
func ServerTLSCredsOpts(certFile, keyFile string) ([]grpc.ServerOption, error) {
	var opts []grpc.ServerOption
	if certFile != "" && keyFile != "" {
		creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load credentials: %v", err)
		}
		opts = append(opts, grpc.Creds(creds))
	}
	return opts, nil
}

// ServermTLSCredsOpts returns a slice of ServerOption with mTLS
// credential obtained from certFile, keyFile and caFile.
func ServermTLSCredsOpts(certFile, keyFile, caFile string) ([]grpc.ServerOption, error) {
	var opts []grpc.ServerOption
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load x509 key pair: %s", err)
	}
	certPool := x509.NewCertPool()
	bs, err := ioutil.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA file: %s", err)
	}
	if ok := certPool.AppendCertsFromPEM(bs); !ok {
		return nil, fmt.Errorf("failed to append client certs")
	}
	tlsConfig := &tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{certificate},
		ClientCAs:    certPool,
	}
	opts = append(opts, grpc.Creds(credentials.NewTLS(tlsConfig)))
	return opts, nil
}

// DialTLSCredsOpts returns a slice of DialOption with TLS
// credential obtained from certFile.
func DialTLSCredsOpts(certFile string) ([]grpc.DialOption, error) {
	opts := []grpc.DialOption{grpc.WithDefaultCallOptions()}
	if certFile == "" {
		opts = append(opts, grpc.WithInsecure())
	} else {
		creds, err := credentials.NewClientTLSFromFile(certFile, "")
		if err != nil {
			return nil, fmt.Errorf("failed to load credentials: %v", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	}
	return opts, nil
}

// DialmTLSCredsOpts returns a slice of DialOption with mTLS
// credential obtained from certFile, keyFile and caFile.
func DialmTLSCredsOpts(certFile, keyFile, caFile string) ([]grpc.DialOption, error) {
	opts := []grpc.DialOption{grpc.WithDefaultCallOptions()}
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load x509 key pair: %s", err)
	}
	certPool := x509.NewCertPool()
	bs, err := ioutil.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA file: %s", err)
	}
	if ok := certPool.AppendCertsFromPEM(bs); !ok {
		return nil, fmt.Errorf("failed to append client certs")
	}
	tlsConfig := &tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{certificate},
		RootCAs:      certPool, // use RootCAs to avoid "certificate signed by unknown authority" error.
	}
	opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	return opts, nil
}
