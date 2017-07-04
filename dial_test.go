package tcpfailfast

import (
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
	s.Close()
	os.Exit(code)
}

func TestFailFast(t *testing.T) {
	d := &FailFastDialer{
		Dialer:  net.Dialer{},
		Timeout: 5 * time.Second}

	conn, err := d.Dial("tcp", "10.1.0.20:1000")
	require.NoError(t, err, "Error dialling")
	defer conn.Close()
}
