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

package dialer

import (
	"context"
	"fmt"
	"testing"

	"github.com/openconfig/grpctunnel/tunnel"
	"google.golang.org/grpc"

	tunnelpb "github.com/openconfig/grpctunnel/proto/tunnel"
)

func TestFromServer(t *testing.T) {
	s := &tunnel.Server{}
	tests := []struct {
		desc    string
		s       *tunnel.Server
		wantErr bool
	}{
		{
			desc:    "missing tunnel server",
			wantErr: true,
		}, {
			desc:    "valid",
			s:       s,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		_, err := FromServer(tt.s)
		switch {
		case err == nil && tt.wantErr:
			t.Errorf("%v: got no error, want error.", tt.desc)
		case err != nil && !tt.wantErr:
			t.Errorf("%v: got error, want no error. err: %v", tt.desc, err)
		}
	}
}

func TestFromClient(t *testing.T) {
	c := &tunnel.Client{}
	tests := []struct {
		desc    string
		c       *tunnel.Client
		wantErr bool
	}{
		{
			desc:    "missing tunnel server",
			wantErr: true,
		}, {
			desc:    "valid",
			c:       c,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		_, err := FromClient(tt.c)
		switch {
		case err == nil && tt.wantErr:
			t.Errorf("%v: got no error, want error.", tt.desc)
		case err != nil && !tt.wantErr:
			t.Errorf("%v: got error, want no error. err: %v", tt.desc, err)
		}
	}
}
func mockServerConnGood(ctx context.Context, ts *tunnel.Server, target *tunnel.Target) (*tunnel.Conn, error) {
	return &tunnel.Conn{}, nil
}
func mockServerConnBad(ctx context.Context, ts *tunnel.Server, target *tunnel.Target) (*tunnel.Conn, error) {
	return nil, fmt.Errorf("bad tunnel conn")
}

func mockClientConnGood(ctx context.Context, ts *tunnel.Client, target *tunnel.Target) (*tunnel.Conn, error) {
	return &tunnel.Conn{}, nil
}
func mockClientConnBad(ctx context.Context, ts *tunnel.Client, target *tunnel.Target) (*tunnel.Conn, error) {
	return nil, fmt.Errorf("bad tunnel conn")
}

func mockGrpcDialContextGood(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return &grpc.ClientConn{}, nil
}

func mockGrpcDialContextBad(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return nil, fmt.Errorf("bad grpc conn")
}

func TestDialServer(t *testing.T) {
	s := &tunnel.Server{}
	dialer, err := FromServer(s)
	if err != nil {
		t.Fatalf("failed to create dialer: %v", err)
	}

	serverConnOrig := serverConn
	grpcDialContextOrig := grpcDialContext
	defer func() {
		serverConn = serverConnOrig
		grpcDialContext = grpcDialContextOrig
	}()

	tests := []struct {
		desc                 string
		tDial                string
		mockTunnelServerConn func(ctx context.Context, ts *tunnel.Server, target *tunnel.Target) (*tunnel.Conn, error)
		mockGrpcDialContext  func(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)
		wantErr              bool
	}{
		{
			desc:                 "succeeded",
			tDial:                "target1",
			mockTunnelServerConn: mockServerConnGood,
			mockGrpcDialContext:  mockGrpcDialContextGood,
			wantErr:              false,
		},
		{
			desc:                 "bad tunnel conn",
			tDial:                "target1",
			mockTunnelServerConn: mockServerConnBad,
			mockGrpcDialContext:  grpcDialContextOrig,
			wantErr:              true,
		},
		{
			desc:                 "bad grpc conn",
			tDial:                "target1",
			mockTunnelServerConn: mockServerConnGood,
			mockGrpcDialContext:  mockGrpcDialContextBad,
			wantErr:              true,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, tt := range tests {
		serverConn = tt.mockTunnelServerConn
		grpcDialContext = tt.mockGrpcDialContext

		_, err := dialer.DialContext(ctx, tt.tDial, tunnelpb.TargetType_GNMI_GNOI.String())
		// Check error.
		if tt.wantErr {
			if err == nil {
				t.Errorf("%v: got no error, want error.", tt.desc)
			}
			continue
		}
		if err != nil {
			t.Errorf("%v: got error, want no error: %v", tt.desc, err)
		}
	}
}

func TestDialClient(t *testing.T) {
	c := &tunnel.Client{}
	dialer, err := FromClient(c)
	if err != nil {
		t.Fatalf("failed to create dialer: %v", err)
	}

	clientConnOrig := clientConn
	grpcDialContextOrig := grpcDialContext
	defer func() {
		clientConn = clientConnOrig
		grpcDialContext = grpcDialContextOrig
	}()

	tests := []struct {
		desc                 string
		tDial                string
		mockTunnelClientConn func(ctx context.Context, tc *tunnel.Client, target *tunnel.Target) (*tunnel.Conn, error)
		mockGrpcDialContext  func(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)
		wantErr              bool
	}{
		{
			desc:                 "succeeded",
			tDial:                "target1",
			mockTunnelClientConn: mockClientConnGood,
			mockGrpcDialContext:  mockGrpcDialContextGood,
			wantErr:              false,
		},
		{
			desc:                 "bad tunnel conn",
			tDial:                "target1",
			mockTunnelClientConn: mockClientConnBad,
			mockGrpcDialContext:  grpcDialContextOrig,
			wantErr:              true,
		},
		{
			desc:                 "bad grpc conn",
			tDial:                "target1",
			mockTunnelClientConn: mockClientConnGood,
			mockGrpcDialContext:  mockGrpcDialContextBad,
			wantErr:              true,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, tt := range tests {
		clientConn = tt.mockTunnelClientConn
		grpcDialContext = tt.mockGrpcDialContext

		_, err := dialer.DialContext(ctx, tt.tDial, tunnelpb.TargetType_GNMI_GNOI.String())
		// Check error.
		if tt.wantErr {
			if err == nil {
				t.Errorf("%v: got no error, want error.", tt.desc)
			}
			continue
		}
		if err != nil {
			t.Errorf("%v: got error, want no error: %v", tt.desc, err)
		}
	}
}
