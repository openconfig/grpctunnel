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
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"

	"github.com/openconfig/grpctunnel/bidi"
	tpb "github.com/openconfig/grpctunnel/proto/tunnel"
)

// regStream abstracts the tunnel Register streams.
type regStream interface {
	Recv() (*tpb.RegisterOp, error)
	Send(*tpb.RegisterOp) error
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
func (r *regSafeStream) Send(data *tpb.RegisterOp) error {
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
// It is only called when the server accepts new session from the client.
type ServerRegHandlerFunc func(ss ServerSession) error

// ServerHandlerFunc handles sessions the server receives from the client.
type ServerHandlerFunc func(ss ServerSession, rwc io.ReadWriteCloser) error

// ServerAddTargHandlerFunc is called for each target registered by a client. It
// will be called for target additions.
type ServerAddTargHandlerFunc func(t Target) error

// ServerDeleteTargHandlerFunc is called for each target registered by a client. It
// will be called for target additions.
type ServerDeleteTargHandlerFunc func(t Target) error

// ServerConfig contains the config for the server.
type ServerConfig struct {
	AddTargetHandler    ServerAddTargHandlerFunc
	DeleteTargetHandler ServerDeleteTargHandlerFunc
	RegisterHandler     ServerRegHandlerFunc
	Handler             ServerHandlerFunc
	LocalTargets        []Target
}

func (s *Server) bridgeRegHandler(ss ServerSession) error {
	addr := s.clientFromTarget(ss.Target)
	if addr == nil {
		return fmt.Errorf("failed to call bridgeRegHandler: target (%s: %s) not registered", ss.Target.ID, ss.Target.Type)
	}
	if addr == ss.Addr {
		return fmt.Errorf("failed to call bridgeRegHandler: address (%s) of the target is the same as the session", addr)
	}
	return nil
}

func (s *Server) bridgeHandler(ss ServerSession, rwc io.ReadWriteCloser) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addr := s.clientFromTarget(ss.Target)
	session, err := s.NewSession(ctx, ServerSession{Target: ss.Target, Addr: addr})
	if err != nil {
		return fmt.Errorf("error creating new session: %v", err)
	}

	if err = bidi.Copy(session, rwc); err != nil {
		return fmt.Errorf("error from bidi copy: %v", err)
	}

	return nil
}

// ServerSession is used by NewSession and the register handler. In the register
// handler it is used by the client to indicate what it's trying to connect to.
// In NewSession, ServerSession will indicate what target to connect to, as well
// as potentially specifying the client to connect to.
type ServerSession struct {
	Addr   net.Addr
	Target Target
}

// Target consists id and type, used to represent a tunnel target.
type Target struct {
	ID   string
	Type string
}

// clientRegInfo contains registration information on the server side.
type clientRegInfo struct {
	targets map[Target]struct{}
	rs      regStream
}

func (i clientRegInfo) IsZero() bool {
	return i.rs == nil && i.targets == nil
}

// Server is the server implementation of an endpoint.
type Server struct {
	tpb.UnimplementedTunnelServer
	endpoint

	sc ServerConfig

	cmu     sync.RWMutex
	clients map[net.Addr]clientRegInfo

	tmu      sync.RWMutex
	rTargets map[Target]net.Addr // remote targets
	lTargets map[Target]struct{} // local targets

	smu sync.RWMutex
	sub map[net.Addr]map[string]struct{} // subscription information

	errCh chan error // for logging server errors without disruptting the tunnel
}

// NewServer creates a new tunnel server.
func NewServer(sc ServerConfig) (*Server, error) {
	if (sc.RegisterHandler == nil) != (sc.Handler == nil) {
		return nil, errors.New("tunnel: can't create server: only 1 handler set")
	}
	lTargets := make(map[Target]struct{})
	for _, t := range sc.LocalTargets {
		lTargets[t] = struct{}{}
	}
	return &Server{
		clients:  make(map[net.Addr]clientRegInfo),
		rTargets: make(map[Target]net.Addr),
		lTargets: lTargets,
		sc:       sc,
		endpoint: endpoint{
			tag:       1,
			conns:     make(map[session]chan ioOrErr),
			increment: 1,
		},
		sub:   map[net.Addr]map[string]struct{}{},
		errCh: make(chan error),
	}, nil
}

// clientInfo returns a collection of registration related information.
func (s *Server) clientInfo(addr net.Addr) clientRegInfo {
	s.cmu.RLock()
	defer s.cmu.RUnlock()
	return s.clients[addr]
}

// clientTargets returns all the targets of a given client. If addr is nil, return all the targets.
func (s *Server) clientTargets(addr net.Addr) map[Target]struct{} {
	s.cmu.RLock()
	defer s.cmu.RUnlock()
	// Make a deep copy.
	targets := make(map[Target]struct{})

	if addr == nil {
		for t := range s.rTargets {
			targets[t] = struct{}{}
		}
		return targets
	}

	info, ok := s.clients[addr]
	if !ok {
		return nil
	}

	for t := range info.targets {
		targets[t] = struct{}{}
	}
	return targets
}

func (s *Server) clientFromTarget(t Target) net.Addr {
	s.tmu.RLock()
	defer s.tmu.RUnlock()

	return s.rTargets[t]
}

// addClient adds a client to the clients map.
func (s *Server) addClient(addr net.Addr, rs regStream) error {
	s.cmu.Lock()
	defer s.cmu.Unlock()
	if _, ok := s.clients[addr]; ok {
		return fmt.Errorf("client exists for %q", addr.String())
	}
	s.clients[addr] = clientRegInfo{rs: rs, targets: make(map[Target]struct{})}

	return nil
}

// deleteClient removes a client from the clients map, and delete the corresponding targets from the targets map.
func (s *Server) deleteClient(addr net.Addr) {
	s.cmu.Lock()
	defer s.cmu.Unlock()
	clientInfo, ok := s.clients[addr]
	if !ok {
		return
	}

	for t := range clientInfo.targets {
		target := tpb.Target{TargetId: t.ID, TargetType: t.Type}
		s.deleteTarget(addr, &target, false)
	}

	delete(s.clients, addr)
}

// errorTargetRegisterOp returns a RegisterOp message of the form Registration with error.
func errorTargetRegisterOp(id, typ, err string) *tpb.RegisterOp {
	return &tpb.RegisterOp{Registration: &tpb.RegisterOp_Target{Target: &tpb.Target{TargetId: id, TargetType: typ, Error: err}}}
}

// addTargetToMap adds a target to the targets map.
func (s *Server) addTargetToMap(addr net.Addr, t Target) error {
	s.tmu.Lock()
	defer s.tmu.Unlock()

	if c, ok := s.rTargets[t]; ok {
		return fmt.Errorf("target %q already registered for client %q", t.ID, c)
	}
	s.rTargets[t] = addr
	return nil
}

// deleteTargetFromMap deletes a target from the targets map.
func (s *Server) deleteTargetFromMap(t Target) error {
	s.tmu.Lock()
	defer s.tmu.Unlock()

	if c, ok := s.rTargets[t]; !ok {
		return fmt.Errorf("target %q is not registered for client %q", t.ID, c)
	}

	delete(s.rTargets, t)
	return nil
}

// addTargetToClient adds a target to the clients map.
func (s *Server) addTargetToClient(addr net.Addr, t Target) {
	s.cmu.Lock()
	defer s.cmu.Unlock()
	// Already checked its entry does exists.
	s.clients[addr].targets[t] = struct{}{}
}

// deleteTargetFromClient deletes a target to the clients map.
func (s *Server) deleteTargetFromClient(addr net.Addr, t Target) {
	s.cmu.Lock()
	defer s.cmu.Unlock()
	// Already checked its entry exists.
	delete(s.clients[addr].targets, t)
}

// addTarget registers a target for a given client. It registers
// is in the clients map and targets map.
func (s *Server) addTarget(addr net.Addr, target *tpb.Target) error {
	t := Target{ID: target.TargetId, Type: target.TargetType}
	clientInfo := s.clientInfo(addr)
	if clientInfo.IsZero() {
		return fmt.Errorf("client %q not registered", addr)
	}
	rs := clientInfo.rs

	if _, ok := s.lTargets[t]; ok {
		err := fmt.Errorf("target (%s, %s) clashes with server's local target", t.ID, t.Type)
		if err := rs.Send(errorTargetRegisterOp(target.TargetId, target.TargetType, err.Error())); err != nil {
			return fmt.Errorf("failed to send session error: %v", err)
		}
		return err
	}

	targets := s.clientTargets(addr)
	if _, ok := targets[t]; ok {
		err := fmt.Errorf("target %q already registered in s.clients", t)
		if err := rs.Send(errorTargetRegisterOp(target.TargetId, target.TargetType, err.Error())); err != nil {
			return fmt.Errorf("failed to send session error: %v", err)
		}
		return err
	}

	if err := s.addTargetToMap(addr, t); err != nil {
		if err := rs.Send(errorTargetRegisterOp(target.TargetId, target.TargetType, err.Error())); err != nil {
			return fmt.Errorf("failed to send session error: %v", err)
		}
		return err
	}

	s.addTargetToClient(addr, t)

	if err := rs.Send(&tpb.RegisterOp{Registration: &tpb.RegisterOp_Target{Target: &tpb.Target{
		TargetId:   target.TargetId,
		TargetType: target.TargetType,
		Accept:     true,
	}}}); err != nil {
		return fmt.Errorf("failed to send session ack: %v", err)
	}

	if s.sc.AddTargetHandler != nil {
		if err := s.sc.AddTargetHandler(t); err != nil {
			return fmt.Errorf("failed to cal addTargetHandler: %v", err)
		}
	}

	if err := s.sendUpdates(t, true); err != nil {
		return fmt.Errorf("failed to send target subscription updates: %v", err)
	}

	return nil
}

func (s *Server) handleSubscription(addr net.Addr, sub *tpb.Subscription) error {
	switch op := sub.GetOp(); op {
	case tpb.Subscription_SUBCRIBE:
		return s.subscribe(addr, sub)
	case tpb.Subscription_UNSUBCRIBE:
		return s.unsubscribe(addr, sub)
	default:
		return fmt.Errorf("invalid subcription op: %d", op)
	}
}

func (s *Server) addSubscription(addr net.Addr, typ string) error {
	s.smu.Lock()
	defer s.smu.Unlock()

	ts, ok := s.sub[addr]
	if !ok {
		s.sub[addr] = make(map[string]struct{})
	}
	if _, ok := ts[typ]; ok {
		return fmt.Errorf("%s is already subscribed by %s", typ, addr)
	}

	s.sub[addr][typ] = struct{}{}
	return nil
}

func (s *Server) subscribe(addr net.Addr, sub *tpb.Subscription) error {

	targets := s.clientTargets(nil)
	// Combine with local targets.
	for t := range s.lTargets {
		targets[t] = struct{}{}
	}
	allTypes := map[string]struct{}{}
	for t := range targets {
		allTypes[t.Type] = struct{}{}
	}

	for typ := range allTypes {
		if sub.TargetType != "" && sub.TargetType != typ {
			continue
		}

		if err := s.addSubscription(addr, typ); err != nil {
			clientInfo := s.clientInfo(addr)
			if clientInfo.IsZero() {
				return fmt.Errorf("client %q not registered", addr)
			}
			rs := clientInfo.rs

			if err := rs.Send(&tpb.RegisterOp{Registration: &tpb.RegisterOp_Subscription{Subscription: &tpb.Subscription{
				TargetType: sub.TargetType,
				Error:      err.Error(),
				Op:         sub.Op,
			}}}); err != nil {
				return fmt.Errorf("failed to send session error: %v", err)
			}
		}
	}

	// Send the initial updates.
	for t := range targets {
		if sub.TargetType == "" || sub.TargetType == t.Type {
			if err := s.sendUpdate(addr, t, true); err != nil {
				return fmt.Errorf("failed to send initial updates for subscription from %s: %v", addr, err)
			}
		}
	}

	// Send ack.
	clientInfo := s.clientInfo(addr)
	rs := clientInfo.rs
	if err := rs.Send(&tpb.RegisterOp{Registration: &tpb.RegisterOp_Subscription{
		Subscription: &tpb.Subscription{
			TargetType: sub.TargetType,
			Op:         sub.Op,
			Accept:     true}}}); err != nil {
		return fmt.Errorf("failed to send subscription ack for %s: %v", addr, err)
	}

	return nil
}

func (s *Server) deleteSubscriber(addr net.Addr, typ string) error {
	s.smu.Lock()
	defer s.smu.Unlock()

	if _, ok := s.sub[addr]; !ok {
		return fmt.Errorf("client %s is not in subscription list", addr)
	}

	if typ == "" {
		delete(s.sub, addr)
		return nil
	}

	delete(s.sub[addr], typ)
	return nil
}

func (s *Server) unsubscribe(addr net.Addr, sub *tpb.Subscription) error {
	clientInfo := s.clientInfo(addr)
	if clientInfo.IsZero() {
		return fmt.Errorf("client %s is not registered", addr)
	}
	rs := clientInfo.rs

	if err := s.deleteSubscriber(addr, sub.TargetType); err != nil {
		if e := rs.Send(&tpb.RegisterOp{Registration: &tpb.RegisterOp_Subscription{
			Subscription: &tpb.Subscription{
				TargetType: sub.TargetType,
				Op:         sub.Op,
				Accept:     true}}}); e != nil {
			return fmt.Errorf("failed to send unsubscription ack for %s: %v", addr, err)
		}
		return err
	}

	if err := rs.Send(&tpb.RegisterOp{Registration: &tpb.RegisterOp_Subscription{
		Subscription: &tpb.Subscription{
			TargetType: sub.TargetType,
			Op:         sub.Op,
			Accept:     true}}}); err != nil {
		return fmt.Errorf("failed to send unsubscription ack for %s: %v", addr, err)
	}
	return nil
}

func (s *Server) subscribers(typ string) map[net.Addr]struct{} {
	s.smu.RLock()
	defer s.smu.RUnlock()
	sbs := make(map[net.Addr]struct{})

	for a, ts := range s.sub {
		if typ == "" {
			sbs[a] = struct{}{}
		} else if _, ok := ts[typ]; ok {
			sbs[a] = struct{}{}
		}
	}
	return sbs
}

func (s *Server) sendUpdate(addr net.Addr, target Target, add bool) error {
	op := tpb.Target_ADD
	if !add {
		op = tpb.Target_REMOVE
	}

	clientInfo := s.clientInfo(addr)
	rs := clientInfo.rs
	if err := rs.Send(&tpb.RegisterOp{Registration: &tpb.RegisterOp_Target{Target: &tpb.Target{
		TargetId:   target.ID,
		TargetType: target.Type,
		Op:         op,
	}}}); err != nil {
		return fmt.Errorf("failed to send subscription update for %s: %v", addr, err)
	}

	return nil
}

func (s *Server) sendUpdates(target Target, add bool) error {
	addrs := s.subscribers(target.Type)

	if addrs == nil {
		return nil
	}

	var e error
	for addr := range addrs {
		if err := s.sendUpdate(addr, target, add); err != nil {
			if e != nil {
				e = err
			} else {
				e = fmt.Errorf("%s\n%s", err, e)
			}
		}
	}
	return e
}

// deleteTarget unregisters a target for a given client. It unregisters
// it from the clients map and targets map. Optionally, it can send an ack via regStream.
func (s *Server) deleteTarget(addr net.Addr, target *tpb.Target, ack bool) error {

	clientInfo := s.clientInfo(addr)

	if clientInfo.IsZero() {
		return fmt.Errorf("client %q not registered", addr)
	}
	rs := clientInfo.rs
	t := Target{ID: target.TargetId, Type: target.TargetType}

	if _, ok := clientInfo.targets[t]; !ok {
		err := fmt.Errorf("target %q is not registered in s.clients", t)
		if err := rs.Send(errorTargetRegisterOp(target.TargetId, target.TargetType, err.Error())); err != nil {
			return fmt.Errorf("failed to send session error: %v", err)
		}
		return err
	}

	if err := s.deleteTargetFromMap(t); err != nil {
		if ack {
			if err := rs.Send(errorTargetRegisterOp(target.TargetId, target.TargetType, err.Error())); err != nil {
				return fmt.Errorf("failed to send session error: %v", err)
			}
		}
		return err
	}

	s.deleteTargetFromClient(addr, t)
	if ack {
		if err := rs.Send(&tpb.RegisterOp{Registration: &tpb.RegisterOp_Target{Target: &tpb.Target{
			TargetId:   target.TargetId,
			TargetType: target.TargetType,
			Accept:     true,
		}}}); err != nil {
			return fmt.Errorf("failed to send session ack: %v", err)
		}
	}

	if s.sc.DeleteTargetHandler != nil {
		if err := s.sc.DeleteTargetHandler(t); err != nil {
			return fmt.Errorf("error calling target deletion handler client: %v", err)
		}
	}
	if err := s.deleteSubscriber(addr, ""); err != nil {
		return fmt.Errorf("failed to delete %q from subscription list: %v", target.TargetId, err)
	}
	if err := s.sendUpdates(t, false); err != nil {
		return fmt.Errorf("failed to send target subscription updates: %v", err)
	}

	return nil
}

// handleTarget handles target registration. It supports addition and removal.
func (s *Server) handleTarget(addr net.Addr, target *tpb.Target) error {
	switch op := target.GetOp(); op {
	case tpb.Target_ADD:
		return s.addTarget(addr, target)
	case tpb.Target_REMOVE:
		return s.deleteTarget(addr, target, true)
	default:
		return fmt.Errorf("invalid target op: %d", op)
	}
}

// deleteTargets unregisters all targets of a given client.
func (s *Server) deleteTargets(addr net.Addr, ack bool) error {
	if clientInfo := s.clientInfo(addr); clientInfo.IsZero() {
		e := fmt.Errorf("client %q not registered", addr)
		return e
	}

	for target := range s.clientTargets(addr) {
		t := tpb.Target{TargetId: target.ID, TargetType: target.Type}
		if err := s.deleteTarget(addr, &t, ack); err != nil {
			return err
		}
	}
	return nil
}

// sendError receives error from Register in a non-blocking way.
func (s *Server) sendError(err error) {
	select {
	case s.errCh <- err:
	default: // default do nothing, so it won't be blocking
	}
}

// ErrorChan returns a channel that sends errors from the operations such as Register.
func (s *Server) ErrorChan() <-chan error {
	return s.errCh
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

	if err := s.addClient(p.Addr, rs); err != nil {
		return fmt.Errorf("error adding client: %v", err)
	}
	defer s.deleteClient(p.Addr)
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()
	defer s.deleteTargets(p.Addr, false)

	// The loop will handle target registration and new sessions based on the
	// registration stream type. It will ignore errors and only return if
	// Recv from regstream failed.
	for {
		reg, err := rs.Recv()
		if err != nil {
			return err
		}
		switch reg.Registration.(type) {
		case *tpb.RegisterOp_Session:
			s.newClientSession(ctx, reg.GetSession(), p.Addr, rs)
		case *tpb.RegisterOp_Target:
			if err := s.handleTarget(p.Addr, reg.GetTarget()); err != nil {
				s.sendError(fmt.Errorf("failed to handle target resigtration: %v", err))
			}
		case *tpb.RegisterOp_Subscription:
			if err := s.handleSubscription(p.Addr, reg.GetSubscription()); err != nil {
				s.sendError(fmt.Errorf("failed to handle subscription resigtration: %v", err))
			}
		default:
			s.sendError(fmt.Errorf("unknown registration op from %s: %s", p.Addr, reg.Registration))
		}
	}

}

func (s *Server) newClientSession(ctx context.Context, session *tpb.Session, addr net.Addr, rs regStream) {
	tag := session.GetTag()
	if session.GetError() != "" {
		if ch := s.connection(tag, addr); ch != nil {
			ch <- ioOrErr{err: errors.New(session.GetError())}
			return
		}
		s.sendError(fmt.Errorf("no connection associated with tag: %v", tag))
		return
	}

	var err error
	t := Target{ID: session.TargetId, Type: session.TargetType}
	// If client is requesting a remote target, only call the bridge register handler.
	// We might extend it to allow calling the customized handler in the future.
	if _, ok := s.rTargets[t]; ok {
		err = s.bridgeRegHandler(ServerSession{addr, t})
	} else {
		err = s.sc.RegisterHandler(ServerSession{addr, t})
	}
	if err != nil {
		if err := rs.Send(&tpb.RegisterOp{Registration: &tpb.RegisterOp_Session{Session: &tpb.Session{Tag: tag, Error: err.Error()}}}); err != nil {
			s.sendError(fmt.Errorf("failed to send session error: %v", err))
			return
		}
		return
	}

	retCh := make(chan ioOrErr)
	if err := s.addConnection(tag, addr, retCh); err != nil {
		if err := rs.Send(&tpb.RegisterOp{Registration: &tpb.RegisterOp_Session{Session: &tpb.Session{Tag: tag, Error: err.Error()}}}); err != nil {
			s.sendError(fmt.Errorf("failed to send session error: %v", err))
			return
		}
		return
	}

	if err := rs.Send(&tpb.RegisterOp{Registration: &tpb.RegisterOp_Session{Session: &tpb.Session{
		Tag:        session.Tag,
		Accept:     true,
		TargetId:   session.TargetId,
		TargetType: session.TargetType,
	}}}); err != nil {
		s.sendError(fmt.Errorf("failed to send session ack: %v", err))
		return
	}
	// ctx is a child of the register stream's context
	select {
	case <-ctx.Done():
		s.sendError(ctx.Err())
		return
	case ioe := <-retCh:
		if ioe.err != nil {
			return
		}
		go func() {
			// If client is requesting a remote target, only call the bridge handler.
			// We might extend it to allow calling the customized handler in the future.
			var err error
			if _, ok := s.rTargets[t]; ok {
				err = s.bridgeHandler(ServerSession{addr, t}, ioe.rwc)
			} else {
				err = s.sc.Handler(ServerSession{addr, t}, ioe.rwc)
			}
			if err != nil {
				s.sendError(err)
			}
			return
		}()
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

// NewTarget sends a target addition registration via regStream.
func (c *Client) NewTarget(target Target) error {
	return c.rs.Send(&tpb.RegisterOp{Registration: &tpb.RegisterOp_Target{Target: &tpb.Target{
		TargetId:   target.ID,
		Op:         tpb.Target_ADD,
		TargetType: target.Type,
	}}})
}

// DeleteTarget sends a target deletion registration via regStream.
func (c *Client) DeleteTarget(target Target) error {
	return c.rs.Send(&tpb.RegisterOp{Registration: &tpb.RegisterOp_Target{Target: &tpb.Target{
		TargetId:   target.ID,
		Op:         tpb.Target_REMOVE,
		TargetType: target.Type,
	}}})
}

// Subscribe send a subscription request to the server for targets updates.
func (c *Client) Subscribe(typ string) error {
	c.pmu.Lock()
	defer c.pmu.Unlock()

	if _, ok := c.peerTypeTargets[typ]; !ok {
		c.peerTypeTargets[typ] = make(map[Target]struct{})
	}

	return c.rs.Send(&tpb.RegisterOp{Registration: &tpb.RegisterOp_Subscription{Subscription: &tpb.Subscription{
		Op:         tpb.Subscription_SUBCRIBE,
		TargetType: typ,
	}}})
}

// Unsubscribe send a unsubscription request to the server for stopping receiving targets updates.
func (c *Client) Unsubscribe(typ string) error {
	c.pmu.Lock()
	defer c.pmu.Unlock()

	if _, ok := c.peerTypeTargets[typ]; !ok {
		return fmt.Errorf("target type %s is not subscribed yet", typ)
	}

	return c.rs.Send(&tpb.RegisterOp{Registration: &tpb.RegisterOp_Subscription{Subscription: &tpb.Subscription{
		Op:         tpb.Subscription_UNSUBCRIBE,
		TargetType: typ,
	}}})
}

func (c *Client) addPeerTarget(t *tpb.Target) error {
	c.pmu.Lock()
	defer c.pmu.Unlock()

	// Skip itself.
	if _, ok := c.targets[Target{ID: t.TargetId, Type: t.TargetType}]; ok {
		return nil
	}

	if _, ok := c.peerTypeTargets[t.TargetType]; !ok {
		c.peerTypeTargets[t.TargetType] = make(map[Target]struct{})
	}

	c.peerTypeTargets[t.TargetType][Target{ID: t.TargetId, Type: t.TargetType}] = struct{}{}

	return c.cc.PeerAddHandler(Target{ID: t.TargetId, Type: t.TargetType})
}

// PeerTargets returns all the peer targets matching the given type. If type is an empty string, it will return all targets.
func (c *Client) PeerTargets(typ string) map[Target]struct{} {
	c.pmu.Lock()
	defer c.pmu.Unlock()

	targets := make(map[Target]struct{})
	for tp, ts := range c.peerTypeTargets {
		if tp == typ || typ == "" {
			for t := range ts {
				targets[t] = struct{}{}
			}
		}
	}
	return targets
}

func (c *Client) deletePeerTarget(t *tpb.Target) error {
	c.pmu.Lock()
	defer c.pmu.Unlock()

	if _, ok := c.peerTypeTargets[t.TargetType]; !ok {
		delete(c.peerTypeTargets[t.TargetType], Target{ID: t.TargetId, Type: t.TargetType})
	}
	return c.cc.PeerDelHandler(Target{ID: t.TargetId, Type: t.TargetType})
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
		regInfo := s.clientInfo(ss.Addr)
		if !regInfo.IsZero() && regInfo.rs != nil {
			return s.handleSession(ctx, tag, ss.Addr, ss.Target, regInfo.rs)
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
	for addr, clientInfo := range s.clients {
		wg.Add(1)
		go func(addr net.Addr, stream regStream) {
			defer wg.Done()
			rwc, err := s.handleSession(ctx, tag, addr, ss.Target, stream)
			if err != nil {
				errCh <- err
				return
			}
			cancel()
			ch <- rwc
		}(addr, clientInfo.rs)
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

func (s *Server) handleSession(ctx context.Context, tag int32, addr net.Addr, target Target, stream regStream) (_ io.ReadWriteCloser, err error) {
	retCh := make(chan ioOrErr)
	if err = s.addConnection(tag, addr, retCh); err != nil {
		return nil, fmt.Errorf("handleSession: failed to add connection: %v", err)
	}
	defer func() {
		if err != nil {
			s.deleteConnection(tag, addr)
		}
	}()

	if err = stream.Send(&tpb.RegisterOp{Registration: &tpb.RegisterOp_Session{Session: &tpb.Session{
		Tag:        tag,
		Accept:     true,
		TargetId:   target.ID,
		TargetType: target.Type,
	}}}); err != nil {
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

// ClientPeerAddHandlerFunc is called when a peer target is registered.
type ClientPeerAddHandlerFunc func(target Target) error

// ClientPeerDelHandlerFunc is called when a peer target is deleted.
type ClientPeerDelHandlerFunc func(target Target) error

// ClientRegHandlerFunc defines the targetIDs that the handler function can accept.
type ClientRegHandlerFunc func(target Target) error

// ClientHandlerFunc handles sessions the client receives from the server.
type ClientHandlerFunc func(target Target, rwc io.ReadWriteCloser) error

// ClientConfig contains the config for the client.
type ClientConfig struct {
	RegisterHandler ClientRegHandlerFunc
	Handler         ClientHandlerFunc
	PeerAddHandler  ClientPeerAddHandlerFunc
	PeerDelHandler  ClientPeerDelHandlerFunc
	Opts            []grpc.CallOption
	Subscriptions   []string
}

// Client implementation of an endpoint.
type Client struct {
	endpoint

	block chan struct{}
	tc    tpb.TunnelClient
	cc    ClientConfig

	cmu  sync.RWMutex
	rs   *regSafeStream
	addr net.Addr // peer address to use in endpoint map

	targets map[Target]struct{}

	pmu             sync.RWMutex
	peerTypeTargets map[string]map[Target]struct{}

	err        error
	emu        sync.RWMutex
	cancelFunc func()
}

// cancel performs cancellations of Start() and streamHandler(), and records error.
func (c *Client) cancel(err error) {
	c.emu.Lock()
	defer c.emu.Unlock()
	// Avoid calling multiple times.
	if c.cancelFunc == nil {
		return
	}

	c.cancelFunc()
	c.cancelFunc = nil
	c.err = err
}

// Error returns the error collected from streamHandler.
func (c *Client) Error() error {
	c.emu.RLock()
	defer c.emu.RUnlock()

	return c.err
}

// NewClient creates a new tunnel client.
func NewClient(tc tpb.TunnelClient, cc ClientConfig, ts map[Target]struct{}) (*Client, error) {
	if (cc.RegisterHandler == nil) != (cc.Handler == nil) {
		return nil, errors.New("tunnel: can't create client: only 1 handler set")
	}

	peerTypeTargets := make(map[string]map[Target]struct{})
	for _, typ := range cc.Subscriptions {
		peerTypeTargets[typ] = make(map[Target]struct{})
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
		targets:         ts,
		peerTypeTargets: peerTypeTargets,
	}, nil
}

// NewSession requests a new stream identified on the server side
// by targetID.
func (c *Client) NewSession(target Target) (_ io.ReadWriteCloser, err error) {
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
	if err = c.rs.Send(&tpb.RegisterOp{Registration: &tpb.RegisterOp_Session{Session: &tpb.Session{
		Tag:        tag,
		TargetId:   target.ID,
		TargetType: target.Type,
	}}}); err != nil {
		return nil, err
	}
	ioe := <-retCh
	if ioe.err != nil {
		return nil, ioe.err
	}
	return ioe.rwc, nil
}

// Register initializes the client register stream and determines the
// capabilities of the tunnel server.
func (c *Client) Register(ctx context.Context) (err error) {
	c.cmu.Lock()
	defer c.cmu.Unlock()

	ctx, c.cancelFunc = context.WithCancel(ctx)
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

	for target := range c.targets {
		c.NewTarget(target)
	}

	for typ := range c.peerTypeTargets {
		c.Subscribe(typ)
	}

	return nil
}

// Run initializes the client register stream and determines the capabilities
// of the tunnel server. Once done, it starts the stream handler responsible
// for determining what to do with received requests.
func (c *Client) Run(ctx context.Context) error {
	if err := c.Register(ctx); err != nil {
		return err
	}
	c.Start(ctx)
	return c.Error()
}

// Start handles received register stream requests.
func (c *Client) Start(ctx context.Context) {
	var err error
	defer func() {
		c.cancel(err)
	}()

	select {
	case c.block <- struct{}{}:
		defer func() {
			<-c.block
		}()
	default:
		err = errors.New("client is already running")
		return
	}

	for {
		var reg *tpb.RegisterOp
		reg, err = c.rs.Recv()
		if err != nil {
			return
		}

		switch reg.Registration.(type) {
		case *tpb.RegisterOp_Session:
			session := reg.GetSession()
			if !session.GetAccept() {
				err = fmt.Errorf("connection %d not accepted by server", session.GetTag())
				return
			}
			tag := session.Tag
			tID := session.GetTargetId()
			tType := session.GetTargetType()
			go func() {
				if err := c.streamHandler(ctx, tag, Target{ID: tID, Type: tType}); err != nil {
					c.cancel(err)
				}
			}()
		case *tpb.RegisterOp_Target:
			target := reg.GetTarget()
			switch op := target.GetOp(); op {
			case tpb.Target_ADD:
				if e := c.addPeerTarget(target); e != nil {
					err = fmt.Errorf("failed to add peer target: %v", e)
					return
				}
			case tpb.Target_REMOVE:
				if e := c.deletePeerTarget(target); e != nil {
					err = fmt.Errorf("failed to delete peer target: %v", e)
					return
				}
			default:
				if !target.GetAccept() {
					err = fmt.Errorf("target registration (%s, %s) not accepted by server", target.TargetId, target.TargetType)
					return
				}
			}
		case *tpb.RegisterOp_Subscription:
			sub := reg.GetSubscription()
			op := sub.GetOp()
			if op == tpb.Subscription_SUBCRIBE || op == tpb.Subscription_UNSUBCRIBE {
				if !sub.GetAccept() {
					err = fmt.Errorf("subscription request (%s) not accepted by server", sub.TargetType)
					return
				}
			}
		}
	}
}

func (c *Client) streamHandler(ctx context.Context, tag int32, t Target) (e error) {
	var err error
	defer func() {
		// notify the server of the failure
		if err != nil {
			if err = c.rs.Send(&tpb.RegisterOp{Registration: &tpb.RegisterOp_Session{Session: &tpb.Session{Tag: tag, Error: err.Error()}}}); err != nil {
				e = fmt.Errorf("failed to send session error: %v. Original err: %v", err, e)
			}
		}
	}()
	// When tag is < 0, it means that the session request originated at this
	// client and NewSession is likely waiting for a returned connection.
	if tag < 0 {
		if err = c.returnedStream(ctx, tag); err != nil {
			e = fmt.Errorf("returnStream: error from returnedStream: %v", err)
			return
		}
		return nil
	}
	// Otherwise we attempt to handle the new target ID.
	if err = c.cc.RegisterHandler(t); err != nil {
		e = fmt.Errorf("returnStream: error from RegisterHandler: %v", err)
		return
	}
	if err = c.newClientStream(ctx, tag, t); err != nil {
		e = fmt.Errorf("returnStream: error from handleNewClientStream: %v", err)
		return
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
func (c *Client) newClientStream(ctx context.Context, tag int32, t Target) error {
	stream, err := c.newTunnelStream(ctx, tag)
	if err != nil {
		return err
	}
	return c.cc.Handler(t, stream)
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
