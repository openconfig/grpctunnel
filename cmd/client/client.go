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
//
// This binary creates a tunnel client to proxy incoming connections
// over a grpc transport.
// Exmaples to use this binary with ssh's ProxyCommand option:
// TLS:
//		ssh -o ProxyCommand="client
//		--tunnel_server_address=localhost:$PORT \
//		--cert_file=$CERT_FILE \
//		--dial_target=target1 \
//		--dial_target_type=SSH" $USER@localhost
// mTLS:
// ssh -o ProxyCommand="client
// --tunnel_server_address=localhost:$PORT \
// --cert_file=$CERT_FILE \
// --key_file=$KEY_FILE \
// --ca_file=$CA_FILE \
// --dial_target=target1 \
// --dial_target_type=SSH" $USER@localhost
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/openconfig/grpctunnel/bidi"
	"github.com/openconfig/grpctunnel/tunnel"
	"google.golang.org/grpc"

	tpb "github.com/openconfig/grpctunnel/proto/tunnel"
)

var (
	tunnelAddress  = flag.String("tunnel_server_address", "", "The address of the tunnel")
	dialTarget     = flag.String("dial_target", "", "The client uses target to register at the server.")
	dialTargetType = flag.String("dial_target_type", "", "The type of target protocol, e.g. GNMI or SSH.")
	certFile       = flag.String("cert_file", "", "The certificate file location")
	keyFile        = flag.String("key_file", "", "The private key file location")
	caFile         = flag.String("ca_file", "", "The CA file location (for mTLS). If provided, it will be handled as mTLS")

	// for setting retry backoff when waiting for target.
	retryBaseDelay     = time.Second
	retryMaxDelay      = time.Minute
	retryRandomization = 0.5
)

// config defines the parameters to run a tunnel client.
type config struct {
	tunnelAddress,
	caFile,
	keyFile,
	certFile,
	dialTarget, // The remote target to dial
	dialTargetType string // The remote target type to dial
}

type stdIOConn struct {
	io.Reader
	io.WriteCloser
}

func getBackOff() *backoff.ExponentialBackOff {
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0 // Retry Subscribe indefinitely.
	bo.InitialInterval = retryBaseDelay
	bo.MaxInterval = retryMaxDelay
	bo.RandomizationFactor = retryRandomization
	return bo
}

// Run starts a tunnel client, connecting to the tunnel server via the provided tunnel address.
// The client uses the target to identify whether it can handle the target (u) sent by the server.
func run(ctx context.Context, conf config) error {
	var opts []grpc.DialOption
	var err error
	if len(conf.caFile) == 0 {
		opts, err = tunnel.DialTLSCredsOpts(conf.certFile)
	} else {
		opts, err = tunnel.DialmTLSCredsOpts(conf.certFile, conf.keyFile, conf.caFile)
	}

	if err != nil {
		return err
	}
	clientConn, err := grpc.Dial(conf.tunnelAddress, opts...)
	if err != nil {
		return fmt.Errorf("grpc dial error: %v", err)
	}
	defer clientConn.Close()

	peers := make(map[tunnel.Target]struct{})
	var peerMux sync.Mutex

	peerAddHandler := func(t tunnel.Target) error {
		peerMux.Lock()
		defer peerMux.Unlock()
		peers[t] = struct{}{}
		log.Printf("peer target %s added\n", t)
		return nil
	}

	peerDelHandler := func(t tunnel.Target) error {
		peerMux.Lock()
		defer peerMux.Unlock()
		if _, ok := peers[t]; ok {
			delete(peers, t)
			log.Printf("peer target %s deleted\n", t)
		}
		return nil
	}

	targets := make(map[tunnel.Target]struct{})
	client, err := tunnel.NewClient(tpb.NewTunnelClient(clientConn), tunnel.ClientConfig{
		PeerAddHandler: peerAddHandler,
		PeerDelHandler: peerDelHandler,
		Subscriptions:  []string{conf.dialTargetType},
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
		client.Start(ctx)
		if err := client.Error(); err != nil {
			errCh <- err
		}
	}()

	dialTarget := tunnel.Target{ID: conf.dialTarget, Type: conf.dialTargetType}
	foundDialTarget := func() bool {
		peerMux.Lock()
		defer peerMux.Unlock()
		_, ok := peers[dialTarget]
		return ok
	}

	// Dial the target with retry.
	go func() {
		bo := getBackOff()
		for !foundDialTarget() {
			wait := bo.NextBackOff()
			log.Printf("dial target %s (type: %s) not found. reconnecting in %s (all targets found: %s) \n", conf.dialTarget, conf.dialTargetType, wait, peers)
			time.Sleep(wait)
		}

		session, err := client.NewSession(dialTarget)
		if err != nil {
			log.Printf("error from new session: %v", err)
			errCh <- err
			return
		}
		log.Printf("new session established for target: %s\n", dialTarget)

		// Once a tunnel session is established, it connects it to a stdio.
		stdio := &stdIOConn{Reader: os.Stdin, WriteCloser: os.Stdout}
		if err = bidi.Copy(session, stdio); err != nil {
			log.Printf("error from bidi copy: %v\n", err)
			return
		}

	}()

	// Listen for any request to create a new session.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return fmt.Errorf("exiting: %s", err)
	}
}

func main() {
	flag.Parse()
	if err := run(context.Background(), config{
		tunnelAddress:  *tunnelAddress,
		dialTarget:     *dialTarget,
		dialTargetType: *dialTargetType,
		certFile:       *certFile,
		keyFile:        *keyFile,
		caFile:         *caFile,
	}); err != nil {
		log.Fatal(err)
	}
}
