package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edmand46/arpack/parser"
)

func TestGenerateLua_BasicTypes(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "BasicTypes",
				Fields: []parser.Field{
					{Name: "Int8Field", Kind: parser.KindPrimitive, Primitive: parser.KindInt8},
					{Name: "Int16Field", Kind: parser.KindPrimitive, Primitive: parser.KindInt16},
					{Name: "Int32Field", Kind: parser.KindPrimitive, Primitive: parser.KindInt32},
					{Name: "Uint8Field", Kind: parser.KindPrimitive, Primitive: parser.KindUint8},
					{Name: "Uint16Field", Kind: parser.KindPrimitive, Primitive: parser.KindUint16},
					{Name: "Uint32Field", Kind: parser.KindPrimitive, Primitive: parser.KindUint32},
					{Name: "Float32Field", Kind: parser.KindPrimitive, Primitive: parser.KindFloat32},
					{Name: "Float64Field", Kind: parser.KindPrimitive, Primitive: parser.KindFloat64},
					{Name: "BoolField", Kind: parser.KindPrimitive, Primitive: parser.KindBool},
					{Name: "StringField", Kind: parser.KindPrimitive, Primitive: parser.KindString},
				},
			},
		},
	}

	lua, err := GenerateLuaSchema(schema, "test")
	if err != nil {
		t.Fatalf("GenerateLuaSchema failed: %v", err)
	}

	luaStr := string(lua)

	if !strings.Contains(luaStr, "function M.new_basic_types()") {
		t.Error("Missing constructor for BasicTypes")
	}
	if !strings.Contains(luaStr, "function M.serialize_basic_types(msg)") {
		t.Error("Missing serializer for BasicTypes")
	}
	if !strings.Contains(luaStr, "function M.deserialize_basic_types(data, offset)") {
		t.Error("Missing deserializer for BasicTypes")
	}

	if !strings.Contains(luaStr, "int8_field = 0") {
		t.Error("Missing int8_field in constructor")
	}
	if !strings.Contains(luaStr, "string_field = ''") {
		t.Error("Missing string_field default value")
	}
	if !strings.Contains(luaStr, "bool_field = false") {
		t.Error("Missing bool_field default value")
	}
}

func TestGenerateLua_Enum(t *testing.T) {
	schema := parser.Schema{
		Enums: []parser.Enum{
			{
				Name:      "Opcode",
				Primitive: parser.KindUint16,
				Values: []parser.EnumValue{
					{Name: "Unknown", Value: "0"},
					{Name: "Join", Value: "1"},
					{Name: "Leave", Value: "2"},
				},
			},
		},
		Messages: []parser.Message{
			{
				Name: "MessageWithEnum",
				Fields: []parser.Field{
					{Name: "Op", Kind: parser.KindPrimitive, Primitive: parser.KindUint16, NamedType: "Opcode"},
				},
			},
		},
	}

	enumNames := map[string]struct{}{"Opcode": {}}
	_ = enumNames

	lua, err := GenerateLuaSchema(schema, "test")
	if err != nil {
		t.Fatalf("GenerateLuaSchema failed: %v", err)
	}

	luaStr := string(lua)

	if !strings.Contains(luaStr, "M.Opcode = {") {
		t.Error("Missing Opcode enum table")
	}
	if !strings.Contains(luaStr, "Unknown = 0") {
		t.Error("Missing Unknown enum value")
	}
	if !strings.Contains(luaStr, "Join = 1") {
		t.Error("Missing Join enum value")
	}
}

func TestGenerateLua_NestedMessage(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "Vector3",
				Fields: []parser.Field{
					{Name: "X", Kind: parser.KindPrimitive, Primitive: parser.KindFloat32},
					{Name: "Y", Kind: parser.KindPrimitive, Primitive: parser.KindFloat32},
					{Name: "Z", Kind: parser.KindPrimitive, Primitive: parser.KindFloat32},
				},
			},
			{
				Name: "Player",
				Fields: []parser.Field{
					{Name: "Position", Kind: parser.KindNested, TypeName: "Vector3"},
					{Name: "Health", Kind: parser.KindPrimitive, Primitive: parser.KindInt32},
				},
			},
		},
	}

	lua, err := GenerateLuaSchema(schema, "test")
	if err != nil {
		t.Fatalf("GenerateLuaSchema failed: %v", err)
	}

	luaStr := string(lua)

	if !strings.Contains(luaStr, "function M.new_vector3()") {
		t.Error("Missing constructor for Vector3")
	}
	if !strings.Contains(luaStr, "function M.new_player()") {
		t.Error("Missing constructor for Player")
	}
	if !strings.Contains(luaStr, "position = M.new_vector3()") {
		t.Error("Missing nested initialization in Player constructor")
	}
	if !strings.Contains(luaStr, "M.serialize_vector3") {
		t.Error("Missing Vector3 serializer call")
	}
}

func TestGenerateLua_FixedArray(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "WithFixedArray",
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
				},
			},
		},
	}

	lua, err := GenerateLuaSchema(schema, "test")
	if err != nil {
		t.Fatalf("GenerateLuaSchema failed: %v", err)
	}

	luaStr := string(lua)

	if !strings.Contains(luaStr, "values = {}") {
		t.Error("Missing values array initialization")
	}
	if !strings.Contains(luaStr, "for _i_values = 1, 3 do") {
		t.Error("Missing fixed array loop in serializer")
	}
}

func TestGenerateLua_Slice(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "WithSlice",
				Fields: []parser.Field{
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

	lua, err := GenerateLuaSchema(schema, "test")
	if err != nil {
		t.Fatalf("GenerateLuaSchema failed: %v", err)
	}

	luaStr := string(lua)

	if !strings.Contains(luaStr, "items = {}") {
		t.Error("Missing items slice initialization")
	}
	if !strings.Contains(luaStr, "local _len_items = #(msg.items or {})") {
		t.Error("Missing slice length serialization")
	}
	if !strings.Contains(luaStr, "for _i_items = 1, _len_items do") {
		t.Error("Missing slice iteration in serializer")
	}
}

func TestGenerateLua_BoolPacking(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "WithBools",
				Fields: []parser.Field{
					{Name: "A", Kind: parser.KindPrimitive, Primitive: parser.KindBool},
					{Name: "B", Kind: parser.KindPrimitive, Primitive: parser.KindBool},
					{Name: "C", Kind: parser.KindPrimitive, Primitive: parser.KindBool},
					{Name: "Value", Kind: parser.KindPrimitive, Primitive: parser.KindInt32},
				},
			},
		},
	}

	lua, err := GenerateLuaSchema(schema, "test")
	if err != nil {
		t.Fatalf("GenerateLuaSchema failed: %v", err)
	}

	luaStr := string(lua)

	if !strings.Contains(luaStr, "local _bool_byte_0 = 0") {
		t.Error("Missing bool byte packing variable")
	}
	if !strings.Contains(luaStr, "if msg.a then _bool_byte_0 = bit.bor(_bool_byte_0, 1) end") {
		t.Error("Missing first bool packing check with bit.bor")
	}
	if !strings.Contains(luaStr, "if msg.b then _bool_byte_0 = bit.bor(_bool_byte_0, 2) end") {
		t.Error("Missing second bool packing check with bit.bor")
	}
	if !strings.Contains(luaStr, "msg.a = bit.band(_bool_byte_0, 1) ~= 0") {
		t.Error("Missing bit.band for bool deserialization")
	}
}

func TestGenerateLua_QuantizedFloat(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "WithQuantized",
				Fields: []parser.Field{
					{
						Name:      "Position",
						Kind:      parser.KindPrimitive,
						Primitive: parser.KindFloat32,
						Quant: &parser.QuantInfo{
							Min:  -500,
							Max:  500,
							Bits: 16,
						},
					},
				},
			},
		},
	}

	lua, err := GenerateLuaSchema(schema, "test")
	if err != nil {
		t.Fatalf("GenerateLuaSchema failed: %v", err)
	}

	luaStr := string(lua)

	if !strings.Contains(luaStr, "math.floor") {
		t.Error("Missing math.floor for quantization")
	}
	if !strings.Contains(luaStr, "write_u16_le") {
		t.Error("Missing u16 write for 16-bit quantization")
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"A", "a"},
		{"AB", "ab"},
		{"AbCd", "ab_cd"},
		{"ABC", "abc"},
		{"PlayerID", "player_id"},
		{"HTTPResponse", "http_response"},
		{"XMLHttpRequest", "xml_http_request"},
		{"getHTTPResponse", "get_http_response"},
	}

	for _, tt := range tests {
		result := toSnakeCase(tt.input)
		if result != tt.expected {
			t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestLuaHelpersGenerated(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name:   "Empty",
				Fields: []parser.Field{},
			},
		},
	}

	lua, err := GenerateLuaSchema(schema, "test")
	if err != nil {
		t.Fatalf("GenerateLuaSchema failed: %v", err)
	}

	luaStr := string(lua)

	helpers := []string{
		"local bit = require('bit')",
		"buffer too short for u8",
		"buffer too short for bool",
		"local function write_u8(n)",
		"buffer too short for u16",
		"local function write_u16_le(n)",
		"buffer too short for u32",
		"local function write_u32_le(n)",
		"local function read_f32_le(data, offset)",
		"local function write_f32_le(n)",
		"local function read_f64_le(data, offset)",
		"local function write_f64_le(n)",
		"local function write_bool(v)",
		"buffer too short for string",
		"local function write_string(s)",
	}

	for _, helper := range helpers {
		if !strings.Contains(luaStr, helper) {
			t.Errorf("Missing helper: %s", helper)
		}
	}
}

func TestGenerateLua_Int64NotSupported(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "WithInt64",
				Fields: []parser.Field{
					{Name: "Value", Kind: parser.KindPrimitive, Primitive: parser.KindInt64},
				},
			},
		},
	}

	_, err := GenerateLuaSchema(schema, "test")
	if err == nil {
		t.Fatal("Expected error for int64 field, got nil")
	}
	if !strings.Contains(err.Error(), "int64/uint64") {
		t.Errorf("Expected error mentioning int64/uint64, got: %v", err)
	}
	if !strings.Contains(err.Error(), "LuaJIT/Defold") {
		t.Errorf("Expected error mentioning LuaJIT/Defold, got: %v", err)
	}
}

func TestGenerateLua_Uint64NotSupported(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "WithUint64",
				Fields: []parser.Field{
					{Name: "Value", Kind: parser.KindPrimitive, Primitive: parser.KindUint64},
				},
			},
		},
	}

	_, err := GenerateLuaSchema(schema, "test")
	if err == nil {
		t.Fatal("Expected error for uint64 field, got nil")
	}
	if !strings.Contains(err.Error(), "int64/uint64") {
		t.Errorf("Expected error mentioning int64/uint64, got: %v", err)
	}
}

func TestGenerateLua_Int64InSliceNotSupported(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "WithInt64Slice",
				Fields: []parser.Field{
					{
						Name: "Values",
						Kind: parser.KindSlice,
						Elem: &parser.Field{
							Kind:      parser.KindPrimitive,
							Primitive: parser.KindInt64,
						},
					},
				},
			},
		},
	}

	_, err := GenerateLuaSchema(schema, "test")
	if err == nil {
		t.Fatal("Expected error for int64 in slice, got nil")
	}
	if !strings.Contains(err.Error(), "int64/uint64") {
		t.Errorf("Expected error mentioning int64/uint64, got: %v", err)
	}
}

func TestGenerateLua_BoundsChecks(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "SimpleMessage",
				Fields: []parser.Field{
					{Name: "ID", Kind: parser.KindPrimitive, Primitive: parser.KindUint32},
					{Name: "Name", Kind: parser.KindPrimitive, Primitive: parser.KindString},
				},
			},
		},
	}

	lua, err := GenerateLuaSchema(schema, "test")
	if err != nil {
		t.Fatalf("GenerateLuaSchema failed: %v", err)
	}

	luaStr := string(lua)

	// Check that bounds check function exists
	if !strings.Contains(luaStr, "check_bounds") {
		t.Error("Missing check_bounds function")
	}

	// Check that read_u16_le has bounds check
	if !strings.Contains(luaStr, "buffer too short for u16") {
		t.Error("Missing bounds check in read_u16_le")
	}

	// Check that read_u32_le has bounds check
	if !strings.Contains(luaStr, "buffer too short for u32") {
		t.Error("Missing bounds check in read_u32_le")
	}

	// Check that read_string has bounds check
	if !strings.Contains(luaStr, "buffer too short for string") {
		t.Error("Missing bounds check in read_string")
	}

	// Check that deserialize function has min size check (message name is preserved in error)
	if !strings.Contains(luaStr, "buffer too short for SimpleMessage") {
		t.Error("Missing min size check in deserialize function")
	}

	// Check that read_u8 has bounds check
	if !strings.Contains(luaStr, "buffer too short for u8") {
		t.Error("Missing bounds check in read_u8")
	}

	// Check that read_bool has bounds check
	if !strings.Contains(luaStr, "buffer too short for bool") {
		t.Error("Missing bounds check in read_bool")
	}

	// Check that read_i8 has bounds check
	if !strings.Contains(luaStr, "buffer too short for i8") {
		t.Error("Missing bounds check in read_i8")
	}
}

func TestGenerateLua_RuntimeFloatEdgeCases(t *testing.T) {
	if _, err := exec.LookPath("luajit"); err != nil {
		t.Skip("luajit not found")
	}

	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "FloatEdges",
				Fields: []parser.Field{
					{Name: "F32", Kind: parser.KindPrimitive, Primitive: parser.KindFloat32},
					{Name: "F64", Kind: parser.KindPrimitive, Primitive: parser.KindFloat64},
				},
			},
		},
	}

	lua, err := GenerateLuaSchema(schema, "messages")
	if err != nil {
		t.Fatalf("GenerateLuaSchema failed: %v", err)
	}

	dir := t.TempDir()
	modulePath := filepath.Join(dir, "messages_gen.lua")
	if err := os.WriteFile(modulePath, lua, 0o600); err != nil {
		t.Fatalf("write module: %v", err)
	}

	scriptPath := filepath.Join(dir, "check.lua")
	script := `local messages = require("messages_gen")

local function bytes_to_hex(s)
    return (s:gsub(".", function(c) return string.format("%02x", string.byte(c)) end))
end

local neg_zero = string.char(0, 0, 0, 128, 0, 0, 0, 0, 0, 0, 0, 128)
local msg = messages.deserialize_float_edges(neg_zero, 1)
print(bytes_to_hex(messages.serialize_float_edges(msg)))

local subnormal = string.char(1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0)
msg = messages.deserialize_float_edges(subnormal, 1)
print(bytes_to_hex(messages.serialize_float_edges(msg)))
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cmd := exec.Command("luajit", "check.lua")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("luajit failed: %v\n%s", err, out)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 output lines, got %d: %q", len(lines), string(out))
	}

	if lines[0] != "000000800000000000000080" {
		t.Fatalf("negative zero roundtrip mismatch: %s", lines[0])
	}
	if lines[1] != "010000000100000000000000" {
		t.Fatalf("subnormal roundtrip mismatch: %s", lines[1])
	}
}
