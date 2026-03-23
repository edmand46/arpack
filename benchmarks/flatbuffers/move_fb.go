package benchfbs

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

// Vec3 mirrors the MoveMessage's Vector3 fields for FlatBuffers encoding.
type Vec3 struct {
	X, Y, Z float32
}

// MoveMsg is the Go struct used in the FlatBuffers benchmark.
type MoveMsg struct {
	Position               Vec3
	Velocity               [3]float32
	Waypoints              []Vec3
	PlayerID               uint32
	Active, Visible, Ghost bool
	Name                   string
}

// vtable slot indices for MoveMessage fields (0-based slot -> vtable offset = 4 + 2*slot)
// slot 0 -> position  (vtable offset 4)
// slot 1 -> velocity  (vtable offset 6)
// slot 2 -> waypoints (vtable offset 8)
// slot 3 -> player_id (vtable offset 10)
// slot 4 -> active    (vtable offset 12)
// slot 5 -> visible   (vtable offset 14)
// slot 6 -> ghost     (vtable offset 16)
// slot 7 -> name      (vtable offset 18)
const (
	slotPosition  = 0
	slotVelocity  = 1
	slotWaypoints = 2
	slotPlayerID  = 3
	slotActive    = 4
	slotVisible   = 5
	slotGhost     = 6
	slotName      = 7
)

// buildVec3 writes a Vec3 as a table with 3 float32 fields and returns its offset.
// Vec3 slots: x=0, y=1, z=2
func buildVec3(b *flatbuffers.Builder, v Vec3) flatbuffers.UOffsetT {
	b.StartObject(3)
	b.PrependFloat32Slot(0, v.X, 0)
	b.PrependFloat32Slot(1, v.Y, 0)
	b.PrependFloat32Slot(2, v.Z, 0)
	return b.EndObject()
}

// Marshal serialises msg into a FlatBuffer using b and returns the finished bytes.
// The builder is reset before use, so callers can reuse it across calls.
func Marshal(b *flatbuffers.Builder, msg *MoveMsg) []byte {
	b.Reset()

	// 1. Build all variable-length data first (must be done outside object construction).

	// name string
	nameOff := b.CreateString(msg.Name)

	// velocity vector: 3 × float32
	b.StartVector(4, 3, 4)
	for i := 2; i >= 0; i-- {
		b.PrependFloat32(msg.Velocity[i])
	}
	velOff := b.EndVector(3)

	// waypoints vector: repeated Vec3 tables (build each table first, collect offsets)
	wpOffsets := make([]flatbuffers.UOffsetT, len(msg.Waypoints))
	for i, wp := range msg.Waypoints {
		wpOffsets[i] = buildVec3(b, wp)
	}
	b.StartVector(4, len(wpOffsets), 4)
	for i := len(wpOffsets) - 1; i >= 0; i-- {
		b.PrependUOffsetT(wpOffsets[i])
	}
	wpVecOff := b.EndVector(len(wpOffsets))

	// position table
	posOff := buildVec3(b, msg.Position)

	// 2. Build the MoveMessage table.
	b.StartObject(8)
	b.PrependUOffsetTSlot(slotPosition, posOff, 0)
	b.PrependUOffsetTSlot(slotVelocity, velOff, 0)
	b.PrependUOffsetTSlot(slotWaypoints, wpVecOff, 0)
	b.PrependUint32Slot(slotPlayerID, msg.PlayerID, 0)
	b.PrependBoolSlot(slotActive, msg.Active, false)
	b.PrependBoolSlot(slotVisible, msg.Visible, false)
	b.PrependBoolSlot(slotGhost, msg.Ghost, false)
	b.PrependUOffsetTSlot(slotName, nameOff, 0)
	root := b.EndObject()

	b.Finish(root)
	return b.FinishedBytes()
}

// readVec3 reads a Vec3 from a table at the given absolute position in buf.
func readVec3(buf []byte, tablePos flatbuffers.UOffsetT) Vec3 {
	tab := flatbuffers.Table{Bytes: buf, Pos: tablePos}
	var v Vec3
	if o := tab.Offset(4); o != 0 { // slot 0 -> vtable offset 4
		v.X = tab.GetFloat32(tab.Pos + flatbuffers.UOffsetT(o))
	}
	if o := tab.Offset(6); o != 0 { // slot 1 -> vtable offset 6
		v.Y = tab.GetFloat32(tab.Pos + flatbuffers.UOffsetT(o))
	}
	if o := tab.Offset(8); o != 0 { // slot 2 -> vtable offset 8
		v.Z = tab.GetFloat32(tab.Pos + flatbuffers.UOffsetT(o))
	}
	return v
}

// Unmarshal reads all fields from a finished FlatBuffer into out.
func Unmarshal(buf []byte, out *MoveMsg) {
	// The root offset is stored at byte 0 of the finished buffer.
	rootPos := flatbuffers.GetUOffsetT(buf)
	tab := flatbuffers.Table{Bytes: buf, Pos: rootPos}

	// position (slot 0, vtable offset 4)
	if o := tab.Offset(4); o != 0 {
		absOff := tab.Pos + flatbuffers.UOffsetT(o)
		posPos := tab.Indirect(absOff)
		out.Position = readVec3(buf, posPos)
	}

	// velocity vector (slot 1, vtable offset 6)
	if o := tab.Offset(6); o != 0 {
		vecStart := tab.Vector(flatbuffers.UOffsetT(o))
		for i := 0; i < 3; i++ {
			out.Velocity[i] = flatbuffers.GetFloat32(buf[int(vecStart)+i*4:])
		}
	}

	// waypoints vector (slot 2, vtable offset 8)
	if o := tab.Offset(8); o != 0 {
		n := tab.VectorLen(flatbuffers.UOffsetT(o))
		out.Waypoints = make([]Vec3, n)
		vecStart := tab.Vector(flatbuffers.UOffsetT(o))
		for i := 0; i < n; i++ {
			// Each element is an UOffsetT pointing to the table.
			elemOff := vecStart + flatbuffers.UOffsetT(i*4)
			tablePos := elemOff + flatbuffers.GetUOffsetT(buf[elemOff:])
			out.Waypoints[i] = readVec3(buf, tablePos)
		}
	}

	// player_id (slot 3, vtable offset 10)
	if o := tab.Offset(10); o != 0 {
		out.PlayerID = tab.GetUint32(tab.Pos + flatbuffers.UOffsetT(o))
	}

	// active (slot 4, vtable offset 12)
	if o := tab.Offset(12); o != 0 {
		out.Active = tab.GetBool(tab.Pos + flatbuffers.UOffsetT(o))
	}

	// visible (slot 5, vtable offset 14)
	if o := tab.Offset(14); o != 0 {
		out.Visible = tab.GetBool(tab.Pos + flatbuffers.UOffsetT(o))
	}

	// ghost (slot 6, vtable offset 16)
	if o := tab.Offset(16); o != 0 {
		out.Ghost = tab.GetBool(tab.Pos + flatbuffers.UOffsetT(o))
	}

	// name (slot 7, vtable offset 18)
	if o := tab.Offset(18); o != 0 {
		absOff := tab.Pos + flatbuffers.UOffsetT(o)
		out.Name = tab.String(absOff)
	}
}
