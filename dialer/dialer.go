/*
Copyright 2022 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package dialer implements a gRPC dialer over a tunneled connection.
package dialer

import (
	"context"
	"fmt"
	"net"

	"github.com/openconfig/grpctunnel/tunnel/tunnel"
	"google.golang.org/grpc"
)

// ClientDialer performs dialing to targets behind a given tunnel connection.
type ClientDialer struct {
	tc *tunnel.Client
}

// FromClient creates a new target dialer with an existing tunnel client
// connection.
// Sample code with error handling elided:
//
// conn, err := grpc.DialContext(ctx, tunnelAddress)
// client := tpb.NewTunnelClient(conn)
// tc := tunnel.NewClient(client, tunnel.ClientConfig{}, nil)
// d := dialer.FromClient(tc)
func FromClient(tc *tunnel.Client) (*ClientDialer, error) {
	if tc == nil {
		return nil, fmt.Errorf("tunnel server connection is nil")
	}
	return &ClientDialer{tc: tc}, nil
}

// DialContext establishes a grpc.Conn to a remote tunnel client via the
// attached tunnel server and returns an error if the connection is not
// established.
//
// The dialer can be used to create connections to multiple targets behind the
// same tunnel server used to instantiate the dialer.
//
// conn, err := d.DialContext(ctx, "target1", "target-type1", opts1)
// conn, err := d.DialContext(ctx, "target2", "target-type2", opts2)
func (d *ClientDialer) DialContext(ctx context.Context, target, targetType string, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	withContextDialer := grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		session, err := d.tc.NewSession(tunnel.Target{ID: target, Type: targetType})
		if err != nil {
			return nil, err
		}
		return &tunnel.Conn{session}, nil
	})
	opts = append(opts, withContextDialer)
	return grpc.DialContext(ctx, target, opts...)
}

