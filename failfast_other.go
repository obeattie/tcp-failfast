// +build !linux,!darwin

package tcpfailfast

import (
	"net"
	"time"
)

func failFast(tcp *net.TCPConn, timeout time.Duration) error {
	return nil // Unsupported platform
}
