package tunnel

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	tpb "github.com/openconfig/grpctunnel/proto/tunnel"
)

var (
	// RetryBaseDelay is the initial retry interval for re-connecting tunnel server/client.
	RetryBaseDelay = time.Second
	// RetryMaxDelay caps the retry interval for re-connecting attempts.
	RetryMaxDelay = time.Minute
	// RetryRandomization is the randomization factor applied to the retry
	// interval.
	RetryRandomization = 0.5
)

// Conn is a wraper as a net.Conn interface.
type Conn struct {
	io.ReadWriteCloser
}

// LocalAddr is trivial implementation, in order to match interface net.Conn.
func (tc *Conn) LocalAddr() net.Addr { return nil }

// RemoteAddr is trivial implementation, in order to match interface net.Conn.
func (tc *Conn) RemoteAddr() net.Addr { return nil }

// SetDeadline is trivial implementation, in order to match interface net.Conn.
func (tc *Conn) SetDeadline(t time.Time) error { return nil }

// SetReadDeadline is trivial implementation, in order to match interface net.Conn.
func (tc *Conn) SetReadDeadline(t time.Time) error { return nil }

// SetWriteDeadline is trivial implementation, in order to match interface net.Conn.
func (tc *Conn) SetWriteDeadline(t time.Time) error { return nil }

// ServerConn (re-)tries and returns a tunnel connection.
func ServerConn(ctx context.Context, ts *Server, addr string, target *Target) (*Conn, error) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0 // Retry Subscribe indefinitely.
	bo.InitialInterval = RetryBaseDelay
	bo.MaxInterval = RetryMaxDelay
	bo.RandomizationFactor = RetryRandomization

	for {
		session, err := ts.NewSession(ctx, ServerSession{Target: *target})
		if err == nil {
			return &Conn{session}, nil
		}
		duration := bo.NextBackOff()
		log.Printf("Failed to get tunnel connection: %v.\nRetrying in %s.", err, duration)
		time.Sleep(duration)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

	}

}

func registerTunnelClient(ctx context.Context, addr string, cert string, l *Listener,
	targets map[Target]struct{}) (*Client, error) {
	opts := []grpc.DialOption{grpc.WithDefaultCallOptions()}
	if cert == "" {
		opts = append(opts, grpc.WithInsecure())
	} else {
		creds, err := credentials.NewClientTLSFromFile(cert, "")
		if err != nil {
			return nil, fmt.Errorf("failed to load credentials: %v", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	}
	var err error
	l.cc, err = grpc.Dial(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("grpc dial error: %v", err)
	}

	registerHandler := func(t Target) error {
		if _, ok := targets[t]; !ok {
			return fmt.Errorf("client cannot handle target ID: %s, type: %s", t.ID, t.Type)
		}
		log.Printf("register handler received id: %v, type: %v", t.ID, t.Type)
		return nil
	}

	handler := func(t Target, i io.ReadWriteCloser) error {
		log.Printf("handler called for id: %v, type: %v", t.ID, t.Type)
		l.chIO <- i
		return nil
	}

	client, err := NewClient(tpb.NewTunnelClient(l.cc), ClientConfig{
		RegisterHandler: registerHandler,
		Handler:         handler,
	}, targets)
	if err != nil {
		return nil, fmt.Errorf("failed to create tunnel client: %v", err)
	}
	if err := client.Register(ctx); err != nil {
		return nil, err
	}
	return client, nil
}

// Listener wraps a tunnel connection.
type Listener struct {
	conn  []io.ReadWriteCloser
	addr  tunnelAddr
	chIO  chan io.ReadWriteCloser
	chErr chan error
	cc    *grpc.ClientConn
}

// Accept waits and returns a tunnel connection.
func (l *Listener) Accept() (net.Conn, error) {
	select {
	case err := <-l.chErr:
		return nil, fmt.Errorf("failed to get tunnel listener: %v", err)
	case conn := <-l.chIO:
		l.conn = append(l.conn, conn)
		log.Printf("tunnel listen setup")
		return &Conn{conn}, nil
	}
}

// Close close the embedded connection. Will need more implementation to handle multiple connections.
func (l *Listener) Close() error {
	var errs []string
	if l.cc != nil {
		if e := l.cc.Close(); e != nil {
			errs = append(errs, fmt.Sprintf("failed to close listener.cc: %v", e))
		}
	}

	if l.conn != nil {
		for _, conn := range l.conn {
			if e := conn.Close(); e != nil {
				errs = append(errs, fmt.Sprintf("failed to close listener.conn: %v", e))
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, "\n"))
	}
	return nil
}

// Addr is a trivial implementation.
func (l *Listener) Addr() net.Addr { return l.addr }

type tunnelAddr struct {
	network string
	address string
}

func (a tunnelAddr) Network() string { return a.network }
func (a tunnelAddr) String() string  { return a.address }

// Listen create a tunnel client and returns a Listener.
func Listen(ctx context.Context, addr string, cert string, targets map[Target]struct{}) (net.Listener, error) {
	l := &Listener{}
	l.addr = tunnelAddr{network: "tcp", address: addr}
	l.chErr = make(chan error)
	l.chIO = make(chan io.ReadWriteCloser)

	// Doing a for loop so that it will retry even the tunnel server is not reachable.
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0 // Retry Subscribe indefinitely.
	bo.InitialInterval = RetryBaseDelay
	bo.MaxInterval = RetryMaxDelay
	bo.RandomizationFactor = RetryRandomization

	for {
		if c, err := registerTunnelClient(ctx, addr, cert, l, targets); err == nil {
			go func() {
				c.Start(ctx)
				if err := c.Error(); err != nil {
					l.chErr <- err
				}
			}()
			return l, nil
		}
		if err := l.Close(); err != nil {
			log.Printf("%v", err)
		}

		// tunnel client establishes a tunnel session if it succeeded.
		// retry if it fails.
		duration := bo.NextBackOff()
		log.Printf("Tunnel listener will retry in %s.", duration)
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("tunnel listener context canceled")
		case err := <-l.chErr:
			log.Printf("failed to get tunnel listener: %v", err)
		case <-time.After(duration):
		}
	}
}
