package tcpfailfast

import (
	"fmt"
	"io"
	"log"
	"sync/atomic"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/songgao/water"
)

type TestServer struct {
	iface        *water.Interface
	radioSilence atomic.Value // bool
	seq          uint32
}

func (t *TestServer) Init() {
	t.radioSilence.Store(false)

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
				log.Println("TUN EOF")
				return
			default:
				log.Fatalf("Error reading from TUN: %v", err)
			}
		}
	}()
}

func (t *TestServer) SetRadioSilence(v bool) {
	t.radioSilence.Store(v)
}

func (t *TestServer) writePacket(ls ...gopacket.SerializableLayer) {
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true}
	gopacket.SerializeLayers(buf, opts, ls...)
	t.iface.Write(buf.Bytes())

	op := gopacket.NewPacket(buf.Bytes(), layers.IPProtocolIPv4, gopacket.DecodeOptions{})
	log.Println("➡️  " + op.String())
}

func (t *TestServer) receive(b []byte) {
	// receive is called when we receive a raw IP protocol on the TUN interface.
	// It implements TCP in an _extremely_ bare-bones and hacky fashion and
	// knows how to do the SYN/SYN-ACK/ACK handshake and ACK inbound packets.
	// When the test server is instructed to go dark, it will not respond to
	// packets at all (ie. it will not respond to handshakes and it will not
	// ack inbound packets.
	p := gopacket.NewPacket(b, layers.IPProtocolIPv4, gopacket.DecodeOptions{})
	if err := p.ErrorLayer(); err != nil {
		log.Fatalf("Error decoding packet: %v", err)
	}
	log.Println("⬅️  " + p.String())

	tcpIn, ok := p.TransportLayer().(*layers.TCP)
	if !ok {
		return
	}
	ipIn := p.NetworkLayer().(*layers.IPv4)
	ipOut := &(*ipIn)
	ipOut.SrcIP, ipOut.DstIP = ipOut.DstIP, ipOut.SrcIP

	if t.radioSilence.Load().(bool) {
		log.Println("📻  🙊 Ignoring packet; radio silence")
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
