package variables

const (
	// SRCP version
	Version = 1

	// Message types
	AuthRequest  = 0x01
	AuthResponse = 0x02
	KeyExchange  = 0x03
	Message      = 0x04
	MessageAck   = 0x05
	Disconnect   = 0x06

	// Authentication status
	AuthSuccess = 0x00
	AuthFailure = 0x01

	// Disconnection reasons
	UserRequest   = 0x00
	ServerRequest = 0x01
)
