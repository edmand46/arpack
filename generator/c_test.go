package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edmand46/arpack/parser"
)

func TestCSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"Simple", "simple"},
		{"PlayerID", "player_id"},
		{"HTTPRequest", "http_request"},
		{"XMLParser", "xml_parser"},
		{"MoveMessage", "move_message"},
		{"position", "position"},
		{"X", "x"},
		{"HTTPServer", "http_server"},
		{"URLHandler", "url_handler"},
	}

	for _, tc := range tests {
		result := snakeCase(tc.input)
		if result != tc.expected {
			t.Errorf("snakeCase(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestCGenerateSchema_BasicTypes(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "BasicTypes",
				Fields: []parser.Field{
					{Name: "Int8Field", Kind: parser.KindPrimitive, Primitive: parser.KindInt8},
					{Name: "Int16Field", Kind: parser.KindPrimitive, Primitive: parser.KindInt16},
					{Name: "Int32Field", Kind: parser.KindPrimitive, Primitive: parser.KindInt32},
					{Name: "Int64Field", Kind: parser.KindPrimitive, Primitive: parser.KindInt64},
					{Name: "Uint8Field", Kind: parser.KindPrimitive, Primitive: parser.KindUint8},
					{Name: "Uint16Field", Kind: parser.KindPrimitive, Primitive: parser.KindUint16},
					{Name: "Uint32Field", Kind: parser.KindPrimitive, Primitive: parser.KindUint32},
					{Name: "Uint64Field", Kind: parser.KindPrimitive, Primitive: parser.KindUint64},
					{Name: "Float32Field", Kind: parser.KindPrimitive, Primitive: parser.KindFloat32},
					{Name: "Float64Field", Kind: parser.KindPrimitive, Primitive: parser.KindFloat64},
					{Name: "BoolField", Kind: parser.KindPrimitive, Primitive: parser.KindBool},
					{Name: "StringField", Kind: parser.KindPrimitive, Primitive: parser.KindString},
				},
			},
		},
	}

	header, _, err := GenerateCSchema(schema, "test")
	if err != nil {
		t.Fatalf("GenerateCSchema failed: %v", err)
	}

	headerStr := string(header)

	// Check struct declaration
	if !strings.Contains(headerStr, "typedef struct test_basic_types {") {
		t.Error("Missing test_basic_types struct declaration")
	}

	// Check primitive field types
	if !strings.Contains(headerStr, "int8_t int8_field;") {
		t.Error("Missing int8_field")
	}
	if !strings.Contains(headerStr, "int16_t int16_field;") {
		t.Error("Missing int16_field")
	}
	if !strings.Contains(headerStr, "int32_t int32_field;") {
		t.Error("Missing int32_field")
	}
	if !strings.Contains(headerStr, "int64_t int64_field;") {
		t.Error("Missing int64_field")
	}
	if !strings.Contains(headerStr, "uint8_t uint8_field;") {
		t.Error("Missing uint8_field")
	}
	if !strings.Contains(headerStr, "uint16_t uint16_field;") {
		t.Error("Missing uint16_field")
	}
	if !strings.Contains(headerStr, "uint32_t uint32_field;") {
		t.Error("Missing uint32_field")
	}
	if !strings.Contains(headerStr, "uint64_t uint64_field;") {
		t.Error("Missing uint64_field")
	}
	if !strings.Contains(headerStr, "float float32_field;") {
		t.Error("Missing float32_field")
	}
	if !strings.Contains(headerStr, "double float64_field;") {
		t.Error("Missing float64_field")
	}
	if !strings.Contains(headerStr, "bool bool_field;") {
		t.Error("Missing bool_field")
	}
	if !strings.Contains(headerStr, "arpack_string_view string_field;") {
		t.Error("Missing string_field")
	}
}

func TestCGenerateSchema_Enum(t *testing.T) {
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

	header, _, err := GenerateCSchema(schema, "test")
	if err != nil {
		t.Fatalf("GenerateCSchema failed: %v", err)
	}

	headerStr := string(header)

	// Check enum declaration
	if !strings.Contains(headerStr, "typedef enum test_opcode {") {
		t.Error("Missing test_opcode enum declaration")
	}
	if !strings.Contains(headerStr, "test_opcode_unknown = 0,") {
		t.Error("Missing Unknown enum value")
	}
	if !strings.Contains(headerStr, "test_opcode_join = 1,") {
		t.Error("Missing Join enum value")
	}
	if !strings.Contains(headerStr, "test_opcode_leave = 2,") {
		t.Error("Missing Leave enum value")
	}
	if !strings.Contains(headerStr, "test_opcode op;") {
		t.Error("Enum-backed field should use generated enum type")
	}
}

func TestCGenerateSchema_HeaderGuard(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{Name: "Simple", Fields: []parser.Field{}},
		},
	}

	header, _, err := GenerateCSchema(schema, "my_base")
	if err != nil {
		t.Fatalf("GenerateCSchema failed: %v", err)
	}

	headerStr := string(header)

	if !strings.Contains(headerStr, "#ifndef MY_BASE_GEN_H") {
		t.Error("Missing header guard ifndef")
	}
	if !strings.Contains(headerStr, "#define MY_BASE_GEN_H") {
		t.Error("Missing header guard define")
	}
	if !strings.Contains(headerStr, "#endif // MY_BASE_GEN_H") {
		t.Error("Missing header guard endif")
	}
}

func TestCGenerateSchema_RuntimeTypes(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{Name: "Simple", Fields: []parser.Field{}},
		},
	}

	header, _, err := GenerateCSchema(schema, "test")
	if err != nil {
		t.Fatalf("GenerateCSchema failed: %v", err)
	}

	headerStr := string(header)

	// Check arpack_status enum
	if !strings.Contains(headerStr, "typedef enum arpack_status {") {
		t.Error("Missing arpack_status enum")
	}
	if !strings.Contains(headerStr, "ARPACK_OK = 0,") {
		t.Error("Missing ARPACK_OK")
	}

	// Check string view
	if !strings.Contains(headerStr, "typedef struct arpack_string_view {") {
		t.Error("Missing arpack_string_view")
	}

	// Check bytes view
	if !strings.Contains(headerStr, "typedef struct arpack_bytes_view {") {
		t.Error("Missing arpack_bytes_view")
	}

	// Check standard includes
	if !strings.Contains(headerStr, "#include <stdint.h>") {
		t.Error("Missing stdint.h include")
	}
	if !strings.Contains(headerStr, "#include <stddef.h>") {
		t.Error("Missing stddef.h include")
	}
	if !strings.Contains(headerStr, "#include <stdbool.h>") {
		t.Error("Missing stdbool.h include")
	}
}

func TestCGenerateSchema_NestedMessages(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "Inner",
				Fields: []parser.Field{
					{Name: "Value", Kind: parser.KindPrimitive, Primitive: parser.KindInt32},
				},
			},
			{
				Name: "Outer",
				Fields: []parser.Field{
					{Name: "InnerMsg", Kind: parser.KindNested, TypeName: "Inner"},
				},
			},
		},
	}

	header, _, err := GenerateCSchema(schema, "test")
	if err != nil {
		t.Fatalf("GenerateCSchema failed: %v", err)
	}

	headerStr := string(header)

	// Check inner struct
	if !strings.Contains(headerStr, "typedef struct test_inner {") {
		t.Error("Missing test_inner struct")
	}

	// Check outer struct with nested field
	if !strings.Contains(headerStr, "test_inner inner_msg;") {
		t.Error("Missing nested inner_msg field")
	}
}

func TestCGenerateSchema_FixedArrays(t *testing.T) {
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
				},
			},
		},
	}

	header, _, err := GenerateCSchema(schema, "test")
	if err != nil {
		t.Fatalf("GenerateCSchema failed: %v", err)
	}

	headerStr := string(header)

	if !strings.Contains(headerStr, "float values[3];") {
		t.Error("Missing fixed array field")
	}
}

func TestCGenerateSchema_BoolPacking(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "BoolMessage",
				Fields: []parser.Field{
					{Name: "Active", Kind: parser.KindPrimitive, Primitive: parser.KindBool},
					{Name: "Visible", Kind: parser.KindPrimitive, Primitive: parser.KindBool},
					{Name: "Ghost", Kind: parser.KindPrimitive, Primitive: parser.KindBool},
					{Name: "Count", Kind: parser.KindPrimitive, Primitive: parser.KindInt32},
				},
			},
		},
	}

	header, source, err := GenerateCSchema(schema, "test")
	if err != nil {
		t.Fatalf("GenerateCSchema failed: %v", err)
	}

	headerStr := string(header)
	sourceStr := string(source)

	// Check struct fields
	if !strings.Contains(headerStr, "bool active;") {
		t.Error("Missing active field")
	}
	if !strings.Contains(headerStr, "bool visible;") {
		t.Error("Missing visible field")
	}

	// Check encode uses bool packing
	if !strings.Contains(sourceStr, "_boolByte") {
		t.Error("Bool packing not used in encode")
	}
	if !strings.Contains(sourceStr, "_arpack_write_u8(buf, &offset, _boolByte);") {
		t.Error("Bool byte not written")
	}
}

func TestCGenerateSchema_QuantizedFloats(t *testing.T) {
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

	_, source, err := GenerateCSchema(schema, "test")
	if err != nil {
		t.Fatalf("GenerateCSchema failed: %v", err)
	}

	sourceStr := string(source)

	// Check 8-bit quantization (uses 255 as max value)
	if !strings.Contains(sourceStr, "255") {
		t.Error("Missing 8-bit quantization")
	}

	// Check 16-bit quantization (uses 65535 as max value)
	if !strings.Contains(sourceStr, "65535") {
		t.Error("Missing 16-bit quantization")
	}

	// Check encode uses quantization
	if !strings.Contains(sourceStr, "_arpack_write_u8(buf, &offset, _qv)") {
		t.Error("8-bit quantized value not written")
	}
	if !strings.Contains(sourceStr, "_arpack_write_u16_le(buf, &offset, _qv)") {
		t.Error("16-bit quantized value not written")
	}
}

func TestCGenerateSchema_VariableLength(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "VarMessage",
				Fields: []parser.Field{
					{Name: "Id", Kind: parser.KindPrimitive, Primitive: parser.KindUint32},
					{Name: "Name", Kind: parser.KindPrimitive, Primitive: parser.KindString},
					{
						Name: "Data",
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

	header, _, err := GenerateCSchema(schema, "test")
	if err != nil {
		t.Fatalf("GenerateCSchema failed: %v", err)
	}

	headerStr := string(header)

	// Check min_size function exists
	if !strings.Contains(headerStr, "size_t test_var_message_min_size(void);") {
		t.Error("Missing min_size function declaration")
	}

	// Check size function exists
	if !strings.Contains(headerStr, "arpack_status test_var_message_size(const test_var_message *msg, size_t *out_size);") {
		t.Error("Missing size function declaration")
	}

	// Check encode function exists
	if !strings.Contains(headerStr, "arpack_status test_var_message_encode(const test_var_message *msg, uint8_t *buf, size_t buf_len, size_t *out_written);") {
		t.Error("Missing encode function declaration")
	}

	// Check decode function exists without context (only byte slices and strings)
	if !strings.Contains(headerStr, "arpack_status test_var_message_decode(test_var_message *msg, const uint8_t *buf, size_t buf_len, size_t *out_read);") {
		t.Error("Missing decode function declaration")
	}
}

func TestCCompile_SampleSchema(t *testing.T) {
	hasCC := false
	if _, err := exec.LookPath("cc"); err == nil {
		hasCC = true
	} else if _, err := exec.LookPath("gcc"); err == nil {
		hasCC = true
	} else if _, err := exec.LookPath("clang"); err == nil {
		hasCC = true
	}

	if !hasCC {
		t.Skip("No C compiler found (tried cc, gcc, clang)")
	}

	schema, err := parser.ParseSchemaFile("../testdata/sample.go")
	if err != nil {
		t.Fatalf("Failed to parse sample.go: %v", err)
	}

	// Generate C code
	header, source, err := GenerateCSchema(schema, "sample")
	if err != nil {
		t.Fatalf("GenerateCSchema failed: %v", err)
	}

	// Create temp directory
	tmpDir := t.TempDir()

	// Write header
	headerPath := filepath.Join(tmpDir, "sample.gen.h")
	if err := os.WriteFile(headerPath, header, 0644); err != nil {
		t.Fatalf("Failed to write header: %v", err)
	}

	// Write source
	sourcePath := filepath.Join(tmpDir, "sample.gen.c")
	if err := os.WriteFile(sourcePath, source, 0644); err != nil {
		t.Fatalf("Failed to write source: %v", err)
	}

	// Compile
	objPath := filepath.Join(tmpDir, "sample.gen.o")
	cmd := exec.Command("cc", "-std=c11", "-Wall", "-Wextra", "-Wno-unused-function", "-c", sourcePath, "-o", objPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("C compilation failed:\n%s\n%s", string(output), err)
	}

	// Verify object file exists
	if _, err := os.Stat(objPath); os.IsNotExist(err) {
		t.Fatal("Object file was not created")
	}
}

func TestCGenerateSchema_DecodeContext(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "Inner",
				Fields: []parser.Field{
					{Name: "Value", Kind: parser.KindPrimitive, Primitive: parser.KindInt32},
				},
			},
			{
				Name: "CtxMessage",
				Fields: []parser.Field{
					{
						Name: "Items",
						Kind: parser.KindSlice,
						Elem: &parser.Field{
							Kind:     parser.KindNested,
							TypeName: "Inner",
						},
					},
				},
			},
		},
	}

	header, _, err := GenerateCSchema(schema, "test")
	if err != nil {
		t.Fatalf("GenerateCSchema failed: %v", err)
	}

	headerStr := string(header)

	// Check decode context struct exists
	if !strings.Contains(headerStr, "typedef struct test_ctx_message_decode_ctx {") {
		t.Error("Missing decode context struct")
	}

	// Check data pointer in context
	if !strings.Contains(headerStr, "test_inner *items_data;") {
		t.Error("Missing items_data field in context")
	}

	// Check capacity field in context
	if !strings.Contains(headerStr, "uint16_t items_cap;") {
		t.Error("Missing items_cap field in context")
	}

	// Check decode function with context
	if !strings.Contains(headerStr, "arpack_status test_ctx_message_decode(test_ctx_message *msg, const uint8_t *buf, size_t buf_len, test_ctx_message_decode_ctx *ctx, size_t *out_read);") {
		t.Error("Missing decode function with context")
	}
}

func TestCGenerateSchema_PrimitiveSlices(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "SliceMessage",
				Fields: []parser.Field{
					{
						Name: "Values",
						Kind: parser.KindSlice,
						Elem: &parser.Field{
							Kind:      parser.KindPrimitive,
							Primitive: parser.KindUint16,
						},
					},
					{
						Name: "Floats",
						Kind: parser.KindSlice,
						Elem: &parser.Field{
							Kind:      parser.KindPrimitive,
							Primitive: parser.KindFloat32,
						},
					},
				},
			},
		},
	}

	header, source, err := GenerateCSchema(schema, "test")
	if err != nil {
		t.Fatalf("GenerateCSchema failed: %v", err)
	}

	headerStr := string(header)
	if !strings.Contains(headerStr, "typedef struct test_uint16_slice_view {") {
		t.Fatal("Missing uint16 slice view typedef")
	}
	if !strings.Contains(headerStr, "const uint16_t *data;") {
		t.Fatal("uint16 slice view should reference uint16_t")
	}
	if !strings.Contains(headerStr, "typedef struct test_float32_slice_view {") {
		t.Fatal("Missing float32 slice view typedef")
	}
	if !strings.Contains(headerStr, "const float *data;") {
		t.Fatal("float32 slice view should reference float")
	}
	if !strings.Contains(headerStr, "uint16_t *values_data;") {
		t.Fatal("Missing decode context storage for uint16 slice")
	}
	if !strings.Contains(headerStr, "float *floats_data;") {
		t.Fatal("Missing decode context storage for float32 slice")
	}

	compileCGeneratedObject(t, "test", header, source)
}

func TestCGenerateSchema_FixedArrayNestedAndQuantized(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "Vector3",
				Fields: []parser.Field{
					{
						Name:      "X",
						Kind:      parser.KindPrimitive,
						Primitive: parser.KindFloat32,
						Quant:     &parser.QuantInfo{Min: -500, Max: 500, Bits: 16},
					},
				},
			},
			{
				Name: "ArrayMessage",
				Fields: []parser.Field{
					{
						Name:     "Points",
						Kind:     parser.KindFixedArray,
						FixedLen: 2,
						Elem: &parser.Field{
							Kind:     parser.KindNested,
							TypeName: "Vector3",
						},
					},
					{
						Name:     "Samples",
						Kind:     parser.KindFixedArray,
						FixedLen: 3,
						Elem: &parser.Field{
							Kind:      parser.KindPrimitive,
							Primitive: parser.KindFloat32,
							Quant:     &parser.QuantInfo{Min: 0, Max: 10, Bits: 8},
						},
					},
				},
			},
		},
	}

	header, source, err := GenerateCSchema(schema, "test")
	if err != nil {
		t.Fatalf("GenerateCSchema failed: %v", err)
	}

	sourceStr := string(source)
	if !strings.Contains(sourceStr, "test_vector3_encode(&msg->points[_i0],") {
		t.Fatal("Nested fixed array elements should call nested encode")
	}
	if !strings.Contains(sourceStr, "msg->samples[_i0]") {
		t.Fatal("Quantized fixed array elements should be encoded through recursive element access")
	}

	compileCGeneratedObject(t, "test", header, source)
}

func TestCVariableLength_BoundsChecks(t *testing.T) {
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

	header, source, err := GenerateCSchema(schema, "test")
	if err != nil {
		t.Fatalf("GenerateCSchema failed: %v", err)
	}

	harness := `#include <stdio.h>
#include "test.gen.h"

int main(void) {
    static const uint8_t truncated[] = {0x03, 0x00, 'a'};
    test_string_message decoded;
    size_t read = 0;
    arpack_status status = test_string_message_decode(&decoded, truncated, sizeof(truncated), &read);
    printf("DECODE=%d\n", (int)status);

    test_string_message encoded;
    encoded.name.data = "abc";
    encoded.name.len = 3;
    uint8_t out[2];
    size_t written = 0;
    status = test_string_message_encode(&encoded, out, sizeof(out), &written);
    printf("ENCODE=%d\n", (int)status);
    return 0;
}
`

	output := runGeneratedCProgram(t, "test", header, source, harness)
	if !strings.Contains(output, "DECODE=1") {
		t.Fatalf("decode should fail with ARPACK_ERR_BUFFER_TOO_SHORT, got:\n%s", output)
	}
	if !strings.Contains(output, "ENCODE=1") {
		t.Fatalf("encode should fail with ARPACK_ERR_BUFFER_TOO_SHORT, got:\n%s", output)
	}
}

func TestCGenerateSchema_RejectsFixedArrayOfSlices(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				Name: "BadMessage",
				Fields: []parser.Field{
					{
						Name:     "Values",
						Kind:     parser.KindFixedArray,
						FixedLen: 2,
						Elem: &parser.Field{
							Kind: parser.KindSlice,
							Elem: &parser.Field{
								Kind:      parser.KindPrimitive,
								Primitive: parser.KindUint16,
							},
						},
					},
				},
			},
		},
	}

	_, _, err := GenerateCSchema(schema, "test")
	if err == nil {
		t.Fatal("expected GenerateCSchema to reject fixed arrays of slices")
	}
	if !strings.Contains(err.Error(), "fixed arrays of slices") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func requireCCompiler(t *testing.T) string {
	t.Helper()

	for _, compiler := range []string{"cc", "gcc", "clang"} {
		if _, err := exec.LookPath(compiler); err == nil {
			return compiler
		}
	}

	t.Skip("No C compiler found (tried cc, gcc, clang)")
	return ""
}

func compileCGeneratedObject(t *testing.T, base string, header []byte, source []byte) {
	t.Helper()

	cc := requireCCompiler(t)
	tmpDir := t.TempDir()
	headerPath := filepath.Join(tmpDir, base+".gen.h")
	sourcePath := filepath.Join(tmpDir, base+".gen.c")
	objPath := filepath.Join(tmpDir, base+".gen.o")

	if err := os.WriteFile(headerPath, header, 0644); err != nil {
		t.Fatalf("Failed to write header: %v", err)
	}
	if err := os.WriteFile(sourcePath, source, 0644); err != nil {
		t.Fatalf("Failed to write source: %v", err)
	}

	cmd := exec.Command(cc, "-std=c11", "-Wall", "-Wextra", "-Wno-unused-function", "-c", sourcePath, "-o", objPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("C compilation failed:\n%s\n%s", string(output), err)
	}
}

func runGeneratedCProgram(t *testing.T, base string, header []byte, source []byte, harness string) string {
	t.Helper()

	cc := requireCCompiler(t)
	tmpDir := t.TempDir()
	headerPath := filepath.Join(tmpDir, base+".gen.h")
	sourcePath := filepath.Join(tmpDir, base+".gen.c")
	testPath := filepath.Join(tmpDir, "test.c")
	binPath := filepath.Join(tmpDir, "test")

	if err := os.WriteFile(headerPath, header, 0644); err != nil {
		t.Fatalf("Failed to write header: %v", err)
	}
	if err := os.WriteFile(sourcePath, source, 0644); err != nil {
		t.Fatalf("Failed to write source: %v", err)
	}
	if err := os.WriteFile(testPath, []byte(harness), 0644); err != nil {
		t.Fatalf("Failed to write harness: %v", err)
	}

	cmd := exec.Command(cc, "-std=c11", "-Wall", "-Wextra", "-Wno-unused-function", "-o", binPath, testPath, sourcePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("C compilation failed:\n%s\n%s", string(output), err)
	}

	runCmd := exec.Command(binPath)
	runCmd.Dir = tmpDir
	output, err = runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("C program failed:\n%s\n%s", string(output), err)
	}
	return string(output)
}
