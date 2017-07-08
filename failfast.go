package tcpfailfast

import (
	"errors"
	"net"
	"time"
)

var ErrUnsupported = errors.New("tcp-failfast is unsupported on this platform")
func FailFastTCP(tcp *net.TCPConn, timeout time.Duration) error {
	if timeout <= 0 {
		return errors.New("timeout must be > 0")
	}
	return ff(tcp, timeout)
}
