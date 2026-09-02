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
	if !strings.Contains(luaStr, "offset = offset or 1") {
		t.Error("Missing default offset in deserializer")
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

func TestGenerateLua_LuaKeywordFieldNames(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "KeywordFields",
				Fields: []parser.Field{
					{Name: "End", Kind: parser.KindPrimitive, Primitive: parser.KindUint8},
					{Name: "Local", Kind: parser.KindPrimitive, Primitive: parser.KindString},
					{Name: "Function", Kind: parser.KindPrimitive, Primitive: parser.KindBool},
					{Name: "Break", Kind: parser.KindPrimitive, Primitive: parser.KindBool},
				},
			},
		},
	}

	lua, err := GenerateLuaSchema(schema, "test")
	if err != nil {
		t.Fatalf("GenerateLuaSchema failed: %v", err)
	}

	luaStr := string(lua)

	required := []string{
		`["end"] = 0`,
		`["local"] = ''`,
		`["function"] = false`,
		`part_idx = part_idx + 1; parts[part_idx] = write_u8(msg["end"])`,
		`part_idx = part_idx + 1; parts[part_idx] = write_string(msg["local"] or '')`,
		`if msg["function"] then _bool_byte_2 = bit.bor(_bool_byte_2, 1) end`,
		`if msg["break"] then _bool_byte_2 = bit.bor(_bool_byte_2, 2) end`,
		`msg["function"] = bit.band(_bool_byte_2, 1) ~= 0`,
		`msg["break"] = bit.band(_bool_byte_2, 2) ~= 0`,
	}
	for _, want := range required {
		if !strings.Contains(luaStr, want) {
			t.Fatalf("generated Lua missing %q\n%s", want, luaStr)
		}
	}

	for _, invalid := range []string{"msg.end", "msg.local", "msg.function", "msg.break"} {
		if strings.Contains(luaStr, invalid) {
			t.Fatalf("generated Lua contains invalid keyword field access %q\n%s", invalid, luaStr)
		}
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

	if !strings.Contains(luaStr, "math.modf") {
		t.Error("Missing truncating quantization code for Lua, expected math.modf")
	}
	if !strings.Contains(luaStr, `ensure_quant_range(msg.position, -500, 500, "Position")`) {
		t.Error("Missing quantized range guard for Lua")
	}
	if strings.Contains(luaStr, "math.floor(((msg.position - (-500)) / (500 - (-500))) * 65535 + 0.5)") {
		t.Error("Lua quantization should not round to nearest")
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
		result := ToSnakeCase(tt.input)
		if result != tt.expected {
			t.Errorf("ToSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
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
		"local function ensure_u16_limit(n, context)",
		"local function ensure_quant_range(value, min, max, context)",
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

	if !strings.Contains(luaStr, "ensure_u16_limit") {
		t.Error("Missing uint16 overflow helper")
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

func TestGenerateLua_LengthOverflowGuards(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "LengthLimited",
				Fields: []parser.Field{
					{Name: "Name", Kind: parser.KindPrimitive, Primitive: parser.KindString},
					{
						Name: "Items",
						Kind: parser.KindSlice,
						Elem: &parser.Field{
							Kind:      parser.KindPrimitive,
							Primitive: parser.KindUint8,
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

	if !strings.Contains(luaStr, `ensure_u16_limit(len, "string length")`) {
		t.Error("Missing string length overflow guard")
	}

	if !strings.Contains(luaStr, `ensure_u16_limit(_len_items, "slice length for items")`) {
		t.Error("Missing slice length overflow guard")
	}
}

func TestGenerateLua_RuntimeLengthLimits(t *testing.T) {
	if _, err := exec.LookPath("luajit"); err != nil {
		t.Skip("luajit not found")
	}

	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "LengthLimited",
				Fields: []parser.Field{
					{Name: "Name", Kind: parser.KindPrimitive, Primitive: parser.KindString},
					{
						Name: "Items",
						Kind: parser.KindSlice,
						Elem: &parser.Field{
							Kind:      parser.KindPrimitive,
							Primitive: parser.KindUint8,
						},
					},
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

local function emit(label, ok, value)
    if ok then
        print(label .. ":OK")
    else
        print(label .. ":" .. tostring(value))
    end
end

local msg = messages.new_length_limited()

local ok, res = pcall(messages.serialize_length_limited, msg)
emit("EMPTY", ok, res)

msg.name = string.rep("a", 65535)
ok, res = pcall(messages.serialize_length_limited, msg)
emit("STR_MAX", ok, res)

msg.name = string.rep("a", 65536)
ok, res = pcall(messages.serialize_length_limited, msg)
emit("STR_OVER", ok, res)

msg.name = ""
msg.items = {}
for i = 1, 65535 do
    msg.items[i] = 0
end
ok, res = pcall(messages.serialize_length_limited, msg)
emit("SLICE_MAX", ok, res)

msg.items[65536] = 0
ok, res = pcall(messages.serialize_length_limited, msg)
emit("SLICE_OVER", ok, res)
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
	if len(lines) != 5 {
		t.Fatalf("expected 5 output lines, got %d: %q", len(lines), string(out))
	}

	if lines[0] != "EMPTY:OK" {
		t.Fatalf("expected empty serialization to succeed, got %q", lines[0])
	}
	if lines[1] != "STR_MAX:OK" {
		t.Fatalf("expected 65535-byte string serialization to succeed, got %q", lines[1])
	}
	if !strings.Contains(lines[2], "string length exceeds uint16 limit") {
		t.Fatalf("expected string overflow guard, got %q", lines[2])
	}
	if lines[3] != "SLICE_MAX:OK" {
		t.Fatalf("expected 65535-element slice serialization to succeed, got %q", lines[3])
	}
	if !strings.Contains(lines[4], "slice length for items exceeds uint16 limit") {
		t.Fatalf("expected slice overflow guard, got %q", lines[4])
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

func TestGenerateLua_RuntimeQuantizedRangeGuard(t *testing.T) {
	if _, err := exec.LookPath("luajit"); err != nil {
		t.Skip("luajit not found")
	}

	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "WithQuantized",
				Fields: []parser.Field{
					{
						Name:      "Position",
						Kind:      parser.KindPrimitive,
						Primitive: parser.KindFloat32,
						Quant:     &parser.QuantInfo{Min: -500, Max: 500, Bits: 16},
					},
				},
			},
		},
	}

	lua, err := GenerateLuaSchema(schema, "messages")
	if err != nil {
		t.Fatalf("GenerateLuaSchema failed: %v", err)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "messages_gen.lua"), lua, 0o600); err != nil {
		t.Fatalf("write module: %v", err)
	}

	script := `local messages = require("messages_gen")
local msg = messages.new_with_quantized()
msg.position = 501
local ok, res = pcall(messages.serialize_with_quantized, msg)
if ok then
    print("OK")
else
    print(res)
end
`
	if err := os.WriteFile(filepath.Join(dir, "check.lua"), []byte(script), 0o600); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cmd := exec.Command("luajit", "check.lua")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("luajit failed: %v\n%s", err, out)
	}

	got := strings.TrimSpace(string(out))
	if !strings.Contains(got, "quantized value out of range for Position") {
		t.Fatalf("expected quantized range guard, got %q", got)
	}
}

func TestGenerateLua_Map(t *testing.T) {
	schema, err := parser.ParseSchemaSource(`package messages

type MapMessage struct {
	ByName map[string]int32
	ByID   map[uint16]int32
}
`)
	if err != nil {
		t.Fatalf("ParseSchemaSource: %v", err)
	}
	lua, err := GenerateLuaSchema(schema, "messages")
	if err != nil {
		t.Fatalf("GenerateLuaSchema: %v", err)
	}
	code := string(lua)
	for _, want := range []string{
		"for _k in pairs(msg.by_name or {}) do _keys_by_name[#_keys_by_name + 1] = _k end",
		"table.sort(_keys_by_name)",
		`ensure_u16_limit(#_keys_by_name, "map length for by_name")`,
		`if _prev_by_name ~= nil and _k <= _prev_by_name then error("arpack: map keys out of order for by_name") end`,
		"msg.by_id[_k] = _v",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("missing %q in:\n%s", want, code)
		}
	}

	if _, err := exec.LookPath("luajit"); err != nil {
		t.Skip("luajit not found")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "messages_gen.lua"), lua, 0o600); err != nil {
		t.Fatalf("write module: %v", err)
	}
	script := `local messages = require("messages_gen")

local function hex(data)
    local out = {}
    for i = 1, #data do out[#out + 1] = string.format("%02x", string.byte(data, i)) end
    return table.concat(out)
end

local function report(label, ok, err)
    print(label .. ":" .. (ok and "OK" or tostring(err)))
end

local msg = messages.new_map_message()
msg.by_name = { b = 2, a = 1, ab = 3 }
msg.by_id = { [2] = 20, [1] = 10 }
local ok, wire = pcall(messages.serialize_map_message, msg)
local want = "0300" .. "010061" .. "01000000" .. "02006162" .. "03000000" .. "010062" .. "02000000" .. "0200" .. "0100" .. "0a000000" .. "0200" .. "14000000"
report("WIRE", ok and hex(wire) == want, ok and hex(wire) or wire)

local ok2, decoded = pcall(messages.deserialize_map_message, wire)
report("ROUNDTRIP", ok2 and decoded.by_name.ab == 3 and decoded.by_id[1] == 10 and decoded.by_id[2] == 20, decoded)

local ok3, err3 = pcall(messages.deserialize_map_message, string.char(2, 0, 1, 0, 0x62, 1, 0, 0, 0, 1, 0, 0x61, 2, 0, 0, 0, 0, 0))
report("UNSORTED", ok3, err3)
local ok4, err4 = pcall(messages.deserialize_map_message, string.char(2, 0, 1, 0, 0x61, 1, 0, 0, 0, 1, 0, 0x61, 2, 0, 0, 0, 0, 0))
report("DUPLICATE", ok4, err4)
local ok5, err5 = pcall(messages.deserialize_map_message, string.char(0, 0, 2, 0, 2, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0))
report("INT_UNSORTED", ok5, err5)
`
	if err := os.WriteFile(filepath.Join(dir, "check.lua"), []byte(script), 0o600); err != nil {
		t.Fatalf("write script: %v", err)
	}
	cmd := exec.Command("luajit", "check.lua")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("luajit failed: %v\n%s", err, out)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 output lines, got %d: %q", len(lines), string(out))
	}
	if lines[0] != "WIRE:OK" {
		t.Fatalf("keys not written in sorted order: %q", lines[0])
	}
	if lines[1] != "ROUNDTRIP:OK" {
		t.Fatalf("roundtrip failed: %q", lines[1])
	}
	for i, want := range []string{
		"UNSORTED:", "map keys out of order for by_name",
		"DUPLICATE:", "map keys out of order for by_name",
		"INT_UNSORTED:", "map keys out of order for by_id",
	} {
		line := lines[2+i/2]
		if !strings.Contains(line, want) {
			t.Fatalf("line %d: expected %q, got %q", 2+i/2, want, line)
		}
	}
}

func TestGenerateLua_Int64MapKeyNotSupported(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "WithInt64Key",
				Fields: []parser.Field{
					{
						Name: "Values",
						Kind: parser.KindMap,
						Key:  &parser.Field{Kind: parser.KindPrimitive, Primitive: parser.KindInt64},
						Elem: &parser.Field{Kind: parser.KindPrimitive, Primitive: parser.KindUint8},
					},
				},
			},
		},
	}
	_, err := GenerateLuaSchema(schema, "test")
	if err == nil {
		t.Fatal("expected error for int64 map key, got nil")
	}
	if !strings.Contains(err.Error(), "int64/uint64") {
		t.Errorf("expected error mentioning int64/uint64, got: %v", err)
	}
}
