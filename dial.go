package tcpfailfast

import (
	"context"
	"net"
	"time"
)

type FailFastDialer struct {
	net.Dialer
	// Timeout after which retransmissions should stop and the connection should
	// be closed
	Timeout time.Duration
}

func (f *FailFastDialer) timeout() time.Duration {
	if f.Timeout == 0 {
		return 5 * time.Minute
	}
	return f.Timeout
}

func (f *FailFastDialer) Dial(network, address string) (net.Conn, error) {
	c, err := f.Dialer.Dial(network, address)
	if tcp, ok := c.(*net.TCPConn); ok && err == nil {
		err = failFast(tcp, f.timeout())
	}
	return c, err
}

func (f *FailFastDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	c, err := f.Dialer.DialContext(ctx, network, address)
	if tcp, ok := c.(*net.TCPConn); ok && err == nil {
		err = failFast(tcp, f.timeout())
	}
	return c, err
}
