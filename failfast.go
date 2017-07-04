package tcpfailfast

import (
	"errors"
	"net"
	"time"
)

func FailFastTCP(tcp *net.TCPConn, timeout time.Duration) error {
	if timeout <= 0 {
		return errors.New("timeout must be > 0")
	}
	return ff(tcp, timeout)
}
