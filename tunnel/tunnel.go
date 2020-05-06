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

// Package tunnel defines the a TCP over gRPC transport client and server.
package tunnel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"

	tpb "github.com/openconfig/grpctunnel/proto/tunnel"
)

// regStream abstracts the tunnel Register streams.
type regStream interface {
	Recv() (*tpb.Session, error)
	Send(*tpb.Session) error
}

// regStream implements regStream, and wraps the send method in a mutex.
// Per https://github.com/grpc/grpc-go/blob/master/stream.go#L110 it is safe to
// concurrently read and write in separate goroutines, but it is not safe to
// call read (or write) on the same stream in different goroutines.
type regSafeStream struct {
	w sync.Mutex
	regStream
}

// Send blocks until it sends the provided data to the stream.
func (r *regSafeStream) Send(data *tpb.Session) error {
	r.w.Lock()
	defer r.w.Unlock()
	return r.regStream.Send(data)
}

// dataStream abstracts the Tunnel Client and Server streams.
type dataStream interface {
	Recv() (*tpb.Data, error)
	Send(*tpb.Data) error
}

type dataSafeStream struct {
	mu sync.Mutex
	dataStream
}

// Send blocks until it sends the provided data to the stream.
func (d *dataSafeStream) Send(data *tpb.Data) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.dataStream.Send(data)
}

// ioStream defines a gRPC stream that implements io.ReadWriteCloser.
type ioStream struct {
	buf    []byte
	doneCh <-chan struct{}
	cancel context.CancelFunc
	dataSafeStream
}

// newIOStream creates and returns a stream which implements io.ReadWriteCloser.
func newIOStream(ctx context.Context, d dataStream) *ioStream {
	ctx, cancel := context.WithCancel(ctx)
	return &ioStream{
		dataSafeStream: dataSafeStream{
			dataStream: d,
		},
		cancel: cancel,
		doneCh: ctx.Done(),
	}
}

// Read implements io.Reader.
func (s *ioStream) Read(b []byte) (int, error) {
	select {
	case <-s.doneCh:
		return 0, context.Canceled
	default:
		if len(s.buf) != 0 {
			n := copy(b, s.buf)
			s.buf = s.buf[n:]
			return n, nil
		}
		data, err := s.Recv()
		if err != nil {
			return 0, err
		}
		if data.Close {
			return 0, io.EOF
		}
		n := copy(b, data.Data)
		if n < len(data.Data) {
			s.buf = data.Data[n:]
		}
		return n, nil
	}
}

// Write implements io.Writer.
func (s *ioStream) Write(b []byte) (int, error) {
	select {
	case <-s.doneCh:
		return 0, context.Canceled
	default:
		err := s.Send(&tpb.Data{Data: b})
		if err != nil {
			return 0, err
		}
		return len(b), nil
	}
}

// Close implements io.Closer.
func (s *ioStream) Close() error {
	s.cancel()
	return s.Send(&tpb.Data{Close: true})
}

// session defines a unique connection for the tunnel.
type session struct {
	tag  int32
	addr net.Addr // The address from the gRPC stream peer.
}

// ioOrErr is used to return either an io.ReadWriteCloser or an error.
type ioOrErr struct {
	rwc io.ReadWriteCloser
	err error
}

// endpoint defines the shared structure between the client and server.
type endpoint struct {
	increment int32

	mu    sync.RWMutex
	tag   int32
	conns map[session]chan ioOrErr
}

// connection returns an IOStream chan for the provided tag and address.
func (e *endpoint) connection(tag int32, addr net.Addr) chan ioOrErr {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.conns[session{tag, addr}]
}

// addConnection creates a connection in the endpoint conns map. If the provided
// tag and address are already in the map, add connection returns an error.
func (e *endpoint) addConnection(tag int32, addr net.Addr, ioe chan ioOrErr) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.conns[session{tag, addr}]; ok {
		return fmt.Errorf("connection %d exists for %q", tag, addr.String())
	}
	e.conns[session{tag, addr}] = ioe
	return nil
}

// deleteConnection deletes a connection from the endpoint conns map, if it exists.
func (e *endpoint) deleteConnection(tag int32, addr net.Addr) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.conns, session{tag, addr})
}

// nextTag returns the next tag from the endpoint, and then increments it to
// prevent tag collisions with other streams.
func (e *endpoint) nextTag() int32 {
	e.mu.Lock()
	defer e.mu.Unlock()
	tag := e.tag
	e.tag += e.increment
	return tag
}

// ServerRegHandlerFunc defines the targetIDs that the handler function can accept.
type ServerRegHandlerFunc func(ss ServerSession) error

// ServerHandlerFunc handles sessions the server receives from the client.
type ServerHandlerFunc func(ss ServerSession, rwc io.ReadWriteCloser) error

// ServerConfig contains the config for the server.
type ServerConfig struct {
	RegisterHandler ServerRegHandlerFunc
	Handler         ServerHandlerFunc
}

// ServerSession is used by NewSession and the register handler. In the register
// handler it is used by the client to indicate what it's trying to connect to.
// In NewSession, ServerSession will indicate what target to connect to, as well
// as potentially specifying the client to connect to.
type ServerSession struct {
	Addr     net.Addr
	TargetID string
}

// Server is the server implementation of an endpoint.
type Server struct {
	endpoint

	sc ServerConfig

	phmu         sync.RWMutex
	peerHandlers map[net.Addr]bool

	cmu     sync.RWMutex
	clients map[net.Addr]regStream
}

// NewServer creates a new tunnel server.
func NewServer(sc ServerConfig) (*Server, error) {
	if (sc.RegisterHandler == nil) != (sc.Handler == nil) {
		return nil, errors.New("tunnel: can't create server: only 1 handler set")
	}
	return &Server{
		clients:      make(map[net.Addr]regStream),
		peerHandlers: make(map[net.Addr]bool),
		sc:           sc,
		endpoint: endpoint{
			tag:       1,
			conns:     make(map[session]chan ioOrErr),
			increment: 1,
		},
	}, nil
}

// peerHandler checks whether a peer addr has a handler defined.
func (s *Server) peerHandler(addr net.Addr) bool {
	s.phmu.RLock()
	defer s.phmu.RUnlock()
	return s.peerHandlers[addr]
}

// addPeerHandler sets whether a peer addr has a handler.
func (s *Server) addPeerHandler(addr net.Addr) {
	s.phmu.RLock()
	defer s.phmu.RUnlock()
	s.peerHandlers[addr] = true
}

// deletePeerHandle deletes a peer handler from the endpoint peerHandlers map, if it exists.
func (s *Server) deletePeerHandler(addr net.Addr) {
	s.phmu.RLock()
	defer s.phmu.RUnlock()
	delete(s.peerHandlers, addr)
}

// client returns an IOStream chan for the provided tag and address.
func (s *Server) client(addr net.Addr) regStream {
	s.cmu.Lock()
	defer s.cmu.Unlock()
	return s.clients[addr]
}

// addClient adds a client to the clients map.
func (s *Server) addClient(addr net.Addr, rs regStream) error {
	s.cmu.Lock()
	defer s.cmu.Unlock()
	if _, ok := s.clients[addr]; ok {
		return fmt.Errorf("client exists for %q", addr.String())
	}
	s.clients[addr] = rs
	return nil
}

// deleteClient removes a client from the clients map.
func (s *Server) deleteClient(addr net.Addr) {
	s.cmu.Lock()
	defer s.cmu.Unlock()
	delete(s.clients, addr)
}

// server capabilities sends and receives capabilities between the endpoints when the
// registration RPC is established. This is the first set of messages that should
// be sent on the registration stream.
func (s *Server) capabilities(addr net.Addr, rs regStream) error {
	lh := s.sc.Handler != nil && s.sc.RegisterHandler != nil
	ph, err := capabilities(lh, rs)
	if err != nil {
		return fmt.Errorf("capabilities error: %v", err)
	}
	if ph {
		s.addPeerHandler(addr)
	}
	return nil
}

// Register handles the gRPC register stream(s).
// The receive direction calls the client-installed handle function.
// The send direction is used by NewSession to get new tunnel sessions.
func (s *Server) Register(stream tpb.Tunnel_RegisterServer) error {
	rs := &regSafeStream{regStream: stream}
	p, ok := peer.FromContext(stream.Context())
	if !ok {
		return errors.New("no peer from stream context")
	}
	if err := s.capabilities(p.Addr, rs); err != nil {
		return fmt.Errorf("error from capabilities: %v", err)
	}
	if err := s.addClient(p.Addr, rs); err != nil {
		return fmt.Errorf("error adding client: %v", err)
	}
	defer s.deleteClient(p.Addr)
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()
	streamErr := make(chan error, 1)
	// The loop within this goroutine will return when an error is received which
	// streamErr is unavailable to receive. This will happen when ctx is canceled.
	go func() {
		for {
			session, err := rs.Recv()
			if err != nil {
				select {
				case streamErr <- err:
				default:
				}
				return
			}
			go func(session *tpb.Session) {
				if err := s.newClientSession(ctx, session, p.Addr, rs); err != nil {
					select {
					case streamErr <- err:
					default:
					}
				}
			}(session)
		}
	}()
	return <-streamErr
}

func (s *Server) newClientSession(ctx context.Context, session *tpb.Session, addr net.Addr, rs regStream) error {
	tag := session.GetTag()
	if session.GetError() != "" {
		if ch := s.connection(tag, addr); ch != nil {
			ch <- ioOrErr{err: errors.New(session.GetError())}
			return nil
		}
		return fmt.Errorf("no connection associated with tag: %v", tag)
	}

	targetID := session.GetTargetId()
	if err := s.sc.RegisterHandler(ServerSession{addr, targetID}); err != nil {
		if err := rs.Send(&tpb.Session{Tag: tag, Error: err.Error()}); err != nil {
			return fmt.Errorf("failed to send session error: %v", err)
		}
		return nil
	}

	retCh := make(chan ioOrErr)
	if err := s.addConnection(tag, addr, retCh); err != nil {
		if err := rs.Send(&tpb.Session{Tag: tag, Error: err.Error()}); err != nil {
			return fmt.Errorf("failed to send session error: %v", err)
		}
		return nil
	}

	if err := rs.Send(&tpb.Session{Tag: session.Tag, Accept: true, TargetId: targetID}); err != nil {
		return fmt.Errorf("failed to send session ack: %v", err)
	}
	// ctx is a child of the register stream's context
	select {
	case <-ctx.Done():
		return ctx.Err()
	case ioe := <-retCh:
		if ioe.err != nil {
			log.Printf("error from return stream: %v", ioe.err)
			return nil
		}
		if err := s.sc.Handler(ServerSession{addr, targetID}, ioe.rwc); err != nil {
			log.Printf("error from handle: %v", err)
		}
		return nil
	}
}

// Tunnel accepts tunnel connections from the client. When it accepts a stream,
// it checks for the stream in the connections map. If it's there, it forwards
// the stream over the IOStream channel.
func (s *Server) Tunnel(stream tpb.Tunnel_TunnelServer) error {
	data, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to get tag from stream: %v", err)
	}
	if data.GetData() != nil {
		return errors.New("received data but only wanted tag")
	}
	tag := data.GetTag()
	p, ok := peer.FromContext(stream.Context())
	if !ok {
		return errors.New("no peer from stream context")
	}
	ch := s.connection(tag, p.Addr)
	if ch == nil {
		return errors.New("no connection associated with tag")
	}
	d := newIOStream(stream.Context(), stream)
	ch <- ioOrErr{rwc: d}
	// doneCh is the done channel created from a child context of stream.Context()
	<-d.doneCh
	return nil
}

// NewSession requests a new stream identified on the client side by uniqueID.
func (s *Server) NewSession(ctx context.Context, ss ServerSession) (io.ReadWriteCloser, error) {
	if len(s.clients) == 0 {
		return nil, errors.New("no clients connected")
	}
	tag := s.nextTag()
	// If ss.Addr is specified, the NewSession request will attempt to create a
	// new stream to an existing client.
	if ss.Addr != nil {
		if stream := s.client(ss.Addr); stream != nil {
			return s.handleSession(ctx, tag, ss.Addr, ss.TargetID, stream)
		}
		return nil, fmt.Errorf("no stream defined for %q", ss.Addr.String())
	}
	// If ss.Addr is not specified, the server will send a message to all clients.
	// The first client which responds without error will be the one to handle
	// the connection.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ch := make(chan io.ReadWriteCloser, len(s.clients))
	errCh := make(chan error, len(s.clients))
	var wg sync.WaitGroup
	// This lock protects only the read of s.clients, and unlocks after the loop.
	s.cmu.RLock()
	for addr, stream := range s.clients {
		wg.Add(1)
		go func(addr net.Addr, stream regStream) {
			defer wg.Done()
			rwc, err := s.handleSession(ctx, tag, addr, ss.TargetID, stream)
			if err != nil {
				errCh <- err
				return
			}
			cancel()
			ch <- rwc
		}(addr, stream)
	}
	s.cmu.RUnlock()

	wg.Wait()
	select {
	case rwc := <-ch:
		return rwc, nil
	default:
	}
	return nil, <-errCh
}

func (s *Server) handleSession(ctx context.Context, tag int32, addr net.Addr, targetID string, stream regStream) (_ io.ReadWriteCloser, err error) {
	if !s.peerHandler(addr) {
		return nil, fmt.Errorf("no handler defined on %q", addr.String())
	}
	retCh := make(chan ioOrErr)
	if err = s.addConnection(tag, addr, retCh); err != nil {
		return nil, fmt.Errorf("handleSession: failed to add connection: %v", err)
	}
	defer func() {
		if err != nil {
			s.deleteConnection(tag, addr)
		}
	}()
	if err = stream.Send(&tpb.Session{Tag: tag, Accept: true, TargetId: targetID}); err != nil {
		return nil, fmt.Errorf("handleSession: failed to send session: %v", err)
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case ioe := <-retCh:
		if ioe.err != nil {
			return nil, ioe.err
		}
		return ioe.rwc, nil
	}
}

// ClientRegHandlerFunc defines the targetIDs that the handler function can accept.
type ClientRegHandlerFunc func(targetID string) error

// ClientHandlerFunc handles sessions the client receives from the server.
type ClientHandlerFunc func(targetID string, rwc io.ReadWriteCloser) error

// ClientConfig contains the config for the client.
type ClientConfig struct {
	RegisterHandler ClientRegHandlerFunc
	Handler         ClientHandlerFunc
	Opts            []grpc.CallOption
}

// Client implementation of an endpoint.
type Client struct {
	endpoint

	block       chan struct{}
	tc          tpb.TunnelClient
	cc          ClientConfig
	peerHandler bool

	cmu  sync.RWMutex
	rs   *regSafeStream
	addr net.Addr // peer address to use in endpoint map
}

// NewClient creates a new tunnel client.
func NewClient(tc tpb.TunnelClient, cc ClientConfig) (*Client, error) {
	if (cc.RegisterHandler == nil) != (cc.Handler == nil) {
		return nil, errors.New("tunnel: can't create client: only 1 handler set")
	}
	return &Client{
		block: make(chan struct{}, 1),
		tc:    tc,
		cc:    cc,
		endpoint: endpoint{
			tag:       -1,
			conns:     make(map[session]chan ioOrErr),
			increment: -1,
		},
	}, nil
}

// NewSession requests a new stream identified on the server side
// by targetID.
func (c *Client) NewSession(targetID string) (_ io.ReadWriteCloser, err error) {
	c.cmu.RLock()
	defer c.cmu.RUnlock()
	if c.addr == nil {
		return nil, errors.New("client not started")
	}
	retCh := make(chan ioOrErr, 1)
	tag := c.nextTag()
	if err = c.addConnection(tag, c.addr, retCh); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			c.deleteConnection(tag, c.addr)
		}
	}()
	if err = c.rs.Send(&tpb.Session{Tag: tag, TargetId: targetID}); err != nil {
		return nil, err
	}
	ioe := <-retCh
	if ioe.err != nil {
		return nil, ioe.err
	}
	return ioe.rwc, nil
}

// Run initializes the client register stream and determines the capabilities
// of the tunnel server. Once done, it starts the stream handler responsible
// for determining what to do with received requests.
func (c *Client) Run(ctx context.Context) error {
	select {
	case c.block <- struct{}{}:
		defer func() {
			<-c.block
		}()
		// wrap critical section in a function to allow use of defer
		setup := func() error {
			c.cmu.Lock()
			defer c.cmu.Unlock()
			stream, err := c.tc.Register(ctx, c.cc.Opts...)
			if err != nil {
				return fmt.Errorf("start: failed to create register stream: %v", err)
			}
			c.rs = &regSafeStream{regStream: stream}
			p, ok := peer.FromContext(stream.Context())
			if !ok {
				return errors.New("no peer from stream context")
			}
			c.addr = p.Addr
			if err = c.capabilities(c.addr, c.rs); err != nil {
				return fmt.Errorf("error from capabilities: %v", err)
			}
			return nil
		}

		if err := setup(); err != nil {
			return err
		}
		return c.start(ctx)
	default:
		return errors.New("client is already running")
	}
}

// client capabilities determines whether a handler is defined locally, and then
// calls capabilities to get the peer handler status. This is the first set of
// messages that will be sent on the registration stream.
func (c *Client) capabilities(addr net.Addr, rs regStream) error {
	lh := (c.cc.Handler != nil) && (c.cc.RegisterHandler != nil)
	ph, err := capabilities(lh, rs)
	if err != nil {
		return err
	}
	if ph {
		c.peerHandler = true
	}
	return nil
}

// start handles received register stream requests.
func (c *Client) start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		for {
			session, err := c.rs.Recv()
			if err != nil {
				select {
				case errCh <- err:
				default:
				}
				return
			}
			if !session.GetAccept() {
				select {
				case errCh <- fmt.Errorf("connection %d not accepted by server", session.GetTag()):
				default:
				}
				return
			}
			tag := session.Tag
			tID := session.GetTargetId()
			go func() {
				if err := c.streamHandler(ctx, tag, tID); err != nil {
					select {
					case errCh <- err:
					default:
					}
				}
			}()
		}
	}()
	return <-errCh
}

func (c *Client) streamHandler(ctx context.Context, tag int32, tID string) error {
	var err error
	defer func() {
		// notify the server of the failure
		if err != nil {
			if err := c.rs.Send(&tpb.Session{Tag: tag, Error: err.Error()}); err != nil {
				log.Printf("failed to send session error: %v", err)
			}
		}
	}()
	// When tag is < 0, it means that the session request originated at this
	// client and NewSession is likely waiting for a returned connection.
	if tag < 0 {
		if err = c.returnedStream(ctx, tag); err != nil {
			return fmt.Errorf("returnStream: error from returnedStream: %v", err)
		}
		return nil
	}
	// Otherwise we attempt to handle the new target ID.
	if err = c.cc.RegisterHandler(tID); err != nil {
		return fmt.Errorf("returnStream: error from RegisterHandler: %v", err)
	}
	if err = c.newClientStream(ctx, tag, tID); err != nil {
		return fmt.Errorf("returnStream: error from handleNewClientStream: %v", err)
	}
	return nil
}

// returnedStream is called when the client receives a tag that is less than 0.
// A tag which is less than 0 indicates the stream originated at the client.
func (c *Client) returnedStream(ctx context.Context, tag int32) (err error) {
	c.cmu.RLock()
	defer c.cmu.RUnlock()
	ch := c.connection(tag, c.addr)
	if ch == nil {
		return fmt.Errorf("No connection associated with tag: %d", tag)
	}
	// notify client session of error, and return error to be sent to server.
	var rwc io.ReadWriteCloser
	defer func() {
		ch <- ioOrErr{rwc: rwc, err: err}
	}()
	rwc, err = c.newTunnelStream(ctx, tag)
	if err != nil {
		return err
	}
	return nil
}

// handleNewClientStream is called when the tag is greater than 0, and the client
// has a handler which can handle the id.
func (c *Client) newClientStream(ctx context.Context, tag int32, id string) error {
	stream, err := c.newTunnelStream(ctx, tag)
	if err != nil {
		return err
	}
	return c.cc.Handler(id, stream)
}

// newTunnelStream creates a new tunnel stream and sends tag to the server. The
// server uses this tag to uniquely identify the connection.
func (c *Client) newTunnelStream(ctx context.Context, tag int32) (*ioStream, error) {
	ts, err := c.tc.Tunnel(ctx, c.cc.Opts...)
	if err != nil {
		return nil, err
	}
	if err = ts.Send(&tpb.Data{Tag: tag}); err != nil {
		return nil, err
	}
	return newIOStream(ctx, ts), nil
}

// capabilities sends and receives capabilities between the endpoints and
// returns whether or not the peer handler has been set or an error.
func capabilities(lh bool, rs regStream) (bool, error) {
	if err := rs.Send(&tpb.Session{
		Capabilities: &tpb.Capabilities{
			Handler: lh,
		},
	}); err != nil {
		return false, fmt.Errorf("failed to send capabilities: %v", err)
	}
	s, err := rs.Recv()
	if err != nil {
		return false, err
	}
	ph := s.GetCapabilities().GetHandler()
	if !ph && !lh {
		return false, errors.New("no handlers defined")
	}
	return ph, nil
}
