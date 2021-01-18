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
// Package main runs either a tunnel client or server based on the command line arguments
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/openconfig/grpctunnel/cmd/client"
	"github.com/openconfig/grpctunnel/cmd/server"
)

var (
	tunnelAddress = flag.String("tunnel_address", "", "The address of the tunnel")
	listenAddress = flag.String("listen_address", "", "The address on which the server listens for incoming connections")
	dialAddress   = flag.String("dial_address", "", "The address the client will dial when it receives a connection")
	target        = flag.String("target", "", `The client uses target to verify whether or not it can handle the target sent by the server.
The server sends this target to the client, to see if the client can handle the request.`)
	certFile = flag.String("cert_file_name", "", "The public key certificate file")
	keyFile  = flag.String("cert_key_file", "", "Server only: the certificate key file location")
)

type config struct {
	tunnelAddress, listenAddress, dialAddress, target, certFile, keyFile string
}

func run(ctx context.Context, conf config) error {
	if conf.tunnelAddress == "" {
		return fmt.Errorf("cannot establish tunnel: no tunnel address set")
	}
	switch {
	case conf.listenAddress != "" && conf.dialAddress != "":
		return fmt.Errorf("ambiguous flags provided: provide either a dial or a listen address")
	case conf.listenAddress != "":
		log.Print("Starting server.")
		if err := server.Run(ctx, server.Config{
			TunnelAddress: conf.tunnelAddress,
			ListenAddress: conf.listenAddress,
			CertFile:      conf.certFile,
			KeyFile:       conf.keyFile,
			Target:        conf.target,
		}); err != nil && err != context.Canceled {
			return fmt.Errorf("server run failed: %v", err)
		}
		log.Print("Server run finished.")
	case conf.dialAddress != "":
		log.Print("Starting client.")
		if err := client.Run(ctx, client.Config{
			TunnelAddress: conf.tunnelAddress,
			DialAddress:   conf.dialAddress,
			CertFile:      conf.certFile,
			Target:        conf.target,
		}); err != nil && err != context.Canceled {
			return fmt.Errorf("Client run failed: %v", err)
		}
		log.Print("Client run finished.")
	default:
		return fmt.Errorf("cannot establish tunnel: provide a dial address or a listen address")
	}
	return nil
}
func main() {
	flag.Parse()
	if err := run(context.Background(), config{
		tunnelAddress: *tunnelAddress,
		listenAddress: *listenAddress,
		dialAddress:   *dialAddress,
		target:        *target,
		certFile:      *certFile,
		keyFile:       *keyFile,
	}); err != nil {
		log.Fatal(err)
	}
}
