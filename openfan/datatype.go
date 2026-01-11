package openfan

import "fmt"

type (
	Command uint8
	Fan     uint8
)

type HardwareInfo struct {
	Revision          string `json:"revision" cbor:"1,keyasint,omitempty,omitzero"`
	MCU               string `json:"mcu" cbor:"2,keyasint,omitempty,omitzero"`
	USB               string `json:"usb" cbor:"3,keyasint,omitempty,omitzero"`
	FanChannelsTotal  string `json:"fan_channels_total" cbor:"4,keyasint,omitempty,omitzero"`
	FanChannelsArch   string `json:"fan_channels_arch" cbor:"5,keyasint,omitempty,omitzero"`
	FanChannelsDriver string `json:"fan_channels_driver" cbor:"6,keyasint,omitempty,omitzero"`
}

type FirmwareInfo struct {
	Revision        string `json:"revision" cbor:"1,keyasint,omitempty,omitzero"`
	ProtocolVersion string `json:"protocol_version" cbor:"2,keyasint,omitempty,omitzero"`
}

func f2x[T ~uint8](v T) (byte, byte) {
	s := fmt.Sprintf("%02X", v)
	return s[0], s[1]
}

func f4x[T ~uint16](v T) (byte, byte, byte, byte) {
	s := fmt.Sprintf("%04X", v)
	return s[0], s[1], s[2], s[3]
}
