package tunnel

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	tpb "github.com/openconfig/grpctunnel/proto/tunnel"
)

// These tests run the tunnel server and clients over real gRPC streams on a
// loopback TCP connection, so a subscriber that stops reading its register
// stream exhausts its actual HTTP/2 flow-control window instead of a
// simulated one. Subscribers use the regular Client API (Subscriptions,
// PeerAddHandler and PeerDelHandler), as tunnel client applications do.

// grpcTarget returns target i of the gRPC tests. The 1 KB names make a few
// dozen target updates enough to fill a 64 KB flow-control window.
func grpcTarget(i int) Target {
	return Target{ID: fmt.Sprintf("target%03d-%s", i, strings.Repeat("x", 1024)), Type: "type1"}
}

// grpcTestServer is a tunnel server serving gRPC on a loopback port. It counts
// the targets it has registered and deleted.
type grpcTestServer struct {
	*Server
	addr string

	mu                  sync.Mutex
	registered, deleted int
}

func startGRPCTestServer(t *testing.T) *grpcTestServer {
	t.Helper()
	ts := &grpcTestServer{}
	s, err := NewServer(ServerConfig{
		AddTargetHandler: func(Target) error {
			ts.mu.Lock()
			defer ts.mu.Unlock()
			ts.registered++
			return nil
		},
		DeleteTargetHandler: func(Target) error {
			ts.mu.Lock()
			defer ts.mu.Unlock()
			ts.deleted++
			return nil
		},
	})
	if err != nil {
		t.Fatalf("NewServer() failed: %v", err)
	}
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() failed: %v", err)
	}
	gs := grpc.NewServer()
	tpb.RegisterTunnelServer(gs, s)
	go gs.Serve(lis)
	t.Cleanup(gs.Stop)
	ts.Server, ts.addr = s, lis.Addr().String()
	return ts
}

// counts returns how many targets the server has registered and deleted.
func (ts *grpcTestServer) counts() (registered, deleted int) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.registered, ts.deleted
}

// dial returns a tunnel service stub on a new connection to the server. The
// connection uses fixed 64 KB flow-control windows, the gRPC minimum, so that
// a client that stops reading stalls the server's writes deterministically.
func (ts *grpcTestServer) dial(t *testing.T) tpb.TunnelClient {
	t.Helper()
	conn, err := grpc.Dial(ts.addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithInitialWindowSize(1<<16),
		grpc.WithInitialConnWindowSize(1<<16))
	if err != nil {
		t.Fatalf("grpc.Dial(%s) failed: %v", ts.addr, err)
	}
	t.Cleanup(func() { conn.Close() })
	return tpb.NewTunnelClient(conn)
}

// startClient registers a tunnel client with config cc and runs its receive
// loop until the test ends.
func (ts *grpcTestServer) startClient(t *testing.T, cc ClientConfig) *Client {
	t.Helper()
	c, err := NewClient(ts.dial(t), cc, nil)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if err := c.Register(ctx); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}
	go c.Start(ctx)
	return c
}

// registerAnchor registers a first target of type1 through c and waits for
// the server to accept it, so that clients can subscribe to type1.
func (ts *grpcTestServer) registerAnchor(t *testing.T, c *Client) Target {
	t.Helper()
	anchor := Target{ID: "anchor", Type: "type1"}
	if err := c.NewTarget(anchor); err != nil {
		t.Fatalf("NewTarget(%v) failed: %v", anchor, err)
	}
	if !eventually(func() bool { r, _ := ts.counts(); return r == 1 }) {
		t.Fatal("server did not register the anchor target")
	}
	return anchor
}

// eventually reports whether cond holds within deadlockTimeout.
func eventually(cond func() bool) bool {
	for deadline := time.Now().Add(deadlockTimeout); !cond(); time.Sleep(time.Millisecond) {
		if time.Now().After(deadline) {
			return false
		}
	}
	return true
}

// peerView records the peer targets that a tunnel client learns through its
// PeerAddHandler and PeerDelHandler. While the view is paused the handlers
// block, so the client stops reading its register stream like a slow
// subscriber.
type peerView struct {
	mu      sync.Mutex
	targets map[Target]bool
	running chan struct{} // closed while the view is not paused
}

func newPeerView() *peerView {
	v := &peerView{targets: map[Target]bool{}, running: make(chan struct{})}
	close(v.running)
	return v
}

func (v *peerView) pause() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.running = make(chan struct{})
}

func (v *peerView) resume() {
	v.mu.Lock()
	defer v.mu.Unlock()
	select {
	case <-v.running:
	default:
		close(v.running)
	}
}

func (v *peerView) set(t Target, present bool) error {
	v.mu.Lock()
	running := v.running
	v.mu.Unlock()
	<-running
	v.mu.Lock()
	defer v.mu.Unlock()
	if present {
		v.targets[t] = true
	} else {
		delete(v.targets, t)
	}
	return nil
}

// config returns a client config that subscribes to type1 into the view.
func (v *peerView) config() ClientConfig {
	return ClientConfig{
		Subscriptions:  []string{"type1"},
		PeerAddHandler: func(t Target) error { return v.set(t, true) },
		PeerDelHandler: func(t Target) error { return v.set(t, false) },
	}
}

func (v *peerView) waitFor(t *testing.T, want map[Target]bool) {
	t.Helper()
	has := func() int {
		v.mu.Lock()
		defer v.mu.Unlock()
		if sameTargets(v.targets, want) {
			return -1
		}
		return len(v.targets)
	}
	if !eventually(func() bool { return has() < 0 }) {
		t.Fatalf("subscriber has %d targets, want the %d current targets", has(), len(want))
	}
}

func TestGRPCStalledSubscriberDoesNotBlockOtherClients(t *testing.T) {
	ts := startGRPCTestServer(t)
	owner := ts.startClient(t, ClientConfig{})
	anchor := ts.registerAnchor(t, owner)
	healthy := newPeerView()
	ts.startClient(t, healthy.config())
	healthy.waitFor(t, map[Target]bool{anchor: true})

	// A subscriber that never reads its register stream.
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	stalled, err := ts.dial(t).Register(ctx)
	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}
	if err := stalled.Send(&tpb.RegisterOp{Registration: &tpb.RegisterOp_Subscription{Subscription: &tpb.Subscription{
		Op:         tpb.Subscription_SUBCRIBE,
		TargetType: "type1",
	}}}); err != nil {
		t.Fatalf("sending subscription failed: %v", err)
	}
	if !eventually(func() bool { return len(ts.subscribers("type1")) == 2 }) {
		t.Fatal("server did not register the second subscription")
	}

	// Register far more target data than the stalled subscriber's window
	// holds. The server must keep processing registrations, and the healthy
	// subscriber must receive all of them.
	const n = 300
	want := map[Target]bool{anchor: true}
	for i := 0; i < n; i++ {
		want[grpcTarget(i)] = true
	}
	runWithDeadline(t, "registering targets", func() error {
		for i := 0; i < n; i++ {
			if err := owner.NewTarget(grpcTarget(i)); err != nil {
				return err
			}
		}
		return nil
	})
	if !eventually(func() bool { r, _ := ts.counts(); return r == n+1 }) {
		r, _ := ts.counts()
		t.Fatalf("server registered %d of %d targets: a subscriber that is not reading blocked registration", r, n+1)
	}
	healthy.waitFor(t, want)

	// Once the stalled subscriber reads its stream, it catches up.
	var mu sync.Mutex
	var updates []*tpb.Target
	go func() {
		for {
			op, err := stalled.Recv()
			if err != nil {
				return
			}
			if tgt := op.GetTarget(); tgt.GetOp() == tpb.Target_ADD || tgt.GetOp() == tpb.Target_REMOVE {
				mu.Lock()
				updates = append(updates, tgt)
				mu.Unlock()
			}
		}
	}()
	view := func() (map[Target]bool, error) {
		mu.Lock()
		defer mu.Unlock()
		return replayTargets(updates)
	}
	if !eventually(func() bool { got, err := view(); return err == nil && sameTargets(got, want) }) {
		got, err := view()
		t.Fatalf("stalled subscriber has %d targets (err: %v) after reading its stream, want %d", len(got), err, len(want))
	}
}

func TestGRPCSlowSubscriberCatchesUp(t *testing.T) {
	ts := startGRPCTestServer(t)
	owner := ts.startClient(t, ClientConfig{})
	anchor := ts.registerAnchor(t, owner)
	slow := newPeerView()
	t.Cleanup(slow.resume)
	ts.startClient(t, slow.config())
	slow.waitFor(t, map[Target]bool{anchor: true})

	// The subscriber's handlers stop returning, so it stops reading its stream
	// while targets are registered and some of them removed again.
	slow.pause()
	const n, removed = 300, 20
	runWithDeadline(t, "changing targets", func() error {
		for i := 0; i < n; i++ {
			if err := owner.NewTarget(grpcTarget(i)); err != nil {
				return err
			}
		}
		for i := 0; i < removed; i++ {
			if err := owner.DeleteTarget(grpcTarget(i)); err != nil {
				return err
			}
		}
		return nil
	})
	if !eventually(func() bool { r, d := ts.counts(); return r == n+1 && d == removed }) {
		r, d := ts.counts()
		t.Fatalf("server processed %d registrations and %d removals, want %d and %d: a slow subscriber blocked other clients", r, d, n+1, removed)
	}

	// When the subscriber resumes, it catches up on the current targets.
	slow.resume()
	want := map[Target]bool{anchor: true}
	for i := removed; i < n; i++ {
		want[grpcTarget(i)] = true
	}
	slow.waitFor(t, want)
}
