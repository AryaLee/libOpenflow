package openflow

import (
	"fmt"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type Packet struct {
	decoded    []gopacket.LayerType
	decodedMap map[string]bool
	Eth        layers.Ethernet
	Arp        layers.ARP
	Ip4        layers.IPv4
	Icmpv4     layers.ICMPv4
	Tcp        layers.TCP
	Udp        layers.UDP
	Dhcpv4     layers.DHCPv4
	Truncated  bool
}

const maxLayers = 10

func ParsePacket(data []byte) (*Packet, error) {
	pkt := Packet{
		decoded:    make([]gopacket.LayerType, 0, maxLayers),
		decodedMap: make(map[string]bool),
	}
	dlayers := []gopacket.DecodingLayer{
		&pkt.Eth, &pkt.Arp, &pkt.Ip4, &pkt.Icmpv4, &pkt.Tcp, &pkt.Udp, &pkt.Dhcpv4}
	parser := gopacket.NewDecodingLayerParser(layers.LayerTypeEthernet, dlayers...)
	parser.IgnoreUnsupported = true
	err := parser.DecodeLayers(data, &pkt.decoded)
	if err != nil {
		return nil, err
	}

	pkt.Truncated = parser.Truncated
	for _, typ := range pkt.decoded {
		pkt.decodedMap[typ.String()] = true
	}
	return &pkt, nil
}

func (p *Packet) IsArp() bool {
	_, ok := p.decodedMap[layers.LayerTypeARP.String()]
	return ok
}

func (p *Packet) IsDhcpv4() bool {
	_, ok := p.decodedMap[layers.LayerTypeDHCPv4.String()]
	return ok
}

func (p *Packet) String() string {
	s := ""
	for _, typ := range p.decoded {
		switch typ {
		case layers.LayerTypeEthernet:
			s += fmt.Sprintf("\t%s\t src: %s, dst: %s, type: %s\n",
				typ, p.Eth.SrcMAC, p.Eth.DstMAC, p.Eth.EthernetType)
		case layers.LayerTypeIPv4:
			s += fmt.Sprintf("\t%s\t src: %s, dst: %s\n ", typ, p.Ip4.SrcIP, p.Ip4.DstIP)
		case layers.LayerTypeARP:
			op := "request"
			if p.Arp.Operation == 2 {
				op = "reply"
			}
			s += fmt.Sprintf("\t%s\t op: %s, srcMac: %s, srcIp: %s, dstMac: %s, dstIp: %s\n ",
				typ, op, net.HardwareAddr(p.Arp.SourceHwAddress), net.IP(p.Arp.SourceProtAddress),
				net.HardwareAddr(p.Arp.DstHwAddress), net.IP(p.Arp.DstProtAddress))
		case layers.LayerTypeTCP:
			s += fmt.Sprintf("\t%s\t srcPort: %s, dstPort: %s\n", typ, p.Tcp.SrcPort, p.Tcp.DstPort)
		case layers.LayerTypeUDP:
			s += fmt.Sprintf("\t%s\t srcPort: %s, dstPort: %s\n", typ, p.Udp.SrcPort, p.Udp.DstPort)
		case layers.LayerTypeICMPv4:
			s += fmt.Sprintf("\t%s\t type: %s, id: %d, seq: %d\n",
				typ, p.Icmpv4.TypeCode, p.Icmpv4.Id, p.Icmpv4.Seq)
		case layers.LayerTypeDHCPv4:
			s += fmt.Sprintf(
				"\t%s\t op: %s, clientMac: %s clientIp: %s, yClientIp: %s, options: %s\n",
				typ, p.Dhcpv4.Operation, p.Dhcpv4.ClientHWAddr, p.Dhcpv4.ClientIP,
				p.Dhcpv4.YourClientIP, p.Dhcpv4.Options)
		}
	}
	return s
}
