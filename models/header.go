package models

type Header struct {
	Version  uint8
	Type     uint8
	Length   uint16
	Sequence uint32
}