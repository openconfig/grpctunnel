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

// Package client creates a tunnel client to proxy incoming connections
// over a grpc transport.
package client

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/openconfig/grpctunnel/bidi"
	"github.com/openconfig/grpctunnel/tunnel"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	tpb "github.com/openconfig/grpctunnel/proto/tunnel"
)

// Config defines the parameters to run a tunnel client.
type Config struct {
	TunnelAddress, DialAddress, ListenAddress, CertFile, Target, TargetType string
}

func listen(ctx context.Context, c *tunnel.Client, listenAddress string, target tunnel.Target) error {
	l, err := net.Listen("tcp", listenAddress)
	if err != nil {
		return fmt.Errorf("failed to listen: %s: %v", listenAddress, err)
	}
	defer l.Close()

	errCh := make(chan error)
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				select {
				case errCh <- fmt.Errorf("failed to accept connection: %v", err):
				default:
				}
				return
			}
			// Errors from this goroutine will be logged only, because we don't want an
			// underlying stream issue to tear the server down
			go func(conn net.Conn) {
				defer conn.Close()
				session, err := c.NewSession(target)
				if err != nil {
					log.Printf("error from new session: %v", err)
					return
				}
				if err = bidi.Copy(session, conn); err != nil {
					log.Printf("error from bidi copy: %v", err)
				}
			}(conn)
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

// Run starts a tunnel client, connecting to the tunnel server via the provided tunnel address.
// The client uses the target to identify whether it can handle the target (u) sent by the server.
func Run(ctx context.Context, conf Config) error {
	opts := []grpc.DialOption{grpc.WithDefaultCallOptions()}
	if conf.CertFile == "" {
		opts = append(opts, grpc.WithInsecure())
	} else {
		creds, err := credentials.NewClientTLSFromFile(conf.CertFile, "")
		if err != nil {
			return fmt.Errorf("failed to load credentials: %v", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	}
	clientConn, err := grpc.Dial(conf.TunnelAddress, opts...)
	if err != nil {
		return fmt.Errorf("grpc dial error: %v", err)
	}
	defer clientConn.Close()

	registerHandler := func(t tunnel.Target) error {
		if t.ID != conf.Target {
			return fmt.Errorf("client cannot handle: %s", t.ID)
		}
		return nil
	}

	handler := func(_ tunnel.Target, i io.ReadWriteCloser) error {
		conn, err := net.Dial("tcp", conf.DialAddress)
		if err != nil {
			log.Printf("Error dialing client: %v", err)
			return err
		}

		if err = bidi.Copy(i, conn); err != nil {
			// Logging this error only as we don't want the client to stop because an
			// underlying stream had an issue
			log.Printf("Copy error: %v", err)
		}

		return nil
	}

	peerAddCh := make(chan tunnel.Target, 1)
	peerAddHandler := func(t tunnel.Target) error {
		peerAddCh <- t
		return nil
	}

	peerDelCh := make(chan tunnel.Target, 1)
	peerDelHandler := func(t tunnel.Target) error {
		peerDelCh <- t
		return nil
	}

	targets := make(map[tunnel.Target]struct{})
	t := tunnel.Target{ID: conf.Target, Type: conf.TargetType}
	targets[t] = struct{}{}
	client, err := tunnel.NewClient(tpb.NewTunnelClient(clientConn), tunnel.ClientConfig{
		RegisterHandler: registerHandler,
		Handler:         handler,
		Subscriptions:   []string{conf.TargetType},
		PeerAddHandler:  peerAddHandler,
		PeerDelHandler:  peerDelHandler,
	}, targets)

	if err != nil {
		return fmt.Errorf("failed to create tunnel client: %v", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 2)
	go func() {
		if err := client.Register(ctx); err != nil {
			errCh <- err
			return
		}
		if err := client.Start(ctx); err != nil {
			errCh <- err
		}
	}()

	// listen for any request to create a new session
	select {
	case target := <-peerAddCh:
		return listen(ctx, client, conf.ListenAddress, target)
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}
