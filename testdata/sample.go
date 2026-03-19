package messages

// Vector3 — трёхмерный вектор с квантизацией.
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

// MoveMessage содержит всё многообразие поддерживаемых типов.
type MoveMessage struct {
	Position  Vector3    // вложенный тип
	Velocity  [3]float32 // фиксированный массив без квантизации
	Waypoints []Vector3  // слайс вложенных типов
	PlayerID  uint32
	// 3 подряд bool → упаковываются в 1 байт
	Active  bool
	Visible bool
	Ghost   bool
	Name    string
}

// SpawnMessage — пример с целочисленными полями и массивами примитивов.
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
