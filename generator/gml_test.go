package generator

import (
	"strings"
	"testing"

	"github.com/edmand46/arpack/parser"
)

func TestGenerateGML_Sample(t *testing.T) {
	schema, err := parser.ParseSchemaFile("../testdata/sample.go")
	if err != nil {
		t.Fatalf("ParseSchemaFile: %v", err)
	}

	src, err := GenerateGMLSchema(schema)
	if err != nil {
		t.Fatalf("GenerateGMLSchema: %v", err)
	}

	code := string(src)
	for _, want := range []string{
		"function Vector3() constructor",
		"static serialize = function(_buf)",
		"Vector3.deserialize = function(_buf)",
		"MoveMessage.deserialize = function(_buf)",
		"SpawnMessage.deserialize = function(_buf)",
		"EnvelopeMessage.deserialize = function(_buf)",
		"function arpack_ensure_readable(_buf, _needed, _context)",
		"function arpack_ensure_u16_length(_length, _context)",
		"function arpack_ensure_quant_range(_value, _min, _max, _context)",
		"enum Opcode",
		"OpcodeUnknown = 0",
		"OpcodeJoinRoom = 2",
		"buffer_write(_buf, buffer_u16, self.code)",
		"buffer_write(_buf, buffer_u64, self.entity_id)",
		"buffer_read(_buf, buffer_u8)",
		"buffer_resize(_buf, buffer_tell(_buf));",
		"buffer_seek(_buf, buffer_seek_start, 0);",
		"return [_msg, buffer_tell(_buf) - _start];",
	} {
		if !strings.Contains(code, want) {
			t.Errorf("missing %q in generated GML", want)
		}
	}

	if !strings.Contains(code, "self.position = new Vector3();") {
		t.Error("missing nested constructor default with type")
	}
	if !strings.Contains(code, "var _msg = new MoveMessage();") {
		t.Error("missing typed constructor call in deserialize")
	}
	if !strings.Contains(code, "self.position.serialize(_buf);") {
		t.Error("missing nested serialize call")
	}
	if !strings.Contains(code, "_msg.position = Vector3.deserialize(_buf)[0];") {
		t.Error("missing nested deserialize call")
	}
}

func TestGenerateGML_QuantizedFloats(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "QuantMessage",
				Fields: []parser.Field{
					{
						Name:      "Q8",
						Kind:      parser.KindPrimitive,
						Primitive: parser.KindFloat32,
						Quant:     &parser.QuantInfo{Min: 0, Max: 100, Bits: 8},
					},
					{
						Name:      "Q16",
						Kind:      parser.KindPrimitive,
						Primitive: parser.KindFloat32,
						Quant:     &parser.QuantInfo{Min: -500, Max: 500, Bits: 16},
					},
				},
			},
		},
	}

	src, err := GenerateGMLSchema(schema)
	if err != nil {
		t.Fatalf("GenerateGMLSchema: %v", err)
	}

	code := string(src)
	if !strings.Contains(code, "arpack_ensure_quant_range(self.q8, 0, 100, \"Q8\");") {
		t.Error("missing 8-bit quantized range guard")
	}
	if !strings.Contains(code, "var _q_self_q8 = (floor((self.q8 - (0)) / (100 - (0)) * 255));") {
		t.Error("missing 8-bit quantization expression")
	}
	if !strings.Contains(code, "var _q_self_q16 = (floor((self.q16 - (-500)) / (500 - (-500)) * 65535));") {
		t.Error("missing 16-bit quantization expression")
	}
	if !strings.Contains(code, "_msg.q8 = (_q_msg_q8 / 255) * (100 - (0)) + (0);") {
		t.Error("missing 8-bit dequantization")
	}
	if !strings.Contains(code, "_msg.q16 = (_q_msg_q16 / 65535) * (500 - (-500)) + (-500);") {
		t.Error("missing 16-bit dequantization")
	}
	if !strings.Contains(code, "buffer_write(_buf, buffer_u8, _q_self_q8);") {
		t.Error("missing 8-bit write")
	}
	if !strings.Contains(code, "buffer_write(_buf, buffer_u16, _q_self_q16);") {
		t.Error("missing 16-bit write")
	}
}

func TestGenerateGML_BoolPacking(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "BoolMessage",
				Fields: []parser.Field{
					{Name: "A", Kind: parser.KindPrimitive, Primitive: parser.KindBool},
					{Name: "B", Kind: parser.KindPrimitive, Primitive: parser.KindBool},
					{Name: "X", Kind: parser.KindPrimitive, Primitive: parser.KindUint32},
					{Name: "D", Kind: parser.KindPrimitive, Primitive: parser.KindBool},
				},
			},
		},
	}

	src, err := GenerateGMLSchema(schema)
	if err != nil {
		t.Fatalf("GenerateGMLSchema: %v", err)
	}

	code := string(src)
	if !strings.Contains(code, "var _bool_byte_0 = 0;") {
		t.Error("missing first bool group packing")
	}
	if !strings.Contains(code, "if (self.a) _bool_byte_0 |= 1 << 0;") {
		t.Error("missing a bool packing")
	}
	if !strings.Contains(code, "if (self.b) _bool_byte_0 |= 1 << 1;") {
		t.Error("missing b bool packing")
	}
	if !strings.Contains(code, "var _bool_byte_2 = 0;") {
		t.Error("missing second bool group after uint32")
	}
	if !strings.Contains(code, "_msg.a = (_bool_byte_0 & (1 << 0)) != 0;") {
		t.Error("missing a bool unpacking")
	}
}

func TestGenerateGML_FixedArraysAndSlices(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "ArrayMessage",
				Fields: []parser.Field{
					{
						Name:     "Values",
						Kind:     parser.KindFixedArray,
						FixedLen: 3,
						Elem: &parser.Field{
							Kind:      parser.KindPrimitive,
							Primitive: parser.KindFloat32,
						},
					},
					{
						Name: "Items",
						Kind: parser.KindSlice,
						Elem: &parser.Field{
							Kind:      parser.KindPrimitive,
							Primitive: parser.KindInt32,
						},
					},
				},
			},
		},
	}

	src, err := GenerateGMLSchema(schema)
	if err != nil {
		t.Fatalf("GenerateGMLSchema: %v", err)
	}

	code := string(src)
	if !strings.Contains(code, "self.values = array_create(3, 0);") {
		t.Error("missing fixed array default")
	}
	if !strings.Contains(code, "if (array_length(self.values) != 3) show_error(\"arpack: fixed array for Values length mismatch: expected 3, got \" + string(array_length(self.values)), true);") {
		t.Error("missing fixed array length guard")
	}
	if !strings.Contains(code, "for (var _ivalues = 0; _ivalues < 3; _ivalues++)") {
		t.Error("missing fixed array serialize loop")
	}
	if !strings.Contains(code, "var _lenitems = array_length(self.items);") {
		t.Error("missing slice length in serialize")
	}
	if !strings.Contains(code, "_lenitems = arpack_ensure_u16_length(_lenitems, \"slice length for Items\");") {
		t.Error("missing slice length guard")
	}
	if !strings.Contains(code, "buffer_write(_buf, buffer_u16, _lenitems);") {
		t.Error("missing slice length prefix")
	}
	if !strings.Contains(code, "var _lenitems = buffer_read(_buf, buffer_u16);") {
		t.Error("missing slice length read")
	}
	if !strings.Contains(code, "_msg.items = array_create(_lenitems);") {
		t.Error("missing slice allocation")
	}
	if !strings.Contains(code, "_msg.values = array_create(3);") {
		t.Error("missing fixed array allocation in deserialize")
	}
}

func TestGenerateGML_Strings(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "StringMessage",
				Fields: []parser.Field{
					{Name: "Name", Kind: parser.KindPrimitive, Primitive: parser.KindString},
				},
			},
		},
	}

	src, err := GenerateGMLSchema(schema)
	if err != nil {
		t.Fatalf("GenerateGMLSchema: %v", err)
	}

	code := string(src)
	if !strings.Contains(code, "var _slen_self_name = string_byte_length(self.name);") {
		t.Error("missing string byte length in serialize")
	}
	if !strings.Contains(code, "var _slen_checked_self_name = arpack_ensure_u16_length(_slen_self_name, \"string length for Name\");") {
		t.Error("missing string length guard")
	}
	if !strings.Contains(code, "buffer_write(_buf, buffer_u16, _slen_checked_self_name);") {
		t.Error("missing string length prefix")
	}
	if !strings.Contains(code, "buffer_write(_buf, buffer_text, self.name);") {
		t.Error("missing string bytes write")
	}
	if !strings.Contains(code, "var _slen_msg_name = buffer_read(_buf, buffer_u16);") {
		t.Error("missing string length read")
	}
	if !strings.Contains(code, "_s += chr(buffer_read(_buf, buffer_u8));") {
		t.Error("missing string byte loop")
	}
	if !strings.Contains(code, "_msg.name = _s;") {
		t.Error("missing string assign")
	}
}

func TestGenerateGML_Enums(t *testing.T) {
	schema := parser.Schema{
		Enums: []parser.Enum{
			{
				Name:      "Status",
				Primitive: parser.KindUint16,
				Values: []parser.EnumValue{
					{Name: "Pending", Value: "0"},
					{Name: "Active", Value: "1"},
					{Name: "Done", Value: "2"},
				},
			},
		},
		Messages: []parser.Message{
			{
				Name: "EnumMessage",
				Fields: []parser.Field{
					{
						Name:      "Status",
						Kind:      parser.KindPrimitive,
						Primitive: parser.KindUint16,
						NamedType: "Status",
					},
				},
			},
		},
	}

	src, err := GenerateGMLSchema(schema)
	if err != nil {
		t.Fatalf("GenerateGMLSchema: %v", err)
	}

	code := string(src)
	if !strings.Contains(code, "enum Status") {
		t.Error("missing enum definition")
	}
	if !strings.Contains(code, "Pending = 0") {
		t.Error("missing Pending enum value")
	}
	if !strings.Contains(code, "self.status = 0;") {
		t.Error("missing enum default")
	}
	if !strings.Contains(code, "buffer_write(_buf, buffer_u16, self.status);") {
		t.Error("missing enum serialization")
	}
}

func TestGenerateGML_RejectsUint64Enums(t *testing.T) {
	schema := parser.Schema{
		Enums: []parser.Enum{
			{
				Name:      "Wide",
				Primitive: parser.KindUint64,
				Values: []parser.EnumValue{
					{Name: "Big", Value: "9007199254740993"},
				},
			},
		},
	}

	_, err := GenerateGMLSchema(schema)
	if err == nil {
		t.Fatal("expected uint64 enum rejection")
	}
	if !strings.Contains(err.Error(), "int64/uint64 enum Wide") {
		t.Fatalf("expected clear uint64 enum error, got %v", err)
	}
}

func TestGenerateGML_HelpersOnlyWhenNeeded(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "Simple",
				Fields: []parser.Field{
					{Name: "ID", Kind: parser.KindPrimitive, Primitive: parser.KindUint32},
				},
			},
		},
	}

	src, err := GenerateGMLSchema(schema)
	if err != nil {
		t.Fatalf("GenerateGMLSchema: %v", err)
	}

	code := string(src)
	if !strings.Contains(code, "function arpack_ensure_readable(_buf, _needed, _context)") {
		t.Error("missing readable guard")
	}
	if strings.Contains(code, "arpack_ensure_u16_length") {
		t.Error("should not emit uint16 length guard without strings/slices")
	}
	if strings.Contains(code, "arpack_ensure_quant_range") {
		t.Error("should not emit quant range guard without quantized fields")
	}
}

func TestGenerateGML_Map(t *testing.T) {
	schema, err := parser.ParseSchemaSource(`package messages

type Vector3 struct {
	X float32
}

type MapMessage struct {
	ByName map[string]int32
	ByID   map[uint16]Vector3
}
`)
	if err != nil {
		t.Fatalf("ParseSchemaSource: %v", err)
	}
	src, err := GenerateGMLSchema(schema)
	if err != nil {
		t.Fatalf("GenerateGMLSchema: %v", err)
	}
	code := string(src)
	for _, want := range []string{
		"function arpack_compare_string_bytes(_a, _b)",
		"self.by_name = ds_map_create();",
		"var _keysbyname = ds_map_keys_to_array(self.by_name);",
		"array_sort(_keysbyname, arpack_compare_string_bytes);",
		"array_sort(_keysbyid, true);",
		`var _lenbyname = arpack_ensure_u16_length(array_length(_keysbyname), "map length for ByName");`,
		"var _v = self.by_id[? _k];",
		"_v.serialize(_buf);",
		`if (_ibyname > 0 && arpack_compare_string_bytes(_k, _prevbyname) <= 0) show_error("arpack: map keys out of order for ByName", true);`,
		`if (_ibyid > 0 && _k <= _prevbyid) show_error("arpack: map keys out of order for ByID", true);`,
		"_v = Vector3.deserialize(_buf)[0];",
		"ds_map_set(_msg.by_name, _k, _v);",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("missing %q in:\n%s", want, code)
		}
	}

	intOnly, err := parser.ParseSchemaSource(`package messages

type M struct {
	ByID map[uint16]int32
}
`)
	if err != nil {
		t.Fatalf("ParseSchemaSource: %v", err)
	}
	src2, err := GenerateGMLSchema(intOnly)
	if err != nil {
		t.Fatalf("GenerateGMLSchema: %v", err)
	}
	if strings.Contains(string(src2), "arpack_compare_string_bytes") {
		t.Fatal("byte compare helper emitted without string-keyed maps")
	}
}
