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
}

func (t *TestServer) Init() {
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
			default:
				log.Fatalf("Error reading from TUN: %v", err)
			}
		}
	}()
}

func (t *TestServer) SetRadioSilence(v bool) {
	t.radioSilence.Store(v)
}

func (t *TestServer) receive(b []byte) {
	p := gopacket.NewPacket(b, layers.IPProtocolIPv4, gopacket.DecodeOptions{})
	if err := p.ErrorLayer(); err != nil {
		log.Fatalf("Error decoding packet: %v", err)
	}
	log.Println(p.String())

	tcpIn := p.TransportLayer().(*layers.TCP)
	ipIn := p.NetworkLayer().(*layers.IPv4)

	if tcpIn.SYN { // SYN-ACK
		ipOut := &layers.IPv4{
			SrcIP:    ipIn.DstIP,
			DstIP:    ipIn.SrcIP,
			Protocol: layers.IPProtocolTCP,
			TTL:      64}
		tcpOut := &layers.TCP{
			SrcPort: tcpIn.DstPort,
			DstPort: tcpIn.SrcPort,
			SYN:     true,
			ACK:     true,
			Ack:     tcpIn.Seq + 1,
			Seq:     1000}
		tcpOut.SetNetworkLayerForChecksum(ipOut)

		buf := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{
			ComputeChecksums: true,
			FixLengths:       true}
		gopacket.SerializeLayers(buf, opts, ipOut, tcpOut)
		t.iface.Write(buf.Bytes())

		op := gopacket.NewPacket(buf.Bytes(), layers.IPProtocolIPv4, gopacket.DecodeOptions{})
		log.Println("➡️  " + op.String())
	}
}

func (t *TestServer) Close() {
	t.iface.Close()
}
