package tcpfailfast

import (
	"fmt"
	"io"
	"sync/atomic"
	"testing"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/songgao/water"
)

func server(t *testing.T) *testServer {
	s := &testServer{
		t: t}
	s.Init()
	return s
}

type testServer struct {
	t      *testing.T
	iface  *water.Interface
	silent atomic.Value // bool
	seq    uint32
}

func (t *testServer) Init() {
	t.silent.Store(false)

	iface, err := water.New(water.Config{
		DeviceType: water.TUN})
	if err != nil {
		panic(fmt.Sprintf("error creating TUN interface: %v", err))
	}
	setupTUN(iface)
	t.iface = iface

	go func() {
		b := make([]byte, 4096)
		for {
			n, err := iface.Read(b)
			switch err {
			case nil:
				t.receive(b[:n])
			case io.EOF:
				if n > 0 {
					t.receive(b[:n])
				}
				t.t.Logf("TUN EOF")
				return
			default:
				t.t.Logf("Error reading from TUN: %v", err)
				return
			}
		}
	}()
}

func (t *testServer) Close() {
	t.iface.Close()
}

func (t *testServer) Silence() {
	t.silent.Store(true)
}

func (t *testServer) writePacket(ls ...gopacket.SerializableLayer) {
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true}
	gopacket.SerializeLayers(buf, opts, ls...)
	t.iface.Write(buf.Bytes())

	op := gopacket.NewPacket(buf.Bytes(), layers.IPProtocolIPv4, gopacket.DecodeOptions{})
	t.t.Logf("➡️  %v", op)
}

func (t *testServer) receive(b []byte) {
	// receive is called when we receive a raw IP protocol on the TUN interface.
	// It implements TCP in an _extremely_ bare-bones and hacky fashion and
	// knows how to do the SYN/SYN-ACK/ACK handshake and ACK inbound packets.
	// When the test server is instructed to go dark, it will not respond to
	// packets at all (ie. it will not respond to handshakes and it will not
	// ack inbound packets.
	p := gopacket.NewPacket(b, layers.IPProtocolIPv4, gopacket.DecodeOptions{})
	if err := p.ErrorLayer(); err != nil {
		t.t.Fatalf("Error decoding packet: %v", err)
	}
	t.t.Logf("⬅️  %v", p)

	tcpIn, ok := p.TransportLayer().(*layers.TCP)
	if !ok {
		return
	}
	ipIn := p.NetworkLayer().(*layers.IPv4)
	ipOut := &(*ipIn)
	ipOut.SrcIP, ipOut.DstIP = ipOut.DstIP, ipOut.SrcIP

	if t.silent.Load().(bool) {
		t.t.Log("📻  🙊 Ignoring packet; radio silence")
		return
	}

	if tcpIn.SYN { // Reply with SYN-ACK
		t.seq++
		tcpOut := &layers.TCP{
			SrcPort: tcpIn.DstPort,
			DstPort: tcpIn.SrcPort,
			SYN:     true,
			ACK:     true,
			Ack:     tcpIn.Seq + 1,
			Window:  tcpIn.Window,
			Seq:     t.seq}
		tcpOut.SetNetworkLayerForChecksum(ipOut)
		t.writePacket(ipOut, tcpOut)
	} else if len(tcpIn.Payload) > 0 {
		t.seq++
		tcpOut := &layers.TCP{
			SrcPort: tcpIn.DstPort,
			DstPort: tcpIn.SrcPort,
			ACK:     true,
			Ack:     tcpIn.Seq + uint32(len(tcpIn.Payload)),
			Window:  tcpIn.Window,
			Seq:     t.seq}
		tcpOut.SetNetworkLayerForChecksum(ipOut)
		t.writePacket(ipOut, tcpOut)
	}
}
