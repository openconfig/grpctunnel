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

// Package server creates a tunnel server which can proxy traffic from a listener
// to the tunnel client.
package server

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/openconfig/grpctunnel/bidi"
	"github.com/openconfig/grpctunnel/tunnel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	tpb "github.com/openconfig/grpctunnel/proto/tunnel"
)

// listen starts the server listener in it's own goroutine. This goroutine will continue
// until either the listener encounters an error, or until the provided context is cancelled.
// The listener goroutine will spawn new goroutines and request a new tunnel stream per
// accepted connection. These goroutines will run until the copy has completed or an error
// is encountered.
func listen(ctx context.Context, server *tunnel.Server, listenAddress string, targets *[]tunnel.Target) error {
	l, err := net.Listen("tcp", listenAddress)
	if err != nil {
		return fmt.Errorf("failed to listen: %s: %v", listenAddress, err)
	}
	defer l.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

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
				log.Printf("received connection. trying a new session.")

				session, err := server.NewSession(ctx, tunnel.ServerSession{Target: (*targets)[0]})
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

// Config defines the parameters to run a tunnel server.
type Config struct {
	TunnelAddress, ListenAddress, CertFile, KeyFile string
}

// Run starts a tunnel server using the provided config.
// The server will spin off a goroutine to listen on the listenAddress provided
// in the config. It will accept and proxy these connections through the tunnel.
// The spawned goroutines will run until an error is encountered by either the
// server or the listener.
func Run(ctx context.Context, conf Config) error {
	var opts []grpc.ServerOption
	if conf.CertFile != "" && conf.KeyFile != "" {
		creds, err := credentials.NewServerTLSFromFile(conf.CertFile, conf.KeyFile)
		if err != nil {
			return fmt.Errorf("failed to load credentials: %v", err)
		}
		opts = append(opts, grpc.Creds(creds))
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

	l, err := net.Listen("tcp", conf.TunnelAddress)
	if err != nil {
		return fmt.Errorf("failed to create listener: %v", err)
	}
	defer l.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 2)

	go func() {
		if err := s.Serve(l); err != nil {
			errCh <- err
		}
	}()

	go func() {
		if err := listen(ctx, ts, conf.ListenAddress, &targets); err != nil {
			errCh <- err
		}
	}()

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

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}
