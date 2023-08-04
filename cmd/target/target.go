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

// This binary creates a tunnel target, which listen for incoming connection.
// If the incoming session is of supported type (SSH or GNMI), then it will be
// relayed to a local port with the provide port.
// Exmaple to use this binary:
//
//	target --config_file=$CONFIG_FILE
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/openconfig/grpctunnel/bidi"
	"github.com/openconfig/grpctunnel/tunnel"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/prototext"

	tcpb "github.com/openconfig/grpctunnel/cmd/target/proto/config"
	tpb "github.com/openconfig/grpctunnel/proto/tunnel"
)

var (
	configFile = flag.String("config_file", "", "The configuration file location")

	// for setting retry backoff when trying to connect to servers.
	retryBaseDelay     = time.Second
	retryMaxDelay      = time.Minute
	retryRandomization = 0.5
)

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

// runTargets starts a tunnel client for the given tunnel server.
// The client registeres targets/target types at the tunnel server and
// wait for incoming connection. For any supporte incoming session,
// it will be relayed to dialAddr.
func runTargets(ctx context.Context, targets []*tcpb.TunnelTarget, server *tcpb.TunnelServer) error {
	var opts []grpc.DialOption
	var err error
	cred := server.GetCredentials().GetTls()
	if len(cred.GetCertFile()) == 0 || len(cred.GetKeyFile()) == 0 {
		opts, err = tunnel.DialTLSCredsOpts(cred.GetCaFile())
	} else {
		opts, err = tunnel.DialmTLSCredsOpts(cred.GetCertFile(), cred.GetKeyFile(), cred.GetCaFile())
	}

	if err != nil {
		return err
	}

	clientConn, err := grpc.Dial(server.GetTunnelServerAddress(), opts...)
	if err != nil {
		return fmt.Errorf("grpc dial error: %v", err)
	}
	defer clientConn.Close()

	registerHandler := func(t tunnel.Target) error {
		for _, target := range targets {
			if t.ID == target.GetTarget() {
				return nil
			}
		}
		return fmt.Errorf("client cannot handle: %s", t.ID)
	}

	// For each tunnel session, it will dial the local port.
	handler := func(t tunnel.Target, i io.ReadWriteCloser) error {
		var dialAddr string
		for _, target := range targets {
			if t.ID == target.GetTarget() && t.Type == target.GetType() {
				dialAddr = target.GetDialAddress()
				break
			}
		}
		if len(dialAddr) == 0 {
			return fmt.Errorf("not maching dial port found for target: %s|%s", t.ID, t.Type)
		}

		conn, err := net.Dial("tcp", dialAddr)
		if err != nil {
			return fmt.Errorf("failed to dial %s: %v", dialAddr, err)
		}

		if err = bidi.Copy(i, conn); err != nil {
			log.Printf("error from bidi copy: %v", err)
		}
		return nil
	}

	ts := make(map[tunnel.Target]struct{})
	for _, target := range targets {
		t := tunnel.Target{ID: target.GetTarget(), Type: target.GetType()}
		ts[t] = struct{}{}
	}

	client, err := tunnel.NewClient(tpb.NewTunnelClient(clientConn), tunnel.ClientConfig{
		RegisterHandler: registerHandler,
		Handler:         handler,
	}, ts)

	if err != nil {
		return fmt.Errorf("failed to create tunnel client: %v", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Register and start listening.
	if err := client.Register(ctx); err != nil {
		return err
	}
	client.Start(ctx)
	return client.Error()
}

// runTargetsWithBackOff run with an exponential backoff retries.
func runTargetsWithBackOff(ctx context.Context, server *tcpb.TunnelServer, targets []*tcpb.TunnelTarget) {
	bo := getBackOff()
	for {
		if err := runTargets(ctx, targets, server); err != nil {
			log.Printf("tunnel client connected to %s failed: %v", server.GetTunnelServerAddress(), err)
			wait := bo.NextBackOff()
			log.Printf("reconnecting %s in %s\n", server.GetTunnelServerAddress(), wait)
			time.Sleep(wait)
		}

		select {
		case <-ctx.Done():
			return
		default:
		}
	}

}

// run collects all targets for corresponding servers and create a tunnel
// client for each server. It will terminate if the context is canceled.
// Failure of an individual target goroutine will not crash the program.
func run(ctx context.Context, conf *tcpb.TunnelTargetConfig) error {
	serverToTargets := make(map[*tcpb.TunnelServer][]*tcpb.TunnelTarget)
	for _, target := range conf.GetTunnelTarget() {
		servers := target.GetTunnelServer()
		if len(servers) == 0 {
			servers = conf.GetTunnelServerDefault()
		}
		if len(servers) == 0 {
			return fmt.Errorf("no server is provided for target: %s, type: %s", target.GetTarget(), target.GetType())
		}
		for _, server := range servers {
			serverToTargets[server] = append(serverToTargets[server], target)
		}
	}

	for server, targets := range serverToTargets {
		go runTargetsWithBackOff(ctx, server, targets)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	}
}

func main() {
	flag.Parse()

	if *configFile == "" {
		log.Fatal("config_file must be specified")
	}

	// Initialize configuration.
	buf, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("Could not read configuration from %q: %v", *configFile, err)
	}
	c := &tcpb.TunnelTargetConfig{}
	if err := prototext.Unmarshal(buf, c); err != nil {
		log.Fatalf("Could not parse configuration from %q: %v", *configFile, err)
	}

	if err := run(context.Background(), c); err != nil {
		log.Fatal(err)
	}
}
