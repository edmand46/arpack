package arpackmsg

type Vector3 struct {
	X float32 `pack:"min=-500,max=500,bits=16"`
	Y float32 `pack:"min=-500,max=500,bits=16"`
	Z float32 `pack:"min=-500,max=500,bits=16"`
}

type Opcode uint16

const (
	OpcodeUnknown Opcode = iota
	OpcodeAuthorize
	OpcodeJoinRoom
)

type MoveMessage struct {
	Position  Vector3
	Velocity  [3]float32
	Waypoints []Vector3
	PlayerID  uint32
	Active    bool
	Visible   bool
	Ghost     bool
	Name      string
}

type SpawnMessage struct {
	EntityID uint64
	Position Vector3
	Health   int16
	Tags     []string
	Data     []uint8
}

type EnvelopeMessage struct {
	Code    Opcode
	Counter uint8
}
