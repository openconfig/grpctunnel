//
// Copyright 2019 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

// This binary runs a tunnel server, which serves as a proxy between tunnel clients.
// Exmaples to use this binary is with ssh's ProxyCommand option:
// TLS:
//
//	server --tunnel_address=localhost:$PORT \
//	--cert_file=$CERT_FILE \
//	--key_file=$KEY_FILE
//
// mTLS:
//
//	server --tunnel_address=localhost:$PORT \
//	--cert_file=$CERT_FILE \
//	--key_file=$KEY_FILE \
//	--ca_file=$CA_FILE
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/openconfig/grpctunnel/tunnel"
	"google.golang.org/grpc"

	tpb "github.com/openconfig/grpctunnel/proto/tunnel"
)

var (
	tunnelAddress = flag.String("tunnel_address", "", "The address of the tunnel")
	certFile      = flag.String("cert_file", "", "The public certificate file location")
	keyFile       = flag.String("key_file", "", "The private key file location")
	caFile        = flag.String("ca_file", "", "The CA file location(for mTLS). If provided, it will be handled as mTLS")
)

// config defines the parameters to run a tunnel server.
type config struct {
	tunnelAddress, certFile, keyFile, caFile string
}

// Run starts a tunnel server using the provided config.
// The server will accept and proxy connections between tunnel clients.
// The spawned goroutines will run until an error is encountered by either the
// server or the client.
func run(ctx context.Context, conf config) error {
	var opts []grpc.ServerOption
	var err error
	if len(conf.caFile) == 0 {
		opts, err = tunnel.ServerTLSCredsOpts(conf.certFile, conf.keyFile)
	} else {
		opts, err = tunnel.ServermTLSCredsOpts(conf.certFile, conf.keyFile, conf.caFile)
	}

	if err != nil {
		return err
	}

	s := grpc.NewServer(opts...)
	defer s.Stop()

	targets := []tunnel.Target{}
	addTagHandler := func(t tunnel.Target) error {
		log.Printf("handling target addition: (%s:%s)", t.ID, t.Type)
		targets = append(targets, t)
		return nil
	}

	deleteTagHandler := func(t tunnel.Target) error {
		log.Printf("handling target deletion: (%s:%s)", t.ID, t.Type)
		return nil
	}

	ts, err := tunnel.NewServer(tunnel.ServerConfig{
		AddTargetHandler:    addTagHandler,
		DeleteTargetHandler: deleteTagHandler,
	})
	if err != nil {
		return fmt.Errorf("failed to create new server: %v", err)
	}
	tpb.RegisterTunnelServer(s, ts)

	l, err := net.Listen("tcp", conf.tunnelAddress)
	if err != nil {
		return fmt.Errorf("failed to create listener to TunnelAddress: %v", err)
	}
	defer l.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errChTS := ts.ErrorChan()
	go func() {
		for {
			select {
			case err := <-errChTS:
				log.Printf("tunnel serer error: %v", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	return s.Serve(l)
}

func main() {
	flag.Parse()
	if err := run(context.Background(), config{
		tunnelAddress: *tunnelAddress,
		certFile:      *certFile,
		keyFile:       *keyFile,
		caFile:        *caFile,
	}); err != nil {
		log.Fatal(err)
	}
}
