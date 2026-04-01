package parser

import (
	"strings"
	"testing"
)

const sampleSource = `package messages

type Vector3 struct {
	X float32 ` + "`" + `pack:"min=-500,max=500,bits=16"` + "`" + `
	Y float32 ` + "`" + `pack:"min=-500,max=500,bits=16"` + "`" + `
	Z float32 ` + "`" + `pack:"min=-500,max=500,bits=16"` + "`" + `
}

type MoveMessage struct {
	Position  Vector3
	Velocity  [3]float32
	Waypoints []Vector3
	PlayerID  uint32
	Name      string
	Active    bool
}

type SpawnMessage struct {
	ID       uint64
	Position Vector3
	Tags     [4]string
	Data     []uint8
}
`

const enumSource = `package messages

type Opcode uint16

const (
	OpcodeUnknown Opcode = iota
	OpcodeAuthorize
	OpcodeJoinRoom
)

type EnvelopeMessage struct {
	Code    Opcode
	Counter uint8
}
`

func TestParseSource_Primitives(t *testing.T) {
	msgs, err := ParseSource(sampleSource)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
}

func TestParseSource_Vector3(t *testing.T) {
	msgs, err := ParseSource(sampleSource)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	v3 := msgs[0]
	if v3.Name != "Vector3" {
		t.Fatalf("expected Vector3, got %s", v3.Name)
	}
	if len(v3.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(v3.Fields))
	}

	for _, f := range v3.Fields {
		if f.Kind != KindPrimitive {
			t.Errorf("field %s: expected KindPrimitive, got %d", f.Name, f.Kind)
		}
		if f.Primitive != KindFloat32 {
			t.Errorf("field %s: expected KindFloat32, got %d", f.Name, f.Primitive)
		}
		if f.Quant == nil {
			t.Errorf("field %s: expected quant info, got nil", f.Name)
			continue
		}
		if f.Quant.Min != -500 || f.Quant.Max != 500 || f.Quant.Bits != 16 {
			t.Errorf("field %s: wrong quant info %+v", f.Name, f.Quant)
		}
	}
}

func TestParseSource_MoveMessage(t *testing.T) {
	msgs, err := ParseSource(sampleSource)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	msg := msgs[1]
	if msg.Name != "MoveMessage" {
		t.Fatalf("expected MoveMessage, got %s", msg.Name)
	}

	tests := []struct {
		name     string
		kind     FieldKind
		typeName string
		fixedLen int
		elemKind FieldKind
	}{
		{"Position", KindNested, "Vector3", 0, 0},
		{"Velocity", KindFixedArray, "", 3, KindPrimitive},
		{"Waypoints", KindSlice, "", 0, KindNested},
		{"PlayerID", KindPrimitive, "", 0, 0},
		{"Name", KindPrimitive, "", 0, 0},
		{"Active", KindPrimitive, "", 0, 0},
	}

	if len(msg.Fields) != len(tests) {
		t.Fatalf("expected %d fields, got %d", len(tests), len(msg.Fields))
	}

	for i, tc := range tests {
		f := msg.Fields[i]
		if f.Name != tc.name {
			t.Errorf("[%d] expected field %s, got %s", i, tc.name, f.Name)
		}
		if f.Kind != tc.kind {
			t.Errorf("field %s: expected kind %d, got %d", tc.name, tc.kind, f.Kind)
		}
		if tc.typeName != "" && f.TypeName != tc.typeName {
			t.Errorf("field %s: expected TypeName %s, got %s", tc.name, tc.typeName, f.TypeName)
		}
		if tc.fixedLen > 0 {
			if f.FixedLen != tc.fixedLen {
				t.Errorf("field %s: expected FixedLen %d, got %d", tc.name, tc.fixedLen, f.FixedLen)
			}
			if f.Elem == nil {
				t.Errorf("field %s: Elem is nil", tc.name)
			} else if f.Elem.Kind != tc.elemKind {
				t.Errorf("field %s: Elem.Kind expected %d, got %d", tc.name, tc.elemKind, f.Elem.Kind)
			}
		}
		if tc.kind == KindSlice {
			if f.Elem == nil {
				t.Errorf("field %s: Elem is nil for slice", tc.name)
			} else if f.Elem.Kind != tc.elemKind {
				t.Errorf("field %s: Elem.Kind expected %d, got %d", tc.name, tc.elemKind, f.Elem.Kind)
			}
		}
	}
}

func TestParseSource_SpawnMessage(t *testing.T) {
	msgs, err := ParseSource(sampleSource)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	msg := msgs[2]
	if msg.Name != "SpawnMessage" {
		t.Fatalf("expected SpawnMessage, got %s", msg.Name)
	}

	// Tags: [4]string
	tags := msg.Fields[2]
	if tags.Kind != KindFixedArray || tags.FixedLen != 4 {
		t.Errorf("Tags: expected KindFixedArray[4], got kind=%d fixedLen=%d", tags.Kind, tags.FixedLen)
	}
	if tags.Elem == nil || tags.Elem.Primitive != KindString {
		t.Errorf("Tags: expected string element")
	}

	// Data: []uint8
	data := msg.Fields[3]
	if data.Kind != KindSlice {
		t.Errorf("Data: expected KindSlice, got %d", data.Kind)
	}
	if data.Elem == nil || data.Elem.Primitive != KindUint8 {
		t.Errorf("Data: expected uint8 element")
	}
}

func TestParseSchemaSource_Enums(t *testing.T) {
	schema, err := ParseSchemaSource(enumSource)
	if err != nil {
		t.Fatalf("ParseSchemaSource: %v", err)
	}

	if len(schema.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(schema.Messages))
	}
	if len(schema.Enums) != 1 {
		t.Fatalf("expected 1 enum, got %d", len(schema.Enums))
	}

	enum := schema.Enums[0]
	if enum.Name != "Opcode" {
		t.Fatalf("expected enum Opcode, got %s", enum.Name)
	}
	if enum.Primitive != KindUint16 {
		t.Fatalf("expected Opcode base kind uint16, got %d", enum.Primitive)
	}
	if len(enum.Values) != 3 {
		t.Fatalf("expected 3 enum values, got %d", len(enum.Values))
	}
	if enum.Values[1].Name != "OpcodeAuthorize" || enum.Values[1].Value != "1" {
		t.Fatalf("unexpected enum value %#v", enum.Values[1])
	}

	field := schema.Messages[0].Fields[0]
	if field.Kind != KindPrimitive {
		t.Fatalf("expected EnvelopeMessage.Code to be primitive, got %d", field.Kind)
	}
	if field.NamedType != "Opcode" {
		t.Fatalf("expected named type Opcode, got %q", field.NamedType)
	}
	if field.Primitive != KindUint16 {
		t.Fatalf("expected underlying uint16, got %d", field.Primitive)
	}
}

func TestQuantTag_Errors(t *testing.T) {
	cases := []struct {
		src     string
		wantErr bool
	}{
		{`package p; type T struct { X float32 ` + "`" + `pack:"min=0,max=100"` + "`" + ` }`, false},
		{`package p; type T struct { X float32 ` + "`" + `pack:"min=100,max=0"` + "`" + ` }`, true},         // max < min
		{`package p; type T struct { X float32 ` + "`" + `pack:"min=0,max=100,bits=32"` + "`" + ` }`, true}, // bad bits
		{`package p; type T struct { X int32 ` + "`" + `pack:"min=0,max=100"` + "`" + ` }`, true},           // quant на int
		{`package p; type T struct { X float32 ` + "`" + `pack:"foo=1"` + "`" + ` }`, true},                 // unknown key
	}

	for i, tc := range cases {
		_, err := ParseSource(tc.src)
		if tc.wantErr && err == nil {
			t.Errorf("[%d] expected error, got nil", i)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("[%d] unexpected error: %v", i, err)
		}
	}
}

func TestWireSize(t *testing.T) {
	msgs, err := ParseSource(sampleSource)
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}

	v3 := msgs[0]
	// Vector3: 3 × uint16 (квантизованные float32) = 6 байт
	if v3.MinWireSize() != 6 {
		t.Errorf("Vector3.MinWireSize: expected 6, got %d", v3.MinWireSize())
	}
	if v3.HasVariableFields() {
		t.Errorf("Vector3 should not have variable fields")
	}
}

func TestNestedUnknownType(t *testing.T) {
	src := `package p
type Msg struct {
	Pos UnknownType
}
`
	_, err := ParseSource(src)
	if err == nil {
		t.Fatal("expected error for unknown nested type, got nil")
	}
}

func TestUnsupportedPlatformDependentIntTypes(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			name: "direct int field",
			src: `package p
type Msg struct {
	X int
}
`,
			wantErr: `platform-dependent type "int" is not supported`,
		},
		{
			name: "direct uint field",
			src: `package p
type Msg struct {
	X uint
}
`,
			wantErr: `platform-dependent type "uint" is not supported`,
		},
		{
			name: "direct uintptr field",
			src: `package p
type Msg struct {
	X uintptr
}
`,
			wantErr: `platform-dependent type "uintptr" is not supported`,
		},
		{
			name: "alias of int",
			src: `package p
type Counter int
type Msg struct {
	X Counter
}
`,
			wantErr: `type "Counter" aliases unsupported platform-dependent "int"`,
		},
		{
			name: "alias of uint",
			src: `package p
type Counter uint
type Msg struct {
	X Counter
}
`,
			wantErr: `type "Counter" aliases unsupported platform-dependent "uint"`,
		},
		{
			name: "alias of uintptr",
			src: `package p
type Handle uintptr
type Msg struct {
	X Handle
}
`,
			wantErr: `type "Handle" aliases unsupported platform-dependent "uintptr"`,
		},
		{
			name: "transitive alias of int",
			src: `package p
type Base int
type Counter Base
type Msg struct {
	X Counter
}
`,
			wantErr: `type "Counter" aliases unsupported platform-dependent "int"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseSource(tc.src)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}
