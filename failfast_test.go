package tcpfailfast

import (
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	timeout = 5 * time.Second
	grace   = timeout * 2
)

func readErr(conn net.Conn) <-chan error {
	c := make(chan error, 1)
	go func() {
		_, err := conn.Read(make([]byte, 4096))
		c <- err
	}()
	return c
}

func countFDs() int {
	d, err := os.Open("/proc/self/fd")
	if err != nil {
		panic(err)
	}
	defer d.Close()

	names, err := d.Readdirnames(-1)
	if err != nil {
		panic(err)
	}
	return len(names)
}

// TestFailFast checks the behaviour of the FailFastTCP function
func TestFailFast(t *testing.T) {
	s := server(t)
	defer s.Close()

	conn, err := net.Dial("tcp", "10.1.0.20:1000")
	require.NoError(t, err, "error dialling")
	conn.(*net.TCPConn).SetNoDelay(true)
	require.NoError(t, FailFastTCP(conn.(*net.TCPConn), timeout), "error failfasting")
	// Write some data before going dark
	conn.Write([]byte("foobar\n"))

	// Go dark and see that the connection is closed within no more than 10 secs
	s.Silence()
	conn.Write([]byte("foobar\n"))
	start := time.Now()
	select {
	case <-readErr(conn):
		require.True(t, time.Since(start) >= timeout, "connection dropped before timeout")

		_, err = conn.Read(make([]byte, 4096))
		require.Equal(t, io.EOF, err) // check we get the right error
	case <-time.After(grace):
		require.FailNow(t, "timed out waiting for connection termination")
	}
}

// TestFailFastLeak checks FD leak
func TestFailFastLeak(t *testing.T) {
	before := countFDs()
	conn, err := net.Dial("tcp", "8.8.8.8:53")
	require.NoError(t, err, "error dialling")
	conn.(*net.TCPConn).SetNoDelay(true)
	require.NoError(t, FailFastTCP(conn.(*net.TCPConn), timeout), "error failfasting")
	conn.Close()
	after := countFDs()
	require.Equal(t, before, after, "filedescriptor leaked")
}

// TestControl checks that a connection is *not* closed early without calling
// FailFastTCP (this is a sanity check to ensure that the platform doesn't
// "fail fast" by default, which would invalidate the result of TestFailFast)
func TestControl(t *testing.T) {
	s := server(t)
	defer s.Close()

	conn, err := net.Dial("tcp", "10.1.0.20:1000")
	require.NoError(t, err, "error dialling")
	conn.(*net.TCPConn).SetNoDelay(true)
	// Write some data before going dark
	conn.Write([]byte("foobar\n"))

	// Go dark and see that the connection is closed within no more than 10 secs
	s.Silence()
	conn.Write([]byte("foobar\n"))
	select {
	case <-readErr(conn):
		require.FailNow(t, "control connection terminated before grace")
	case <-time.After(grace):
	}
}
