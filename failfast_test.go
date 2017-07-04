package tcpfailfast

import (
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var s *TestServer

func TestMain(m *testing.M) {
	s = &TestServer{}
	s.Init()
	code := m.Run()
	time.Sleep(time.Second)
	os.Exit(code)
}

func readErr(conn net.Conn) <-chan error {
	c := make(chan error, 1)
	go func() {
		_, err := conn.Read(make([]byte, 4096))
		c <- err
	}()
	return c
}

func TestFailFast(t *testing.T) {
	conn, err := net.Dial("tcp", "10.1.0.20:1000")
	require.NoError(t, err, "error dialling")
	defer conn.Close()
	conn.(*net.TCPConn).SetNoDelay(true)
	require.NoError(t, FailFastTCP(conn.(*net.TCPConn), 2*time.Second), "error failfasting")

	// Write some data before going dark
	conn.Write([]byte("foobar\n"))
	time.Sleep(10 * time.Millisecond)

	// Go dark and see that the connection is closed within no more than 10 secs
	s.SetRadioSilence(true)
	conn.Write([]byte("foobar\n"))
	select {
	case <-readErr(conn):
		_, err = conn.Read(make([]byte, 4096)) // connection should be dead now
		require.Equal(t, io.EOF, err)
	case <-time.After(10 * time.Second):
		require.FailNow(t, "timed out waiting for connection termination")
	}
}
