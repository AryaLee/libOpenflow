package of14

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/contiv/libOpenflow/openflow/ofbase"
)

// TODO: set real types
type OXM = Oxm
type uint128 = ofbase.Uint128
type Checksum128 [16]byte
type Bitmap128 uint128
type Bitmap512 struct {
	a, b, c, d uint128
}
type Unimplemented struct{}
type BSNVport uint16
type ControllerURI uint16

func (h *Header) MessageType() uint8 {
	return h.Type
}

func (h *Header) MessageName() string {
	return Type(h.Type).String()
}

func (self *Checksum128) Decode(decoder *ofbase.Decoder) error {
	return nil
}

func (self *Checksum128) Serialize(encoder *ofbase.Encoder) error {
	return nil
}
func (self *Bitmap128) Decode(decoder *ofbase.Decoder) error {
	return nil
}

func (self *Bitmap128) Serialize(encoder *ofbase.Encoder) error {
	return nil
}
func (self *Bitmap512) Decode(decoder *ofbase.Decoder) error {
	return nil
}

func (self *Bitmap512) Serialize(encoder *ofbase.Encoder) error {
	return nil
}
func (self *BSNVport) Decode(decoder *ofbase.Decoder) error {
	return nil
}

func (self *BSNVport) Serialize(encoder *ofbase.Encoder) error {
	return nil
}
func (self *ControllerURI) Decode(decoder *ofbase.Decoder) error {
	return nil
}

func (self *ControllerURI) Serialize(encoder *ofbase.Encoder) error {
	return nil
}

type FmCmd uint8

func (self *FmCmd) Serialize(encoder *ofbase.Encoder) error {
	encoder.PutUint8(uint8(*self))
	return nil
}

func (self *FmCmd) Decode(decoder *ofbase.Decoder) error {
	*self = FmCmd(decoder.ReadUint8())
	return nil
}

type MatchBmap uint64

func (self *MatchBmap) Serialize(encoder *ofbase.Encoder) error {
	encoder.PutUint64(uint64(*self))
	return nil
}

func (self *MatchBmap) Decode(decoder *ofbase.Decoder) error {
	*self = MatchBmap(decoder.ReadUint64())
	return nil
}

type WcBmap uint64

func (self *WcBmap) Serialize(encoder *ofbase.Encoder) error {
	encoder.PutUint64(uint64(*self))
	return nil
}

func (self *WcBmap) Decode(decoder *ofbase.Decoder) error {
	*self = WcBmap(decoder.ReadUint64())
	return nil
}

type Match = MatchV3
type PortNo uint32

func (self *PortNo) Serialize(encoder *ofbase.Encoder) error {
	encoder.PutUint32(uint32(*self))
	return nil
}

func (self *PortNo) Decode(decoder *ofbase.Decoder) error {
	*self = PortNo(decoder.ReadUint32())
	return nil
}

func DecodeMessage(data []byte) (ofbase.Message, error) {
	header, err := DecodeHeader(ofbase.NewDecoder(data))
	if err != nil {
		return nil, err
	}

	return header.(ofbase.Message), nil
}

func (self *Port) Serialize(encoder *ofbase.Encoder) error {
	portNo := PortNo(*self)
	return portNo.Serialize(encoder)
}

func (self *Port) Decode(decoder *ofbase.Decoder) error {
	portNo := PortNo(*self)
	if err := portNo.Decode(decoder); err != nil {
		return err
	}
	*self = Port(portNo)
	return nil
}

func jsonValue(value interface{}) ([]byte, error) {
	switch t := value.(type) {
	case net.HardwareAddr:
		value = t.String()
	case net.IP:
		value = t.String()
	default:
		if s, ok := t.(fmt.Stringer); ok {
			value = s.String()
		} else {
			value = t
		}
	}

	return json.Marshal(value)
}
