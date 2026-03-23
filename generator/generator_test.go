package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edmand46/arpack/parser"
)

const samplePath = "../testdata/sample.go"

func TestGenerateGo_Compiles(t *testing.T) {
	msgs, err := parser.ParseFile(samplePath)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	src, err := GenerateGo(msgs, "messages")
	if err != nil {
		t.Fatalf("GenerateGo: %v", err)
	}

	if len(src) == 0 {
		t.Fatal("GenerateGo returned empty output")
	}

	// Проверяем что содержит нужные функции
	code := string(src)
	for _, name := range []string{"Vector3", "MoveMessage", "SpawnMessage", "EnvelopeMessage"} {
		if !strings.Contains(code, "func (m *"+name+") Marshal") {
			t.Errorf("missing Marshal for %s", name)
		}
		if !strings.Contains(code, "func (m *"+name+") Unmarshal") {
			t.Errorf("missing Unmarshal for %s", name)
		}
	}

	t.Logf("Generated Go (%d bytes):\n%s", len(src), code)
}

func TestGenerateGo_RoundTrip(t *testing.T) {
	msgs, err := parser.ParseFile(samplePath)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	src, err := GenerateGo(msgs, "messages")
	if err != nil {
		t.Fatalf("GenerateGo: %v", err)
	}

	// Записываем в temp dir вместе с оригинальными структурами и тестом round-trip
	dir := t.TempDir()

	// Копируем testdata/sample.go (определения структур)
	sampleSrc, err := os.ReadFile(samplePath)
	if err != nil {
		t.Fatalf("ReadFile sample: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "messages.go"), sampleSrc, 0644); err != nil {
		t.Fatal(err)
	}

	// Сохраняем сгенерированный код
	if err := os.WriteFile(filepath.Join(dir, "messages_arpack.go"), src, 0644); err != nil {
		t.Fatal(err)
	}

	// Пишем round-trip тест
	roundTrip := `package messages

import (
	"math"
	"testing"
)

func TestRoundTrip_Vector3(t *testing.T) {
	orig := Vector3{X: 123.45, Y: -200.0, Z: 0.0}
	buf := orig.Marshal(nil)
	var got Vector3
	n, err := got.Unmarshal(buf)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if n != len(buf) {
		t.Errorf("consumed %d bytes, want %d", n, len(buf))
	}
	// Квантизация 16-бит даёт точность ≈ 1000/65535 ≈ 0.015
	const eps = float32(0.02)
	if math.Abs(float64(got.X-orig.X)) > float64(eps) {
		t.Errorf("X: got %v, want %v", got.X, orig.X)
	}
	if math.Abs(float64(got.Y-orig.Y)) > float64(eps) {
		t.Errorf("Y: got %v, want %v", got.Y, orig.Y)
	}
}

func TestRoundTrip_SpawnMessage(t *testing.T) {
	orig := SpawnMessage{
		EntityID: 42,
		Position: Vector3{X: 10, Y: 20, Z: 30},
		Health:   -100,
		Tags:     []string{"hero", "player"},
		Data:     []uint8{1, 2, 3},
	}
	buf := orig.Marshal(nil)
	var got SpawnMessage
	_, err := got.Unmarshal(buf)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.EntityID != orig.EntityID {
		t.Errorf("EntityID: got %d, want %d", got.EntityID, orig.EntityID)
	}
	if got.Health != orig.Health {
		t.Errorf("Health: got %d, want %d", got.Health, orig.Health)
	}
	if len(got.Tags) != len(orig.Tags) {
		t.Fatalf("Tags len: got %d, want %d", len(got.Tags), len(orig.Tags))
	}
	for i, tag := range orig.Tags {
		if got.Tags[i] != tag {
			t.Errorf("Tags[%d]: got %q, want %q", i, got.Tags[i], tag)
		}
	}
	if len(got.Data) != len(orig.Data) {
		t.Fatalf("Data len: got %d, want %d", len(got.Data), len(orig.Data))
	}
}

func TestRoundTrip_MoveMessage(t *testing.T) {
	orig := MoveMessage{
		Position:  Vector3{X: 100, Y: -50, Z: 0},
		Velocity:  [3]float32{1.5, -2.5, 0},
		Waypoints: []Vector3{{X: 10, Y: 20, Z: 0}, {X: -10, Y: 0, Z: 100}},
		PlayerID:  999,
		Active:    true,
		Visible:   false,
		Ghost:     true,
		Name:      "Alice",
	}
	buf := orig.Marshal(nil)
	var got MoveMessage
	_, err := got.Unmarshal(buf)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.PlayerID != orig.PlayerID {
		t.Errorf("PlayerID: got %d, want %d", got.PlayerID, orig.PlayerID)
	}
	if got.Name != orig.Name {
		t.Errorf("Name: got %q, want %q", got.Name, orig.Name)
	}
	if got.Active != orig.Active {
		t.Errorf("Active: got %v, want %v", got.Active, orig.Active)
	}
	if got.Visible != orig.Visible {
		t.Errorf("Visible: got %v, want %v", got.Visible, orig.Visible)
	}
	if got.Ghost != orig.Ghost {
		t.Errorf("Ghost: got %v, want %v", got.Ghost, orig.Ghost)
	}
	if len(got.Waypoints) != len(orig.Waypoints) {
		t.Fatalf("Waypoints len: got %d, want %d", len(got.Waypoints), len(orig.Waypoints))
	}
}

func TestRoundTrip_EnvelopeMessage(t *testing.T) {
	orig := EnvelopeMessage{
		Code:    OpcodeJoinRoom,
		Counter: 7,
	}
	buf := orig.Marshal(nil)
	var got EnvelopeMessage
	_, err := got.Unmarshal(buf)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Code != orig.Code {
		t.Errorf("Code: got %v, want %v", got.Code, orig.Code)
	}
	if got.Counter != orig.Counter {
		t.Errorf("Counter: got %d, want %d", got.Counter, orig.Counter)
	}
}
`
	if err := os.WriteFile(filepath.Join(dir, "roundtrip_test.go"), []byte(roundTrip), 0644); err != nil {
		t.Fatal(err)
	}

	// go.mod для temp пакета
	goMod := "module messages\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Запускаем go test
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test failed:\n%s", out)
	}
	t.Logf("go test output:\n%s", out)
}

func TestGenerateCSharp_Output(t *testing.T) {
	schema, err := parser.ParseSchemaFile(samplePath)
	if err != nil {
		t.Fatalf("ParseSchemaFile: %v", err)
	}

	src, err := GenerateCSharpSchema(schema, "Ragono.Messages")
	if err != nil {
		t.Fatalf("GenerateCSharpSchema: %v", err)
	}

	code := string(src)

	// Проверяем структуру выходного кода
	for _, name := range []string{"Vector3", "MoveMessage", "SpawnMessage", "EnvelopeMessage"} {
		if !strings.Contains(code, "public unsafe struct "+name) {
			t.Errorf("missing struct %s", name)
		}
		if !strings.Contains(code, "public int Serialize") {
			t.Errorf("missing Serialize in %s", name)
		}
		if !strings.Contains(code, "public static int Deserialize") {
			t.Errorf("missing Deserialize in %s", name)
		}
	}

	// Unsafe паттерны
	if !strings.Contains(code, "*(ushort*)ptr") {
		t.Error("missing unsafe ushort pointer cast")
	}
	if !strings.Contains(code, "byte* ptr = buffer") {
		t.Error("missing byte* ptr pattern")
	}

	// Нет BinaryWriter
	if strings.Contains(code, "BinaryWriter") || strings.Contains(code, "BinaryReader") {
		t.Error("should not contain BinaryWriter/BinaryReader")
	}
	if !strings.Contains(code, "public enum Opcode : ushort") {
		t.Error("missing Opcode enum")
	}
	if !strings.Contains(code, "Authorize = 1") || !strings.Contains(code, "JoinRoom = 2") {
		t.Error("missing enum values for Opcode")
	}
	if !strings.Contains(code, "public Opcode Code;") {
		t.Error("EnvelopeMessage.Code should use generated enum type")
	}

	t.Logf("Generated C# (%d bytes):\n%s", len(src), code)
}

func TestBoolPacking_GoCode(t *testing.T) {
	msgs, err := parser.ParseFile(samplePath)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	src, err := GenerateGo(msgs, "messages")
	if err != nil {
		t.Fatalf("GenerateGo: %v", err)
	}
	code := string(src)

	// Должны быть битовые операции
	if !strings.Contains(code, "_boolByte") {
		t.Error("missing bool bit-packing variable (_boolByte)")
	}
	if !strings.Contains(code, "|= 1 <<") {
		t.Error("missing bit OR operation for bool packing")
	}
	if !strings.Contains(code, "&(1<<") {
		t.Error("missing bit AND operation for bool unpacking")
	}

	// НЕ должно быть per-byte записи для bool-полей из MoveMessage
	// (Active, Visible, Ghost упакованы в 1 байт)
	if strings.Contains(code, "append(buf, 1)") {
		t.Error("should not emit per-byte bool encoding for packed bools")
	}
}

func TestBoolPacking_WireSize(t *testing.T) {
	src := `package p
type Flags struct {
	A bool
	B bool
	C bool
	D bool
	E bool
	F bool
	G bool
	H bool
	I bool
}
`
	msgs, err := parser.ParseSource(src)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	// 9 bool: первые 8 → 1 байт, последний → 1 байт = 2 байта
	got := packedMinWireSize(msgs[0].Fields)
	if got != 2 {
		t.Errorf("packedMinWireSize(9 consecutive bools): got %d, want 2", got)
	}

	src3 := `package p
type Flags3 struct {
	A bool
	B bool
	C bool
}
`
	msgs3, err := parser.ParseSource(src3)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	// 3 bool → 1 байт
	got3 := packedMinWireSize(msgs3[0].Fields)
	if got3 != 1 {
		t.Errorf("packedMinWireSize(3 consecutive bools): got %d, want 1", got3)
	}
}

func TestBoolPacking_SegmentFields(t *testing.T) {
	src := `package p
type Msg struct {
	A    bool
	B    bool
	ID   uint32
	C    bool
}
`
	msgs, err := parser.ParseSource(src)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	segs := segmentFields(msgs[0].Fields)
	// Ожидаем: [bool-group(A,B)] [single(ID)] [bool-group(C)]
	if len(segs) != 3 {
		t.Fatalf("expected 3 segments, got %d", len(segs))
	}
	if len(segs[0].bools) != 2 {
		t.Errorf("seg[0]: expected 2 bools, got %d", len(segs[0].bools))
	}
	if segs[1].single == nil || segs[1].single.Name != "ID" {
		t.Errorf("seg[1]: expected single field ID")
	}
	if len(segs[2].bools) != 1 {
		t.Errorf("seg[2]: expected 1 bool, got %d", len(segs[2].bools))
	}
}

func TestBoolPacking_CSharpCode(t *testing.T) {
	schema, err := parser.ParseSchemaFile(samplePath)
	if err != nil {
		t.Fatalf("ParseSchemaFile: %v", err)
	}

	src, err := GenerateCSharpSchema(schema, "Ragono.Messages")
	if err != nil {
		t.Fatalf("GenerateCSharpSchema: %v", err)
	}
	code := string(src)

	if !strings.Contains(code, "_boolByte") {
		t.Error("C#: missing bool bit-packing variable")
	}
	if !strings.Contains(code, "|= (byte)(1 <<") {
		t.Error("C#: missing bit OR for bool packing")
	}
	if !strings.Contains(code, "& (1 <<") {
		t.Error("C#: missing bit AND for bool unpacking")
	}
}
