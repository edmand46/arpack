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

func vectorClose(a, b Vector3) bool {
	const eps = float32(0.02)
	return math.Abs(float64(a.X-b.X)) <= float64(eps) &&
		math.Abs(float64(a.Y-b.Y)) <= float64(eps) &&
		math.Abs(float64(a.Z-b.Z)) <= float64(eps)
}

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

func TestUnmarshal_ClearsReusedReferenceSliceTail(t *testing.T) {
	long := SpawnMessage{
		EntityID: 1,
		Position: Vector3{X: 10, Y: 20, Z: 30},
		Health:   100,
		Tags:     []string{"keep", "drop"},
		Data:     []uint8{1},
	}
	short := SpawnMessage{
		EntityID: 2,
		Position: Vector3{X: -10, Y: 0, Z: 30},
		Health:   50,
		Tags:     []string{"only"},
		Data:     []uint8{2},
	}

	var got SpawnMessage
	if _, err := got.Unmarshal(long.Marshal(nil)); err != nil {
		t.Fatalf("Unmarshal long: %v", err)
	}
	if _, err := got.Unmarshal(short.Marshal(nil)); err != nil {
		t.Fatalf("Unmarshal short: %v", err)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "only" {
		t.Fatalf("Tags after short decode = %#v, want [only]", got.Tags)
	}
	full := got.Tags[:cap(got.Tags)]
	for i := len(got.Tags); i < len(full); i++ {
		if full[i] != "" {
			t.Fatalf("stale Tags tail at %d = %q, want cleared", i, full[i])
		}
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
	for i, want := range orig.Waypoints {
		if !vectorClose(got.Waypoints[i], want) {
			t.Errorf("Waypoints[%d]: got %#v, want %#v", i, got.Waypoints[i], want)
		}
	}
}

func TestUnmarshal_ReusesSliceCapacity(t *testing.T) {
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
	got := MoveMessage{Waypoints: make([]Vector3, 0, 8)}
	_, err := got.Unmarshal(buf)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if cap(got.Waypoints) != 8 {
		t.Fatalf("Waypoints cap = %d, want reused cap 8", cap(got.Waypoints))
	}
	if len(got.Waypoints) != len(orig.Waypoints) {
		t.Fatalf("Waypoints len: got %d, want %d", len(got.Waypoints), len(orig.Waypoints))
	}
	for i, want := range orig.Waypoints {
		if !vectorClose(got.Waypoints[i], want) {
			t.Errorf("Waypoints[%d]: got %#v, want %#v", i, got.Waypoints[i], want)
		}
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

	if !strings.Contains(code, "byte* ptr = buffer") {
		t.Error("missing byte* ptr pattern")
	}
	for _, helper := range []string{
		"WriteU16LE",
		"ReadU16LE",
		"WriteU32LE",
		"ReadU32LE",
		"WriteU64LE",
		"ReadU64LE",
		"WriteFloat32LE",
		"ReadFloat32LE",
		"EnsureFixedArray",
	} {
		if !strings.Contains(code, helper) {
			t.Errorf("missing C# helper %s", helper)
		}
	}
	for _, nativeCast := range []string{
		"*(ushort*)ptr",
		"*(short*)ptr",
		"*(uint*)ptr",
		"*(int*)ptr",
		"*(ulong*)ptr",
		"*(long*)ptr",
		"*(float*)ptr",
		"*(double*)ptr",
	} {
		if strings.Contains(code, nativeCast) {
			t.Errorf("generated C# should not use native-endian pointer cast %q", nativeCast)
		}
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
	if !strings.Contains(code, "internal static unsafe class ArpackGenerated") {
		t.Error("missing shared ArpackGenerated helper class")
	}
	if !strings.Contains(code, "EnsureReadable") {
		t.Error("missing bounds-check helper")
	}
	if !strings.Contains(code, "EnsureU16Length") {
		t.Error("missing uint16 length guard helper")
	}
	if !strings.Contains(code, "EnsureQuantizedRange") {
		t.Error("missing quantized range guard helper")
	}

	t.Logf("Generated C# (%d bytes):\n%s", len(src), code)
}

func TestGenerateGo_RuntimeGuards(t *testing.T) {
	schemaSrc := `package messages

type Quantized struct {
	Value float32 ` + "`" + `pack:"min=0,max=1,bits=8"` + "`" + `
}

type LengthLimited struct {
	Name  string
	Items []uint8
}
`

	schema, err := parser.ParseSchemaSource(schemaSrc)
	if err != nil {
		t.Fatalf("ParseSchemaSource: %v", err)
	}

	src, err := GenerateGoSchema(schema, "messages")
	if err != nil {
		t.Fatalf("GenerateGoSchema: %v", err)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "messages.go"), []byte(schemaSrc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "messages_arpack.go"), src, 0644); err != nil {
		t.Fatal(err)
	}

	runtimeTests := `package messages

import (
	"strings"
	"testing"
)

func expectPanicContaining(t *testing.T, want string, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic containing %q, got nil", want)
		}
		if !strings.Contains(r.(string), want) {
			t.Fatalf("expected panic containing %q, got %v", want, r)
		}
	}()
	fn()
}

func TestLengthGuard_String(t *testing.T) {
	expectPanicContaining(t, "string length for Name exceeds uint16 limit", func() {
		msg := LengthLimited{Name: strings.Repeat("a", 65536)}
		_ = msg.Marshal(nil)
	})
}

func TestLengthGuard_Slice(t *testing.T) {
	expectPanicContaining(t, "slice length for Items exceeds uint16 limit", func() {
		msg := LengthLimited{Items: make([]uint8, 65536)}
		_ = msg.Marshal(nil)
	})
}

func TestQuantizedRangeGuard(t *testing.T) {
	expectPanicContaining(t, "quantized value out of range for Value", func() {
		msg := Quantized{Value: 1.5}
		_ = msg.Marshal(nil)
	})
}
`
	if err := os.WriteFile(filepath.Join(dir, "guards_test.go"), []byte(runtimeTests), 0644); err != nil {
		t.Fatal(err)
	}

	goMod := "module messages\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test failed:\n%s", out)
	}
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

func TestGenerateCSharp_RuntimeGuards(t *testing.T) {
	if _, err := exec.LookPath("dotnet"); err != nil {
		t.Skip("dotnet not found")
	}

	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "Guarded",
				Fields: []parser.Field{
					{Name: "Name", Kind: parser.KindPrimitive, Primitive: parser.KindString},
					{
						Name:      "Ratio",
						Kind:      parser.KindPrimitive,
						Primitive: parser.KindFloat32,
						Quant:     &parser.QuantInfo{Min: 0, Max: 1, Bits: 8},
					},
				},
			},
			{
				Name: "FixedGuarded",
				Fields: []parser.Field{
					{
						Name:     "Values",
						Kind:     parser.KindFixedArray,
						FixedLen: 3,
						Elem: &parser.Field{
							Kind:      parser.KindPrimitive,
							Primitive: parser.KindUint8,
						},
					},
				},
			},
		},
	}

	src, err := GenerateCSharpSchema(schema, "Test.Messages")
	if err != nil {
		t.Fatalf("GenerateCSharpSchema: %v", err)
	}

	out := runGeneratedCSharpProgram(t, src, `using System;
using Test.Messages;

unsafe class Program
{
    static void Emit(string label, Action action)
    {
        try
        {
            action();
            Console.WriteLine(label + ":OK");
        }
        catch (Exception ex)
        {
            Console.WriteLine(label + ":" + ex.GetType().Name + ":" + ex.Message);
        }
    }

    static void Main()
    {
        Emit("TRUNC", () =>
        {
            byte[] data = Array.Empty<byte>();
            Guarded.Deserialize(data, out Guarded _);
        });

        Emit("LEN", () =>
        {
            var msg = new Guarded { Name = new string('a', 65536) };
            byte[] data = new byte[2];
            msg.Serialize(data);
        });

        Emit("QUANT", () =>
        {
            var msg = new Guarded { Ratio = 2f };
            byte[] data = new byte[4];
            msg.Serialize(data);
        });

        Emit("OVERFLOW", () =>
        {
            var msg = new Guarded { Name = "hello", Ratio = 0.5f };
            byte[] data = new byte[3];
            msg.Serialize(data);
        });

        Emit("FIXED_NULL", () =>
        {
            var msg = new FixedGuarded();
            byte[] data = new byte[3];
            msg.Serialize(data);
        });

        Emit("FIXED_SHORT", () =>
        {
            var msg = new FixedGuarded { Values = new byte[] { 1, 2 } };
            byte[] data = new byte[3];
            msg.Serialize(data);
        });

        Emit("FIXED_LONG", () =>
        {
            var msg = new FixedGuarded { Values = new byte[] { 1, 2, 3, 4 } };
            byte[] data = new byte[4];
            msg.Serialize(data);
        });

        Emit("SPAN_OK", () =>
        {
            var msg = new Guarded { Name = "ok", Ratio = 0.5f };
            byte[] data = new byte[16];
            int n = msg.Serialize(data);
            int consumed = Guarded.Deserialize(new ReadOnlySpan<byte>(data, 0, n), out Guarded decoded);
            if (consumed != n || decoded.Name != "ok")
            {
                throw new Exception("span roundtrip failed");
            }
        });
    }
}
`)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 8 {
		t.Fatalf("expected 8 output lines, got %d: %q", len(lines), out)
	}
	if !strings.Contains(lines[0], "ArgumentException:arpack: buffer too short for string length for Name") {
		t.Fatalf("expected truncated-input guard, got %q", lines[0])
	}
	if !strings.Contains(lines[1], "InvalidOperationException:arpack: string length for Name exceeds uint16 limit") {
		t.Fatalf("expected string length guard, got %q", lines[1])
	}
	if !strings.Contains(lines[2], "ArgumentOutOfRangeException:arpack: quantized value out of range for Ratio") {
		t.Fatalf("expected quantized range guard, got %q", lines[2])
	}
	if !strings.Contains(lines[3], "ArgumentException:arpack: buffer too small for string data for Name") {
		t.Fatalf("expected serialize write-bounds guard, got %q", lines[3])
	}
	if !strings.Contains(lines[4], "InvalidOperationException:arpack: fixed array for Values is null; expected length 3") {
		t.Fatalf("expected fixed-array null guard, got %q", lines[4])
	}
	if !strings.Contains(lines[5], "InvalidOperationException:arpack: fixed array for Values length mismatch: expected 3, got 2") {
		t.Fatalf("expected fixed-array short guard, got %q", lines[5])
	}
	if !strings.Contains(lines[6], "InvalidOperationException:arpack: fixed array for Values length mismatch: expected 3, got 4") {
		t.Fatalf("expected fixed-array long guard, got %q", lines[6])
	}
	if lines[7] != "SPAN_OK:OK" {
		t.Fatalf("expected span roundtrip, got %q", lines[7])
	}
}

func runGeneratedCSharpProgram(t *testing.T, generatedSrc []byte, programSrc string) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Messages.cs"), generatedSrc, 0o600); err != nil {
		t.Fatalf("write generated source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Program.cs"), []byte(programSrc), 0o600); err != nil {
		t.Fatalf("write program source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "RuntimeGuards.csproj"), []byte(`<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>net9.0</TargetFramework>
    <AllowUnsafeBlocks>true</AllowUnsafeBlocks>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
  </PropertyGroup>
</Project>
`), 0o600); err != nil {
		t.Fatalf("write csproj: %v", err)
	}

	build := exec.Command("dotnet", "build", "-c", "Release", "-o", "out")
	build.Dir = dir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("dotnet build failed: %v\n%s", err, out)
	}

	run := exec.Command(filepath.Join(dir, "out", "RuntimeGuards"))
	run.Dir = dir
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("runtime guard program failed: %v\n%s", err, out)
	}

	return string(out)
}
