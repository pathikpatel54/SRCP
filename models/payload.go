package models

// AuthRequestPayload struct represents a SRCP AUTH_REQUEST payload
type AuthRequestPayload struct {
	Username [32]byte
	Password [32]byte
}

// AuthResponsePayload struct represents a SRCP AUTH_RESPONSE payload
type AuthResponsePayload struct {
	Status uint8
}

// PublicKeyPayload struct represents a SRCP PUBLIC_KEY payload
type PublicKeyPayload struct {
	Username [32]byte
	Key      [512]byte
}

// MessagePayload struct represents a SRCP MESSAGE payload
type MessagePayload struct {
	Timestamp uint32
	Sender    [32]byte
	Recipient [32]byte
	TextLen   uint16
	Data      [2048]byte
}

// MessageAckPayload struct represents a SRCP MESSAGE_ACK payload
type MessageAckPayload struct {
	Sequence uint32
}

// DisconnectPayload struct represents a SRCP DISCONNECT payload
type DisconnectPayload struct {
	Reason uint8
}
