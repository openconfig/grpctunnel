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

package tunnel

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"

	tpb "github.com/openconfig/grpctunnel/proto/tunnel"
)

var (
	errWrite = errors.New("write error")
	errRead  = errors.New("read error")
)

type registerTestStream struct {
	grpc.ServerStream
	maxSends, sends  int
	ctx              context.Context
	streamRecv       []*tpb.RegisterOp
	streamSend       []*tpb.RegisterOp
	sendErr, recvErr bool
}

func (t *registerTestStream) Recv() (*tpb.RegisterOp, error) {
	if len(t.streamRecv) == 0 || t.recvErr {
		return nil, errRead
	}
	regOp := t.streamRecv[0]
	t.streamRecv = t.streamRecv[1:]
	return regOp, nil
}

func (t *registerTestStream) Send(s *tpb.RegisterOp) error {
	if t.sendErr || t.sends > t.maxSends {
		return errWrite
	}
	t.streamSend = append(t.streamSend, s)
	t.sends++
	return nil
}

func (t *registerTestStream) Context() context.Context {
	return t.ctx
}

type regTestStreamBlocking struct {
	*registerTestStream
}

func (t *regTestStreamBlocking) Recv() (*tpb.RegisterOp, error) {
	s := make(chan *tpb.RegisterOp)
	e := make(chan error)
	go func() {
		if len(t.registerTestStream.streamRecv) > 0 {
			ts, err := t.registerTestStream.Recv()
			if err != nil {
				e <- err
				return
			}
			s <- ts
		}
		return
	}()
	select {
	case <-t.ctx.Done():
		return nil, t.ctx.Err()
	case ts := <-s:
		return ts, nil
	case err := <-e:
		return nil, err
	}
}

type testDataStream struct {
	data *tpb.Data
	ctx  context.Context
	grpc.ServerStream
}

func (td *testDataStream) Recv() (*tpb.Data, error) {
	if td.data == nil {
		return nil, errRead
	}
	data := td.data
	td.data = nil
	return data, nil
}

func (td *testDataStream) Send(data *tpb.Data) error {
	if td.data != nil {
		return errWrite
	}
	td.data = data
	return nil
}

func (td *testDataStream) Context() context.Context {
	return td.ctx
}

type registerClientStream struct {
	grpc.ClientStream

	maxSends, sends  int
	ctx              context.Context
	streamRecv       []*tpb.RegisterOp
	streamSend       []*tpb.RegisterOp
	sendErr, recvErr bool
}

func (t *registerClientStream) Recv() (*tpb.RegisterOp, error) {
	if t.recvErr || len(t.streamRecv) == 0 {
		return nil, errRead
	}
	regOp := t.streamRecv[0]
	t.streamRecv = t.streamRecv[1:]
	return regOp, nil
}

func (t *registerClientStream) Send(s *tpb.RegisterOp) error {
	if t.sendErr || t.sends > t.maxSends {
		return errWrite
	}
	t.streamSend = append(t.streamSend, s)
	t.sends++
	return nil
}

func (t *registerClientStream) Context() context.Context {
	return t.ctx
}

type regClientStreamBlocking struct {
	*registerClientStream
}

func (t regClientStreamBlocking) Recv() (*tpb.RegisterOp, error) {
	s := make(chan *tpb.RegisterOp)
	e := make(chan error)
	go func() {
		if len(t.registerClientStream.streamRecv) > 0 {
			ts, err := t.registerClientStream.Recv()
			if err != nil {
				e <- err
				return
			}
			s <- ts
		}
	}()
	select {
	case <-t.ctx.Done():
		return nil, t.ctx.Err()
	case ts := <-s:
		return ts, nil
	case err := <-e:
		return nil, err
	}
}

type tunnelClientStream struct {
	grpc.ClientStream

	maxSends, sends  int
	ctx              context.Context
	streamRecv       []*tpb.Data
	streamSend       []*tpb.Data
	sendErr, recvErr bool
}

func (t *tunnelClientStream) Recv() (*tpb.Data, error) {
	if len(t.streamRecv) == 0 || t.recvErr {
		return nil, errRead
	}
	session := t.streamRecv[0]
	t.streamRecv = t.streamRecv[1:]
	return session, nil
}

func (t *tunnelClientStream) Send(s *tpb.Data) error {
	if t.sendErr || t.sends > t.maxSends {
		return errWrite
	}
	t.streamSend = append(t.streamSend, s)
	t.sends++
	return nil
}

func (t *tunnelClientStream) Context() context.Context {
	return t.ctx
}

type tunnelClient struct {
	regStream tpb.Tunnel_RegisterClient
	tunStream tpb.Tunnel_TunnelClient
}

func (tc *tunnelClient) Register(ctx context.Context, opts ...grpc.CallOption) (tpb.Tunnel_RegisterClient, error) {
	return tc.regStream, nil
}

func (tc *tunnelClient) Tunnel(ctx context.Context, opts ...grpc.CallOption) (tpb.Tunnel_TunnelClient, error) {
	return tc.tunStream, nil
}

type tunnelBlockingClient struct {
	*tunnelClient
	regStream tpb.Tunnel_RegisterClient
}

func (tbc *tunnelBlockingClient) Register(ctx context.Context, opts ...grpc.CallOption) (tpb.Tunnel_RegisterClient, error) {
	return regClientStreamBlocking{
		registerClientStream: &registerClientStream{
			ctx:          ctx,
			ClientStream: tbc.regStream,
			streamSend:   []*tpb.RegisterOp{},
			streamRecv:   []*tpb.RegisterOp{},
		},
	}, nil
}

type tunnelErrorClient struct{}

func (*tunnelErrorClient) Register(ctx context.Context, opts ...grpc.CallOption) (tpb.Tunnel_RegisterClient, error) {
	return nil, errors.New("test register error")
}

func (*tunnelErrorClient) Tunnel(ctx context.Context, opts ...grpc.CallOption) (tpb.Tunnel_TunnelClient, error) {
	return nil, errors.New("test tunnel error")
}

func TestRegSafeStreamSend(t *testing.T) {
	tests := []struct {
		name    string
		rs      *registerTestStream
		send    *tpb.RegisterOp
		wantErr bool
	}{
		{
			name:    "success",
			rs:      &registerTestStream{maxSends: 1},
			send:    &tpb.RegisterOp{Registration: &tpb.RegisterOp_Session{Session: &tpb.Session{Tag: 5}}},
			wantErr: false,
		},
		{
			name:    "error",
			rs:      &registerTestStream{sendErr: true},
			send:    &tpb.RegisterOp{Registration: &tpb.RegisterOp_Session{Session: &tpb.Session{Tag: 5}}},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := &regSafeStream{regStream: test.rs}
			err := s.Send(test.send)
			if err != nil && !test.wantErr {
				t.Errorf("Send() got err: %v, want nil", err)
			}
			if err == nil && test.wantErr {
				t.Error("Send got nil, want error")
			}
		})
	}
}

func TestDataIOStreamRead(t *testing.T) {
	tests := []struct {
		name    string
		bufSize int
	}{
		{"Read-1", 1},
		{"Read-3", 3},
		{"Read-7", 7},
		{"Read-15", 15},
		{"Read-20", 20},
		{"Read-50", 50},
		{"Read-100", 100},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := []byte("Test string.")
			tds := &testDataStream{data: &tpb.Data{Data: data}}
			d := &ioStream{dataSafeStream: dataSafeStream{dataStream: tds}}
			buf := make([]byte, test.bufSize)
			n, err := d.Read(buf)
			if err != nil {
				t.Errorf("Read() got err: %v, want nil", err)
			}
			if !bytes.Equal(buf[:n], data[:n]) {
				t.Errorf("Read() got %s, want %s", string(buf[:n]), string(data[:n]))
			}
			if !bytes.Equal(d.buf, data[n:]) {
				t.Errorf("Read's buf got %s, want %s", string(d.buf), string(data[n:]))
			}
		})
	}
	t.Run("FullRead", func(t *testing.T) {
		bufSize := 2
		data := []byte("Test string.")
		d := newIOStream(context.Background(), &testDataStream{data: &tpb.Data{Data: data}})
		total := 0
		tmp := []byte{}
		for total < len(data) {
			buf := make([]byte, bufSize)
			n, err := d.Read(buf)
			if err != nil {
				t.Fatalf("Read() got error: %v, want success.", err)
			}
			total += n
			tmp = append(tmp, buf[:n]...)
		}
		if !bytes.Equal(tmp, data) {
			t.Errorf("Read() got %s, want %s", string(tmp), string(data))
		}
	})
}

func TestDataIOStreamReadErrors(t *testing.T) {
	t.Run("eof", func(t *testing.T) {
		tds := &testDataStream{data: &tpb.Data{Close: true}}
		d := newIOStream(context.Background(), tds)
		buf := make([]byte, 50)
		_, err := d.Read(buf)
		if err != io.EOF {
			t.Errorf("Read() got %v, want io.EOF", err)
		}
	})
	t.Run("contextCanceled", func(t *testing.T) {
		tds := &testDataStream{}
		d := newIOStream(context.Background(), tds)
		if err := d.Close(); err != nil {
			t.Fatalf("Close() got %v, want success", err)
		}
		buf := []byte{}
		if _, err := d.Read(buf); err != context.Canceled {
			t.Errorf("Read() got %v, want %v", err, context.Canceled)
		}
	})
	t.Run("error", func(t *testing.T) {
		tds := &testDataStream{}
		d := newIOStream(context.Background(), tds)
		buf := []byte{}
		_, err := d.Read(buf)
		if err != errRead {
			t.Errorf("Read() got %v, want %s", err, errRead)
		}
	})
}

func TestDataIOStreamWrite(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tds := &testDataStream{}
		d := newIOStream(context.Background(), tds)
		write := []byte("Test strings are boring.")
		n, err := d.Write(write)
		if err != nil {
			t.Errorf("Write() got err: %v, want nil", err)
		}
		if n != len(write) {
			t.Errorf("Write() got %d bytes, want to write %d", n, len(write))
		}
	})
	t.Run("contextCancel", func(t *testing.T) {
		tds := &testDataStream{}
		d := newIOStream(context.Background(), tds)
		if err := d.Close(); err != nil {
			t.Fatalf("Close() got %v, want success", err)
		}
		write := []byte("Test strings are boring.")
		if _, err := d.Write(write); err != context.Canceled {
			t.Errorf("Write() got %v, want %v", err, context.Canceled)
		}
	})
	t.Run("failure", func(t *testing.T) {
		tds := &testDataStream{data: &tpb.Data{}}
		d := newIOStream(context.Background(), tds)
		write := []byte("Test strings are boring.")
		_, err := d.Write(write)
		if err == nil {
			t.Error("Write() got nil, want err")
		}
	})
}

func TestDataIOStreamClose(t *testing.T) {
	tds := &testDataStream{}
	d := newIOStream(context.Background(), tds)
	err := d.Close()
	if err != nil {
		t.Fatalf("Close() got %v, want nil", err)
	}
	if !tds.data.GetClose() {
		t.Errorf("Close() set close field to %t, want true", tds.data.GetClose())
	}
}

func TestEndpointMap(t *testing.T) {
	e := &endpoint{
		tag:       1,
		conns:     make(map[session]chan ioOrErr),
		increment: 1,
	}
	addr, err := net.ResolveTCPAddr("tcp", "localhost:22")
	if err != nil {
		t.Fatalf("failed to resolve address: %v", err)
	}
	addr1 := net.Addr(addr)
	addr2 := new(net.Addr)
	c1, c2 := make(chan ioOrErr), make(chan ioOrErr)
	t.Run("addConnection", func(t *testing.T) {
		tag := e.nextTag()
		want := true
		if err := e.addConnection(tag, addr1, c1); err != nil {
			t.Fatalf("addConnection() got %v, want success", err)
		}
		if _, ok := e.conns[session{tag, addr1}]; ok != want {
			t.Errorf("addConnection() got %v, want %v", ok, want)
		}
	})
	t.Run("addConnectionExists", func(t *testing.T) {
		tag := e.nextTag() - 1
		if err := e.addConnection(tag, addr1, c1); err == nil {
			t.Fatalf("addConnection() got %v, want error", err)
		}
	})
	t.Run("addConnectionTag", func(t *testing.T) {
		want := true
		if err := e.addConnection(5, *addr2, c2); err != nil {
			t.Fatalf("addConnection() got %v, want success", err)
		}
		if _, ok := e.conns[session{5, *addr2}]; ok != want {
			t.Errorf("addConnection() got %v, want %v", ok, want)
		}
	})
	t.Run("connection1", func(t *testing.T) {
		if got := e.connection(1, addr1); got != c1 {
			t.Errorf("connection() got %v, want %v", got, c1)
		}
	})
	t.Run("connection5", func(t *testing.T) {
		if got := e.connection(5, *addr2); got != c2 {
			t.Errorf("connection() got %v, want %v", got, c2)
		}
	})
	t.Run("connectionNil", func(t *testing.T) {
		if got := e.connection(-5, *addr2); got != nil {
			t.Errorf("connection() got %v, want nil", got)
		}
	})
	t.Run("deleteConnection", func(t *testing.T) {
		want := false
		e.deleteConnection(1, addr1)
		if _, got := e.conns[session{1, addr1}]; got != want {
			t.Fatalf("deleteConnection() got %v, want %v", got, want)
		}
	})
	t.Run("nextTag", func(t *testing.T) {
		if got := e.nextTag(); got != 3 {
			t.Fatalf("nextTag() got %d, wanted %d", got, 2)
		}
	})
}

func TestNewClientSuccess(t *testing.T) {
	for _, test := range []struct {
		name string
		cc   ClientConfig
	}{
		{
			name: "HandlerOnly",
			cc:   ClientConfig{Handler: func(Target, io.ReadWriteCloser) error { return nil }},
		},
		{
			name: "RegHandlerOnly",
			cc:   ClientConfig{RegisterHandler: func(Target) error { return nil }},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewClient(nil, test.cc, map[Target]struct{}{}); err == nil {
				t.Fatal("NewClient() got success, wanted error")
			}
		})
	}
}

func TestNewClientFailure(t *testing.T) {
	for _, test := range []struct {
		name string
		cc   ClientConfig
	}{
		{
			name: "bothHandlers",
			cc: ClientConfig{
				RegisterHandler: func(Target) error { return nil },
				Handler:         func(Target, io.ReadWriteCloser) error { return nil },
			},
		},
		{
			name: "noHandlers",
			cc:   ClientConfig{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewClient(nil, test.cc, map[Target]struct{}{}); err != nil {
				t.Fatalf("NewClient() got %v, wanted success", err)
			}
		})
	}
}

func serverHandlerError(ss ServerSession, rwc io.ReadWriteCloser) error {
	return errors.New("test: can't handle")
}
func serverRegHandlerError(ss ServerSession) error {
	return errors.New("test: no registered handler")
}

func TestNewServerFailure(t *testing.T) {
	for _, test := range []struct {
		name string
		sc   ServerConfig
	}{
		{
			name: "HandlerOnly",
			sc: ServerConfig{
				Handler: func(ServerSession, io.ReadWriteCloser) error { return nil },
			},
		},
		{
			name: "RegHandlerOnly",
			sc: ServerConfig{
				RegisterHandler: func(ServerSession) error { return nil },
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewServer(test.sc); err == nil {
				t.Fatalf("NewServer() got success, wanted error")
			}
		})
	}
}

func TestNewServerSuccess(t *testing.T) {
	for _, test := range []struct {
		name string
		sc   ServerConfig
	}{
		{
			name: "bothHandlers",
			sc: ServerConfig{
				RegisterHandler: func(ServerSession) error { return nil },
				Handler:         func(ServerSession, io.ReadWriteCloser) error { return nil },
			},
		},
		{
			name: "noHandlers",
			sc:   ServerConfig{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewServer(test.sc); err != nil {
				t.Fatalf("NewServer() got %v, wanted success", err)
			}
		})
	}
}

func TestServerClient(t *testing.T) {
	s, err := NewServer(ServerConfig{})
	if err != nil {
		t.Fatalf("failed to create new server: %v", err)
	}
	ta, err := net.ResolveTCPAddr("tcp", "localhost:22")
	if err != nil {
		t.Fatalf("failed to resolve address: %v", err)
	}
	addr := net.Addr(ta)
	rs := &regSafeStream{}
	t.Run("AddEntry", func(t *testing.T) {
		want := true
		if err := s.addClient(addr, rs); err != nil {
			t.Fatalf("addClient() got %v, want success", err)
		}
		if _, got := s.clients[addr]; got != want {
			t.Errorf("addClient() got %v, want %v", got, want)
		}
	})
	t.Run("AddDuplicateEntry", func(t *testing.T) {
		if err := s.addClient(addr, rs); err == nil {
			t.Fatalf("addClient() got %v, want error", err)
		}
	})
	t.Run("DeleteEntry", func(t *testing.T) {
		want := false
		s.deleteClient(addr)
		if _, got := s.clients[addr]; got != want {
			t.Errorf("deleteClient() got %v, want %v", got, want)
		}
	})
	t.Run("Get-NoEntryInMap", func(t *testing.T) {
		var addr net.Addr
		if got := s.clientInfo(addr); !got.IsZero() {
			t.Errorf("peerHandler() got %v, want nil", got)
		}
	})
	t.Run("Get-ValidEntryInMap", func(t *testing.T) {
		var addr net.Addr
		s.clients[addr] = clientRegInfo{rs: rs}
		if got := s.clientInfo(addr); got.rs != rs {
			t.Errorf("client() got %v, want %v", got, rs)
		}
	})
}

func TestServerRegister(t *testing.T) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:45000")
	if err != nil {
		t.Fatalf("failed to resolve address: %v", err)
	}
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: addr})
	for _, test := range []struct {
		name      string
		ctx       context.Context
		sendErr   bool
		addClient bool
		maxSends  int
	}{
		{name: "NoPeer", ctx: context.Background()},
		{name: "AddClientError", ctx: ctx, addClient: true, maxSends: 10},
		{name: "RecvError", ctx: ctx, maxSends: 10},
	} {
		s, err := NewServer(ServerConfig{})
		if err != nil {
			t.Fatalf("NewServer(ServerConfig{}) failed: %v", err)
		}
		t.Run(test.name, func(t *testing.T) {
			if test.addClient {
				if err := s.addClient(addr, &regSafeStream{}); err != nil {
					t.Fatalf("failed to add client for test: %v", err)
				}
			}
			if err := s.Register(&registerTestStream{
				ctx:        test.ctx,
				sendErr:    test.sendErr,
				maxSends:   test.maxSends,
				streamRecv: []*tpb.RegisterOp{},
			}); err == nil {
				t.Fatalf("Register() want error, got %v", err)
			}
		})
	}
}

func TestServerTunnelError(t *testing.T) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:45000")
	if err != nil {
		t.Fatalf("failed to resolve address: %v", err)
	}
	for _, test := range []struct {
		name string
		ctx  context.Context
		data *tpb.Data
	}{
		{name: "ReceiveError"},
		{name: "DataError", data: &tpb.Data{Data: []byte("data!!")}},
		{name: "PeerError", ctx: context.Background(), data: &tpb.Data{}},
		{name: "ConnectionError", ctx: peer.NewContext(context.Background(), &peer.Peer{Addr: addr}), data: &tpb.Data{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, err := NewServer(ServerConfig{})
			if err != nil {
				t.Fatalf("NewServer(ServerConfig{}) failed: %v", err)
			}
			if err := s.Tunnel(&testDataStream{ctx: test.ctx, data: test.data}); err == nil {
				t.Fatalf("Register() want error, got %v", err)
			}
		})
	}
}

func TestServerTunnelSuccess(t *testing.T) {
	s, err := NewServer(ServerConfig{})
	if err != nil {
		t.Fatalf("NewServer(ServerConfig{}) failed: %v", err)
	}
	t.Run("success", func(t *testing.T) {
		addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:45000")
		if err != nil {
			t.Fatalf("failed to resolve address: %v", err)
		}
		i := make(chan ioOrErr)
		s.addConnection(1, addr, i)
		ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: addr})
		go func() {
			d := <-i
			d.rwc.Close()
		}()
		if err := s.Tunnel(&testDataStream{ctx: ctx, data: &tpb.Data{Tag: 1}}); err != nil {
			t.Fatalf("Register() want success, got %v", err)
		}
	})
}

func TestServerNewSessionErrors(t *testing.T) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:45000")
	if err != nil {
		t.Fatalf("failed to resolve address: %v", err)
	}
	tempAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:45001")
	if err != nil {
		t.Fatalf("failed to resolve address: %v", err)
	}
	for _, test := range []struct {
		name       string
		sAddr      net.Addr
		clientAddr net.Addr
		rs         regStream
		conn       bool
	}{
		{name: "NoClients", sAddr: nil, rs: &regSafeStream{}},
		{name: "NoStream", clientAddr: tempAddr, sAddr: addr, rs: &regSafeStream{}},
		{name: "AddConnectionError", clientAddr: addr, sAddr: addr, rs: &registerTestStream{maxSends: 10}, conn: true},
		{name: "AddConnectionFailSend", clientAddr: addr, sAddr: addr, rs: &registerTestStream{maxSends: 0}, conn: true},
		{name: "StreamSendError", clientAddr: addr, sAddr: addr, rs: &registerTestStream{sendErr: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, err := NewServer(ServerConfig{})
			if err != nil {
				t.Fatalf("NewServer(ServerConfig{}) failed: %v", err)
			}
			if test.clientAddr != nil {
				if err := s.addClient(test.clientAddr, test.rs); err != nil {
					t.Fatalf("failed to add client to test server: %v", err)
				}
			}
			if test.conn {
				if err := s.addConnection(1, test.clientAddr, make(chan ioOrErr)); err != nil {
					t.Fatalf("failed to add connection to test server: %v", err)
				}
			}
			if _, err := s.NewSession(context.Background(), ServerSession{Addr: test.sAddr}); err == nil {
				t.Fatalf("NewSession() want error, got %v", err)
			}
		})
	}
}

func TestServerNewSession(t *testing.T) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:45000")
	if err != nil {
		t.Fatalf("failed to resolve address: %v", err)
	}
	for _, test := range []struct {
		name       string
		numClients int
		ioe        ioOrErr
		wantErr    bool
		sendAll    bool
	}{
		{name: "RetChError", ioe: ioOrErr{err: errors.New("test error")}, wantErr: true},
		{name: "Success", numClients: 2, ioe: ioOrErr{rwc: &ioStream{}}},
		{name: "SuccessSendall", sendAll: true, numClients: 2, ioe: ioOrErr{rwc: &ioStream{}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, err := NewServer(ServerConfig{})
			if err != nil {
				t.Fatalf("NewServer(ServerConfig{}) failed: %v", err)
			}
			if err := s.addClient(addr, &registerTestStream{maxSends: 10}); err != nil {
				t.Fatalf("failed to add client to test server: %v", err)
			}
			addrL := make([]net.Addr, test.numClients)
			if test.numClients > 0 {
				for i := 2; i <= test.numClients; i++ {
					addr1, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("127.0.0.1:%d", i+20000))
					if err != nil {
						t.Fatalf("failed to resolve address: %v", err)
					}
					if err := s.addClient(addr1, &registerTestStream{maxSends: 10}); err != nil {
						t.Fatalf("failed to add client to test server: %v", err)
					}
					addrL = append(addrL, addr1)
				}
			}
			go func() {
				ch := s.connection(1, addr)
				for ch == nil {
					time.Sleep(1 * time.Second)
					ch = s.connection(1, addr)
				}
				if test.sendAll {
					for _, addr := range addrL {
						c := s.connection(1, addr)
						if c == nil {
							continue
						}
						c <- ioOrErr{err: errors.New("test error")}
					}
				}
				ch <- test.ioe
			}()
			rwc, err := s.NewSession(context.Background(), ServerSession{
				Addr: addr,
				Target: Target{
					ID:   "1",
					Type: "foo",
				},
			})
			if err == nil && test.wantErr {
				t.Fatalf("NewSession() want error, got %v", err)
			}
			if err != nil && !test.wantErr {
				t.Fatalf("NewSession() want success, got err: %v", err)
			}
			if rwc != nil && rwc != test.ioe.rwc {
				t.Fatalf("NewSession() want %v, got %v", test.ioe.rwc, rwc)
			}
		})
	}
}

func TestClientRunBlocked(t *testing.T) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:45000")
	if err != nil {
		t.Fatalf("failed to resolve: %v", err)
	}
	p := &peer.Peer{Addr: addr}
	ctx, cancel := context.WithCancel(
		peer.NewContext(context.Background(), p),
	)
	defer cancel()
	client := &tunnelBlockingClient{}
	c, err := NewClient(client, ClientConfig{}, map[Target]struct{}{})
	if err != nil {
		t.Fatalf("failed to create new client: %v", err)
	}
	if err := c.Register(ctx); err != nil {
		t.Fatalf("c.Register() failed: %v", err)
	}

	chanErr := make(chan error, 2)
	clientRun := func() {
		c.Start(ctx)
		if err := c.Error(); err != nil {
			chanErr <- err
		}
	}

	// This test verifies that Run cannot be invoked multiple times in parallel,
	// and only the first call will be successful.
	for i := 0; i < 2; i++ {
		go clientRun()
	}
	err = <-chanErr
	select {
	case err = <-chanErr:
		t.Fatalf("TestClientRunBlocked() received second error: %v", err)
	default:
	}
	// Cancel the context to stop the other goroutine, and wait for the second error.
	cancel()
	<-chanErr

	// This test verifies that Run can be invoked a second time.
	ctx, cancel = context.WithTimeout(
		peer.NewContext(context.Background(), p),
		time.Second*5,
	)
	defer cancel()
	if err := c.Register(ctx); err != nil {
		t.Fatalf("c.Register() failed: %v", err)
	}
	go clientRun()
	select {
	case <-ctx.Done():
	case err = <-chanErr:
		t.Fatalf("c.Run() failed: %v", err)
	}
}

func TestClientRun(t *testing.T) {
	client, err := NewClient(&tunnelClient{}, ClientConfig{}, map[Target]struct{}{})
	if err != nil {
		t.Fatalf("TestClientRun: failed to create new client: %v", err)
	}
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:45000")
	if err != nil {
		t.Fatalf("failed to resolve: %v", err)
	}
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: addr})
	for _, test := range []struct {
		name    string
		wantErr bool
		tc      tpb.TunnelClient
		ctx     context.Context
		c       *Client
	}{
		{
			name:    "RegisterError",
			wantErr: true,
			ctx:     context.Background(),
			tc:      &tunnelErrorClient{},
		},
		{
			name:    "peerError",
			wantErr: true,
			ctx:     context.Background(),
			tc: &tunnelClient{regStream: &registerClientStream{
				ctx:      context.Background(),
				maxSends: 10,
			}},
		},
		{
			name:    "capabilitiesError",
			wantErr: true,
			ctx:     ctx,
			tc: &tunnelClient{regStream: &registerClientStream{
				ctx:      ctx,
				maxSends: 10,
			}},
			c: client,
		},
		{
			name:    "startError",
			wantErr: true,
			ctx:     ctx,
			tc: &tunnelClient{regStream: &registerClientStream{
				ctx:      ctx,
				maxSends: 10,
				streamRecv: []*tpb.RegisterOp{
					&tpb.RegisterOp{},
				},
			}},
			c: client,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var err error
			if test.c == nil {
				test.c, err = NewClient(test.tc, ClientConfig{}, map[Target]struct{}{})
				if err != nil {
					t.Fatalf("failed to create new client: %v", err)
				}
			} else {
				test.c.tc = test.tc
			}
			err = test.c.Register(test.ctx)
			if err != nil {
				if !test.wantErr {
					t.Fatalf("c.Register() got %v, want success", err)
				}
				return
			}

			test.c.Start(test.ctx)
			err = test.c.Error()
			if err == nil && test.wantErr {
				t.Fatal("c.Run() got success, want error")
			}
			if err != nil && !test.wantErr {
				t.Fatalf("c.Run() got %v, want success", err)
			}
		})
	}
}

func TestClientStart(t *testing.T) {
	for _, test := range []struct {
		name    string
		cc      ClientConfig
		client  tpb.TunnelClient
		wantErr bool
	}{
		{
			name:    "SendError",
			wantErr: true,
			client: &tunnelClient{
				regStream: &registerClientStream{
					recvErr: true,
				},
			},
			cc: ClientConfig{},
		},
		{
			name:    "streamNotAccepted",
			wantErr: true,
			client: &tunnelClient{
				regStream: &registerClientStream{
					streamRecv: []*tpb.RegisterOp{
						&tpb.RegisterOp{Registration: &tpb.RegisterOp_Session{Session: &tpb.Session{Tag: 1, Accept: false, Target: "testID"}}},
					},
					streamSend: []*tpb.RegisterOp{},
				},
			},
		},
		{
			name:    "streamHandlerErr",
			wantErr: true,
			client: &tunnelClient{
				regStream: &registerClientStream{
					streamRecv: []*tpb.RegisterOp{
						&tpb.RegisterOp{Registration: &tpb.RegisterOp_Session{Session: &tpb.Session{Tag: 1, Accept: true, Target: "testID"}}},
					},
					streamSend: []*tpb.RegisterOp{},
				},
			},
			cc: ClientConfig{
				Handler: func(t Target, rwc io.ReadWriteCloser) error {
					return errors.New("test: no registered handler")
				},
				RegisterHandler: func(t Target) error {
					return errors.New("test: can't handle")
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, err := NewClient(test.client, test.cc, map[Target]struct{}{})
			if err != nil {
				t.Fatalf("NewClient() failed: %v", err)
			}
			rs, err := test.client.Register(context.Background(), []grpc.CallOption{}...)
			if err != nil {
				t.Fatalf("client register failed: %v", err)
			}
			c.rs = &regSafeStream{regStream: rs}
			var addr net.Addr
			c.addr = addr
			retCh := make(chan ioOrErr, 1)
			err = c.addConnection(-1, addr, retCh)

			ctx := context.Background()
			ctx, c.cancelFunc = context.WithCancel(ctx)
			c.Start(ctx)
			err = c.Error()
			if err == nil && test.wantErr {
				t.Fatal("c.Start() got success, want error")
			}
			if err != nil && !test.wantErr {
				t.Fatalf("c.Start() got %v, want success", err)
			}
		})
	}
}

func TestClientReturnedStream(t *testing.T) {
	for _, test := range []struct {
		name    string
		tag     int32
		cc      ClientConfig
		client  tpb.TunnelClient
		wantErr bool
	}{
		{
			name:    "ConnectionError",
			tag:     -2,
			wantErr: true,
			client:  &tunnelClient{},
			cc:      ClientConfig{},
		},
		{
			name:    "ConnectionError",
			tag:     -1,
			wantErr: true,
			client: &tunnelClient{
				regStream: &registerClientStream{},
				tunStream: &tunnelClientStream{
					sendErr: true,
				},
			},
			cc: ClientConfig{},
		},
		{
			name: "Success",
			tag:  -1,
			client: &tunnelClient{
				regStream: &registerClientStream{},
				tunStream: &tunnelClientStream{
					streamSend: []*tpb.Data{},
				},
			},
			cc: ClientConfig{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, err := NewClient(test.client, test.cc, map[Target]struct{}{})
			if err != nil {
				t.Fatalf("NewClient() failed: %v", err)
			}
			var addr net.Addr
			c.addr = addr
			retCh := make(chan ioOrErr, 1)
			err = c.addConnection(-1, addr, retCh)
			err = c.returnedStream(context.Background(), test.tag)
			if err == nil && test.wantErr {
				t.Fatal("c.returnedStream() got success, want error")
			}
			if err != nil && !test.wantErr {
				t.Fatalf("c.returnedStream() got %v, want success", err)
			}
			if test.tag == -1 {
				ioe := <-retCh
				if test.wantErr && ioe.err == nil {
					t.Fatal("c.returnedStream() got success, want error")
				}
				if !test.wantErr && ioe.err != nil {
					t.Fatalf("c.returnedStream() got %v, want success", err)
				}
			}
		})
	}
}

func TestClientNewClientStream(t *testing.T) {
	for _, test := range []struct {
		name    string
		cc      ClientConfig
		client  tpb.TunnelClient
		wantErr bool
	}{
		{
			name:    "TunnelError",
			client:  &tunnelErrorClient{},
			cc:      ClientConfig{},
			wantErr: true,
		},
		{
			name: "SendError",
			client: &tunnelClient{
				regStream: &registerClientStream{},
				tunStream: &tunnelClientStream{
					sendErr: true,
				},
			},
			cc:      ClientConfig{},
			wantErr: true,
		},
		{
			name: "Success",
			client: &tunnelClient{
				regStream: &registerClientStream{},
				tunStream: &tunnelClientStream{
					streamSend: []*tpb.Data{},
				},
			},
			cc: ClientConfig{
				Handler:         func(t Target, rwc io.ReadWriteCloser) error { return nil },
				RegisterHandler: func(t Target) error { return nil },
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, err := NewClient(test.client, test.cc, map[Target]struct{}{})
			if err != nil {
				t.Fatalf("NewClient() failed: %v", err)
			}
			err = c.newClientStream(context.Background(), 1, Target{ID: "testString"})
			if err == nil && test.wantErr {
				t.Fatal("c.newClientStream() got success, want error")
			}
			if err != nil && !test.wantErr {
				t.Fatalf("c.newClientStream() got %v, want success", err)
			}
		})
	}
}

func TestClientStreamHandler(t *testing.T) {
	handlerError := func(t Target, rwc io.ReadWriteCloser) error {
		return errors.New("test: no registered handler")
	}
	regHandlerError := func(t Target) error {
		return errors.New("test: can't handle")
	}
	for _, test := range []struct {
		name    string
		cc      ClientConfig
		client  tpb.TunnelClient
		tag     int32
		wantErr bool
	}{
		{
			name:    "regStreamSendErr",
			wantErr: true,
			tag:     2,
			client: &tunnelClient{
				regStream: &registerClientStream{
					streamRecv: []*tpb.RegisterOp{
						&tpb.RegisterOp{Registration: &tpb.RegisterOp_Session{Session: &tpb.Session{Tag: 1, Accept: true, Target: "testID"}}},
					},
					sendErr: true,
				},
			},
			cc: ClientConfig{
				Handler:         handlerError,
				RegisterHandler: regHandlerError,
			},
		},
		{
			name:    "returnedStreamErr",
			wantErr: true,
			tag:     -1,
			client: &tunnelClient{
				regStream: &registerClientStream{
					streamRecv: []*tpb.RegisterOp{
						&tpb.RegisterOp{Registration: &tpb.RegisterOp_Session{Session: &tpb.Session{Tag: 1, Accept: true, Target: "testID"}}},
					},
					streamSend: []*tpb.RegisterOp{},
				},
				tunStream: &tunnelClientStream{
					sendErr: true,
					recvErr: true,
				},
			},
			cc: ClientConfig{
				Handler:         handlerError,
				RegisterHandler: regHandlerError,
			},
		},
		{
			name: "returnedStreamSuccess",
			tag:  -1,
			client: &tunnelClient{
				regStream: &registerClientStream{
					streamRecv: []*tpb.RegisterOp{
						&tpb.RegisterOp{Registration: &tpb.RegisterOp_Session{Session: &tpb.Session{Tag: 1, Accept: true, Target: "testID"}}},
					},
					streamSend: []*tpb.RegisterOp{},
				},
				tunStream: &tunnelClientStream{
					streamRecv: []*tpb.Data{},
					streamSend: []*tpb.Data{},
					maxSends:   10,
				},
			},
			cc: ClientConfig{
				Handler:         handlerError,
				RegisterHandler: regHandlerError,
			},
		},
		{
			name:    "newClientStreamError",
			wantErr: true,
			tag:     1,
			client: &tunnelClient{
				regStream: &registerClientStream{
					streamRecv: []*tpb.RegisterOp{
						&tpb.RegisterOp{Registration: &tpb.RegisterOp_Session{Session: &tpb.Session{Tag: 1, Accept: true, Target: "testID"}}},
					},
					streamSend: []*tpb.RegisterOp{},
				},
				tunStream: &tunnelClientStream{
					sendErr: true,
					recvErr: true,
				},
			},
			cc: ClientConfig{
				Handler:         func(t Target, rwc io.ReadWriteCloser) error { return nil },
				RegisterHandler: func(t Target) error { return nil },
			},
		},
		{
			name: "newClientStreamSuccess",
			tag:  1,
			client: &tunnelClient{
				regStream: &registerClientStream{
					streamRecv: []*tpb.RegisterOp{
						&tpb.RegisterOp{Registration: &tpb.RegisterOp_Session{Session: &tpb.Session{Tag: 1, Accept: true, Target: "testID"}}},
					},
					streamSend: []*tpb.RegisterOp{},
				},
				tunStream: &tunnelClientStream{
					maxSends: 10,
				},
			},
			cc: ClientConfig{
				Handler:         func(t Target, rwc io.ReadWriteCloser) error { return nil },
				RegisterHandler: func(t Target) error { return nil },
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, err := NewClient(test.client, test.cc, map[Target]struct{}{})
			if err != nil {
				t.Fatalf("NewClient() failed: %v", err)
			}
			rs, err := test.client.Register(context.Background(), []grpc.CallOption{}...)
			if err != nil {
				t.Fatalf("client Register failed: %v", err)
			}
			c.rs = &regSafeStream{regStream: rs}
			addr, err := net.ResolveTCPAddr("tcp", "localhost:22")
			if err != nil {
				t.Fatalf("failed to resolve IP addr: %v", err)
			}
			c.addr = addr
			retCh := make(chan ioOrErr, 1)
			err = c.addConnection(-1, addr, retCh)
			err = c.streamHandler(context.Background(), test.tag, Target{ID: "testID"})
			if err == nil && test.wantErr {
				t.Fatal("c.streamHandler() got success, want error")
			}
			if err != nil && !test.wantErr {
				t.Fatalf("c.streamHandler() got %v, want success", err)
			}
		})
	}
}

func TestClientNewSessionClientNotStarted(t *testing.T) {
	var tc tpb.TunnelClient
	c, err := NewClient(tc, ClientConfig{}, map[Target]struct{}{})
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}
	_, err = c.NewSession(Target{})
	if err == nil {
		t.Fatalf("NewSession() got success, wanted error: %v", err)
	}
}

func TestClientNewSessionAddConnectionFailure(t *testing.T) {
	var tc tpb.TunnelClient
	addr, err := net.ResolveTCPAddr("tcp", "192.168.0.1:22")
	if err != nil {
		t.Fatalf("failed to resolve addr: %v", err)
	}

	c, err := NewClient(tc, ClientConfig{}, map[Target]struct{}{})
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}
	c.addr = addr

	if err := c.addConnection(-1, addr, make(chan ioOrErr)); err != nil {
		t.Fatalf("AddConnection() failed: %v", err)
	}

	_, err = c.NewSession(Target{})
	if err == nil {
		t.Fatalf("NewSession() got success, wanted error: %v", err)
	}
}

func TestClientNewSessionSendFailure(t *testing.T) {
	var tc tpb.TunnelClient
	addr, err := net.ResolveTCPAddr("tcp", "192.168.0.1:22")
	if err != nil {
		t.Fatalf("failed to resolve addr: %v", err)
	}

	c, err := NewClient(tc, ClientConfig{}, map[Target]struct{}{})
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}
	c.addr = addr

	c.rs = &regSafeStream{
		regStream: &registerClientStream{
			sendErr: true,
		},
	}
	_, err = c.NewSession(Target{})
	if err == nil {
		t.Fatalf("NewSession() got success, wanted error: %v", err)
	}
}

func TestClientNewSessionRetChErr(t *testing.T) {
	var tc tpb.TunnelClient
	addr, err := net.ResolveTCPAddr("tcp", "192.168.0.1:22")
	if err != nil {
		t.Fatalf("failed to resolve addr: %v", err)
	}

	c, err := NewClient(tc, ClientConfig{}, map[Target]struct{}{})
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}
	c.addr = addr

	c.rs = &regSafeStream{
		regStream: &registerClientStream{
			maxSends:   10,
			streamRecv: []*tpb.RegisterOp{},
		},
	}
	go func() {
		var ioe chan ioOrErr
		for ioe == nil {
			ioe = c.connection(-1, addr)
		}
		ioe <- ioOrErr{err: errors.New("test error")}
	}()
	_, err = c.NewSession(Target{})
	if err == nil {
		t.Fatalf("NewSession() got success, wanted error: %v", err)
	}
}
