package bench_test

import (
	"fmt"
	"testing"

	"github.com/edmand46/arpack/benchmarks/arpackmsg"
	benchfbs "github.com/edmand46/arpack/benchmarks/flatbuffers"
	benchpb "github.com/edmand46/arpack/benchmarks/proto"
	flatbuffers "github.com/google/flatbuffers/go"
	"google.golang.org/protobuf/proto"
)

// testMoveArpack returns a fully populated arpackmsg.MoveMessage for benchmarks.
func testMoveArpack() arpackmsg.MoveMessage {
	return arpackmsg.MoveMessage{
		Position:  arpackmsg.Vector3{X: 100, Y: -50, Z: 0},
		Velocity:  [3]float32{1.5, -2.5, 0},
		Waypoints: []arpackmsg.Vector3{{X: 10, Y: 20, Z: 0}, {X: -10, Y: 0, Z: 100}},
		PlayerID:  999,
		Active:    true,
		Visible:   false,
		Ghost:     true,
		Name:      "PlayerOne",
	}
}

// testMoveProto returns a fully populated proto MoveMessage for benchmarks.
func testMoveProto() *benchpb.MoveMessage {
	return &benchpb.MoveMessage{
		Position: &benchpb.Vector3{X: 100, Y: -50, Z: 0},
		Velocity: []float32{1.5, -2.5, 0},
		Waypoints: []*benchpb.Vector3{
			{X: 10, Y: 20, Z: 0},
			{X: -10, Y: 0, Z: 100},
		},
		PlayerId: 999,
		Active:   true,
		Visible:  false,
		Ghost:    true,
		Name:     "PlayerOne",
	}
}

// testMoveFbs returns a fully populated benchfbs.MoveMsg for benchmarks.
func testMoveFbs() *benchfbs.MoveMsg {
	return &benchfbs.MoveMsg{
		Position:  benchfbs.Vec3{X: 100, Y: -50, Z: 0},
		Velocity:  [3]float32{1.5, -2.5, 0},
		Waypoints: []benchfbs.Vec3{{X: 10, Y: 20, Z: 0}, {X: -10, Y: 0, Z: 100}},
		PlayerID:  999,
		Active:    true,
		Visible:   false,
		Ghost:     true,
		Name:      "PlayerOne",
	}
}

// TestMessageSize prints the wire size for each serialization format.
func TestMessageSize(t *testing.T) {
	// ArPack
	apMsg := testMoveArpack()
	apBuf := apMsg.Marshal(nil)
	fmt.Printf("ArPack   wire size: %d bytes\n", len(apBuf))

	// Protobuf
	pbMsg := testMoveProto()
	pbBuf, err := proto.Marshal(pbMsg)
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}
	fmt.Printf("Protobuf wire size: %d bytes\n", len(pbBuf))

	// FlatBuffers
	fbMsg := testMoveFbs()
	b := flatbuffers.NewBuilder(256)
	fbBuf := benchfbs.Marshal(b, fbMsg)
	fmt.Printf("FlatBuf  wire size: %d bytes\n", len(fbBuf))

	// Sanity-check round-trips
	var apOut arpackmsg.MoveMessage
	if _, err := apOut.Unmarshal(apBuf); err != nil {
		t.Fatalf("ArPack Unmarshal: %v", err)
	}
	if apOut.PlayerID != 999 || apOut.Name != "PlayerOne" {
		t.Errorf("ArPack round-trip mismatch: %+v", apOut)
	}

	var pbOut benchpb.MoveMessage
	if err := proto.Unmarshal(pbBuf, &pbOut); err != nil {
		t.Fatalf("proto.Unmarshal: %v", err)
	}
	if pbOut.PlayerId != 999 || pbOut.Name != "PlayerOne" {
		t.Errorf("Proto round-trip mismatch: PlayerId=%d Name=%s", pbOut.PlayerId, pbOut.Name)
	}

	var fbOut benchfbs.MoveMsg
	benchfbs.Unmarshal(fbBuf, &fbOut)
	if fbOut.PlayerID != 999 || fbOut.Name != "PlayerOne" {
		t.Errorf("FlatBuffers round-trip mismatch: %+v", fbOut)
	}
}

// --- ArPack benchmarks ---

func BenchmarkArPack_Marshal(b *testing.B) {
	msg := testMoveArpack()
	buf := msg.Marshal(nil)
	wireSize := len(buf)

	b.ReportAllocs()
	b.SetBytes(int64(wireSize))
	b.ResetTimer()

	var out []byte
	for i := 0; i < b.N; i++ {
		out = msg.Marshal(out[:0])
	}
	_ = out
}

func BenchmarkArPack_Unmarshal(b *testing.B) {
	msg := testMoveArpack()
	buf := msg.Marshal(nil)
	wireSize := len(buf)

	b.ReportAllocs()
	b.SetBytes(int64(wireSize))
	b.ResetTimer()

	var out arpackmsg.MoveMessage
	for i := 0; i < b.N; i++ {
		if _, err := out.Unmarshal(buf); err != nil {
			b.Fatal(err)
		}
	}
}

// --- Protobuf benchmarks ---

func BenchmarkProto_Marshal(b *testing.B) {
	msg := testMoveProto()
	buf, err := proto.Marshal(msg)
	if err != nil {
		b.Fatal(err)
	}
	wireSize := len(buf)

	b.ReportAllocs()
	b.SetBytes(int64(wireSize))
	b.ResetTimer()

	var out []byte
	for i := 0; i < b.N; i++ {
		out, err = proto.MarshalOptions{}.MarshalAppend(out[:0], msg)
		if err != nil {
			b.Fatal(err)
		}
	}
	_ = out
}

func BenchmarkProto_Unmarshal(b *testing.B) {
	msg := testMoveProto()
	buf, err := proto.Marshal(msg)
	if err != nil {
		b.Fatal(err)
	}
	wireSize := len(buf)

	b.ReportAllocs()
	b.SetBytes(int64(wireSize))
	b.ResetTimer()

	var out benchpb.MoveMessage
	for i := 0; i < b.N; i++ {
		out.Reset()
		if err := proto.Unmarshal(buf, &out); err != nil {
			b.Fatal(err)
		}
	}
}

// --- FlatBuffers benchmarks ---

func BenchmarkFlatBuffers_Marshal(b *testing.B) {
	msg := testMoveFbs()
	builder := flatbuffers.NewBuilder(256)
	buf := benchfbs.Marshal(builder, msg)
	wireSize := len(buf)

	b.ReportAllocs()
	b.SetBytes(int64(wireSize))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		benchfbs.Marshal(builder, msg)
	}
}

func BenchmarkFlatBuffers_Unmarshal(b *testing.B) {
	msg := testMoveFbs()
	builder := flatbuffers.NewBuilder(256)
	buf := benchfbs.Marshal(builder, msg)
	wireSize := len(buf)

	b.ReportAllocs()
	b.SetBytes(int64(wireSize))
	b.ResetTimer()

	var out benchfbs.MoveMsg
	for i := 0; i < b.N; i++ {
		out = benchfbs.MoveMsg{}
		benchfbs.Unmarshal(buf, &out)
	}
	_ = out
}
