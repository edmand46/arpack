package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edmand46/arpack/parser"
)

func TestGenerateTypeScript_Primitives(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				PackageName: "test",
				Name:        "PrimitiveMessage",
				Fields: []parser.Field{
					{Name: "F32", Kind: parser.KindPrimitive, Primitive: parser.KindFloat32},
					{Name: "F64", Kind: parser.KindPrimitive, Primitive: parser.KindFloat64},
					{Name: "I8", Kind: parser.KindPrimitive, Primitive: parser.KindInt8},
					{Name: "I16", Kind: parser.KindPrimitive, Primitive: parser.KindInt16},
					{Name: "I32", Kind: parser.KindPrimitive, Primitive: parser.KindInt32},
					{Name: "I64", Kind: parser.KindPrimitive, Primitive: parser.KindInt64},
					{Name: "U8", Kind: parser.KindPrimitive, Primitive: parser.KindUint8},
					{Name: "U16", Kind: parser.KindPrimitive, Primitive: parser.KindUint16},
					{Name: "U32", Kind: parser.KindPrimitive, Primitive: parser.KindUint32},
					{Name: "U64", Kind: parser.KindPrimitive, Primitive: parser.KindUint64},
					{Name: "B", Kind: parser.KindPrimitive, Primitive: parser.KindBool},
					{Name: "S", Kind: parser.KindPrimitive, Primitive: parser.KindString},
				},
			},
		},
	}

	src, err := GenerateTypeScriptSchema(schema)
	if err != nil {
		t.Fatalf("GenerateTypeScriptSchema: %v", err)
	}

	code := string(src)

	// Check field declarations (now using camelCase)
	if !strings.Contains(code, "f32: number = 0;") {
		t.Error("Missing f32 field")
	}
	if !strings.Contains(code, "i64: bigint = 0n;") {
		t.Error("Missing i64 field with bigint type")
	}
	if !strings.Contains(code, "u64: bigint = 0n;") {
		t.Error("Missing u64 field with bigint type")
	}
	if !strings.Contains(code, "b: boolean = false;") {
		t.Error("Missing b field")
	}
	if !strings.Contains(code, "s: string = \"\";") {
		t.Error("Missing s field")
	}

	// Check serialize method exists
	if !strings.Contains(code, "serialize(view: DataView, offset: number): number") {
		t.Error("Missing serialize method")
	}

	// Check deserialize method exists
	if !strings.Contains(code, "static deserialize(view: DataView, offset: number): [PrimitiveMessage, number]") {
		t.Error("Missing deserialize method")
	}
}

func TestGenerateTypeScript_QuantizedFloats(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				PackageName: "test",
				Name:        "QuantMessage",
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

	src, err := GenerateTypeScriptSchema(schema)
	if err != nil {
		t.Fatalf("GenerateTypeScriptSchema: %v", err)
	}

	code := string(src)

	// Check 8-bit quantization (using camelCase field names)
	if !strings.Contains(code, "Math.trunc((this.q8 - (0)) / (100 - (0)) * 255)") {
		t.Error("Missing 8-bit quantization code")
	}
	if !strings.Contains(code, `arpackEnsureQuantizedRange(this.q8, 0, 100, "Q8");`) {
		t.Error("Missing 8-bit quantized range guard")
	}

	// Check 16-bit quantization (using camelCase field names)
	if !strings.Contains(code, "Math.trunc((this.q16 - (-500)) / (500 - (-500)) * 65535)") {
		t.Error("Missing 16-bit quantization code")
	}
	if !strings.Contains(code, `arpackEnsureQuantizedRange(this.q16, -500, 500, "Q16");`) {
		t.Error("Missing 16-bit quantized range guard")
	}
	// Check deserialization with dequantization
	if !strings.Contains(code, "/ 255) * (100 - (0)) + (0)") {
		t.Error("Missing 8-bit dequantization")
	}
	if !strings.Contains(code, "/ 65535) * (500 - (-500)) + (-500)") {
		t.Error("Missing 16-bit dequantization")
	}
}

func TestGenerateTypeScript_BoolPacking(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				PackageName: "test",
				Name:        "BoolMessage",
				Fields: []parser.Field{
					{Name: "A", Kind: parser.KindPrimitive, Primitive: parser.KindBool},
					{Name: "B", Kind: parser.KindPrimitive, Primitive: parser.KindBool},
					{Name: "C", Kind: parser.KindPrimitive, Primitive: parser.KindBool},
					{Name: "X", Kind: parser.KindPrimitive, Primitive: parser.KindUint32},
					{Name: "D", Kind: parser.KindPrimitive, Primitive: parser.KindBool},
					{Name: "E", Kind: parser.KindPrimitive, Primitive: parser.KindBool},
				},
			},
		},
	}

	src, err := GenerateTypeScriptSchema(schema)
	if err != nil {
		t.Fatalf("GenerateTypeScriptSchema: %v", err)
	}

	code := string(src)

	// Check that consecutive bools are packed (using camelCase field names)
	if !strings.Contains(code, "let _boolByte0 = 0;") {
		t.Error("Missing first bool group packing")
	}
	if !strings.Contains(code, "if (this.a) _boolByte0 |= 1 << 0;") {
		t.Error("Missing a bool packing")
	}
	if !strings.Contains(code, "if (this.b) _boolByte0 |= 1 << 1;") {
		t.Error("Missing b bool packing")
	}
	if !strings.Contains(code, "if (this.c) _boolByte0 |= 1 << 2;") {
		t.Error("Missing c bool packing")
	}

	// Check second bool group after uint32 (index is 2, not 4, based on segment index)
	if !strings.Contains(code, "let _boolByte2 = 0;") {
		t.Error("Missing second bool group packing")
	}

	// Check deserialization (using camelCase field names)
	if !strings.Contains(code, "msg.a = (_boolByte0 & (1 << 0)) !== 0;") {
		t.Error("Missing a bool unpacking")
	}
}

func TestGenerateTypeScript_NestedTypes(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				PackageName: "test",
				Name:        "Inner",
				Fields: []parser.Field{
					{Name: "Value", Kind: parser.KindPrimitive, Primitive: parser.KindInt32},
				},
			},
			{
				PackageName: "test",
				Name:        "Outer",
				Fields: []parser.Field{
					{Name: "Inner", Kind: parser.KindNested, TypeName: "Inner"},
				},
			},
		},
	}

	src, err := GenerateTypeScriptSchema(schema)
	if err != nil {
		t.Fatalf("GenerateTypeScriptSchema: %v", err)
	}

	code := string(src)

	// Check nested type default value (using camelCase field name)
	if !strings.Contains(code, "inner: Inner = new Inner();") {
		t.Error("Missing nested type field with default")
	}

	// Check serialize calls nested serialize (using camelCase field name)
	if !strings.Contains(code, "pos += this.inner.serialize(view, pos);") {
		t.Error("Missing nested serialize call")
	}

	// Check deserialize calls nested deserialize
	if !strings.Contains(code, "const [_dvInner, _dnInner] = Inner.deserialize(view, pos);") {
		t.Error("Missing nested deserialize call")
	}
}

func TestGenerateTypeScript_FixedArrays(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				PackageName: "test",
				Name:        "ArrayMessage",
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

	src, err := GenerateTypeScriptSchema(schema)
	if err != nil {
		t.Fatalf("GenerateTypeScriptSchema: %v", err)
	}

	code := string(src)

	// Check default value (using camelCase field name)
	if !strings.Contains(code, "values: number[] = new Array<number>(3).fill(0);") {
		t.Error("Missing fixed array field with default")
	}

	// Check serialization loop (using camelCase field name)
	if !strings.Contains(code, "for (let _iValues = 0; _iValues < 3; _iValues++)") {
		t.Error("Missing fixed array serialization loop")
	}

	// Check deserialization loop (using camelCase field name)
	if !strings.Contains(code, "msg.values = new Array(3);") {
		t.Error("Missing fixed array allocation in deserialize")
	}
}

func TestGenerateTypeScript_Slices(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				PackageName: "test",
				Name:        "SliceMessage",
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

	src, err := GenerateTypeScriptSchema(schema)
	if err != nil {
		t.Fatalf("GenerateTypeScriptSchema: %v", err)
	}

	code := string(src)

	// Check default value (using camelCase field name)
	if !strings.Contains(code, "items: number[] = [];") {
		t.Error("Missing slice field with default")
	}

	// Check length prefix in serialize (using camelCase field name)
	if !strings.Contains(code, `arpackEnsureUint16Length(this.items.length, "slice length for Items")`) {
		t.Error("Missing slice length guard in serialize")
	}
	if !strings.Contains(code, "view.setUint16(pos, _lenthis_items, true);") {
		t.Error("Missing guarded slice length prefix in serialize")
	}

	// Check length reading in deserialize
	if !strings.Contains(code, "const _lenItems = view.getUint16(pos, true);") {
		t.Error("Missing slice length reading in deserialize")
	}

	// Check array allocation in deserialize (using camelCase field name)
	if !strings.Contains(code, "msg.items = new Array(_lenItems);") {
		t.Error("Missing slice allocation in deserialize")
	}
}

func TestGenerateTypeScript_Enums(t *testing.T) {
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
				PackageName: "test",
				Name:        "EnumMessage",
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

	src, err := GenerateTypeScriptSchema(schema)
	if err != nil {
		t.Fatalf("GenerateTypeScriptSchema: %v", err)
	}

	code := string(src)

	// Check enum definition
	if !strings.Contains(code, "export enum Status {") {
		t.Error("Missing enum definition")
	}
	if !strings.Contains(code, "Pending = 0,") {
		t.Error("Missing Pending enum value")
	}
	if !strings.Contains(code, "Active = 1,") {
		t.Error("Missing Active enum value")
	}

	// Check enum field type (using camelCase field name)
	if !strings.Contains(code, "status: Status = 0;") {
		t.Error("Missing enum field with correct type")
	}

	// Check enum serialization (cast to number, using camelCase field name)
	if !strings.Contains(code, "view.setUint16(pos, this.status as number, true);") {
		t.Error("Missing enum cast in serialize")
	}

	// Check enum deserialization (cast from number, using camelCase field name)
	if !strings.Contains(code, "msg.status = (view.getUint16(pos, true) as Status);") {
		t.Error("Missing enum cast in deserialize")
	}
}

func TestGenerateTypeScript_Strings(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				PackageName: "test",
				Name:        "StringMessage",
				Fields: []parser.Field{
					{Name: "Name", Kind: parser.KindPrimitive, Primitive: parser.KindString},
				},
			},
		},
	}

	src, err := GenerateTypeScriptSchema(schema)
	if err != nil {
		t.Fatalf("GenerateTypeScriptSchema: %v", err)
	}

	code := string(src)

	// Check TextEncoder usage
	if !strings.Contains(code, "const arpackTextEncoder = new TextEncoder();") {
		t.Error("Missing shared TextEncoder helper")
	}

	// Check length prefix
	if !strings.Contains(code, "view.setUint16(pos, _slen") {
		t.Error("Missing string length prefix in serialize")
	}
	if !strings.Contains(code, `arpackEnsureUint16Length(_slen`) {
		t.Error("Missing string length guard in serialize")
	}

	// Check TextDecoder usage
	if !strings.Contains(code, "const arpackTextDecoder = new TextDecoder();") {
		t.Error("Missing shared TextDecoder helper")
	}
	if !strings.Contains(code, "arpackTextDecoder.decode(") {
		t.Error("Missing shared TextDecoder in deserialize")
	}
}

func TestGenerateTypeScript_LengthAndRangeHelpers(t *testing.T) {
	schema := parser.Schema{
		Messages: []parser.Message{
			{
				PackageName: "test",
				Name:        "LengthAndQuant",
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
					{
						Name:      "Ratio",
						Kind:      parser.KindPrimitive,
						Primitive: parser.KindFloat32,
						Quant:     &parser.QuantInfo{Min: 0, Max: 1, Bits: 8},
					},
				},
			},
		},
	}

	src, err := GenerateTypeScriptSchema(schema)
	if err != nil {
		t.Fatalf("GenerateTypeScriptSchema: %v", err)
	}

	code := string(src)

	if !strings.Contains(code, "function arpackEnsureUint16Length(length: number, context: string): number") {
		t.Error("Missing uint16 length helper")
	}
	if !strings.Contains(code, "function arpackEnsureQuantizedRange(value: number, min: number, max: number, context: string): void") {
		t.Error("Missing quantized range helper")
	}
	if !strings.Contains(code, `arpackEnsureUint16Length(this.items.length, "slice length for Items")`) {
		t.Error("Missing slice length guard")
	}
	if !strings.Contains(code, `arpackEnsureUint16Length(_slen`) {
		t.Error("Missing string length helper call")
	}
	if !strings.Contains(code, `arpackEnsureQuantizedRange(this.ratio, 0, 1, "Ratio");`) {
		t.Error("Missing quantized range helper call")
	}
}

func TestGenerateTypeScript_RuntimeGuards(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not found")
	}
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm not found")
	}

	schema := parser.Schema{
		Messages: []parser.Message{
			{
				PackageName: "test",
				Name:        "Guarded",
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
		},
	}

	src, err := GenerateTypeScriptSchema(schema)
	if err != nil {
		t.Fatalf("GenerateTypeScriptSchema: %v", err)
	}

	out := runGeneratedTypeScriptProgram(t, src, `
import { Guarded } from "./messages.gen";

function emit(label: string, fn: () => void) {
  try {
    fn();
    console.log(label + ":OK");
  } catch (err) {
    if (err instanceof Error) {
      console.log(label + ":" + err.name + ":" + err.message);
      return;
    }
    console.log(label + ":" + String(err));
  }
}

emit("TRUNC", () => {
  Guarded.deserialize(new DataView(new ArrayBuffer(0)), 0);
});

emit("LEN", () => {
  const msg = new Guarded();
  msg.name = "a".repeat(65536);
  msg.serialize(new DataView(new ArrayBuffer(2)), 0);
});

emit("QUANT", () => {
  const msg = new Guarded();
  msg.ratio = 2;
  msg.serialize(new DataView(new ArrayBuffer(4)), 0);
});
`)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 output lines, got %d: %q", len(lines), out)
	}
	if !strings.Contains(lines[0], "RangeError:arpack: buffer too short for string length for Name") {
		t.Fatalf("expected truncated-input guard, got %q", lines[0])
	}
	if !strings.Contains(lines[1], "RangeError:arpack: string length for Name exceeds uint16 limit") {
		t.Fatalf("expected string length guard, got %q", lines[1])
	}
	if !strings.Contains(lines[2], "RangeError:arpack: quantized value out of range for Ratio") {
		t.Fatalf("expected quantized range guard, got %q", lines[2])
	}
}

func runGeneratedTypeScriptProgram(t *testing.T, generatedSrc []byte, mainSrc string) string {
	t.Helper()

	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", srcDir, err)
	}

	if err := os.WriteFile(filepath.Join(srcDir, "messages.gen.ts"), generatedSrc, 0o600); err != nil {
		t.Fatalf("write generated source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "main.ts"), []byte(mainSrc), 0o600); err != nil {
		t.Fatalf("write main source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
	  "name": "arpack-ts-runtime-test",
	  "private": true,
	  "dependencies": {
	    "typescript": "^5.6.3"
	  }
	}`), 0o600); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte(`{
	  "compilerOptions": {
	    "target": "ES2020",
	    "module": "CommonJS",
	    "moduleResolution": "node",
	    "outDir": "dist",
	    "strict": true,
	    "skipLibCheck": true
	  },
	  "include": ["src/**/*.ts"]
	}`), 0o600); err != nil {
		t.Fatalf("write tsconfig.json: %v", err)
	}

	npmInstall := exec.Command("npm", "install")
	npmInstall.Dir = dir
	if out, err := npmInstall.CombinedOutput(); err != nil {
		t.Fatalf("npm install failed: %v\n%s", err, out)
	}

	tsc := exec.Command("npx", "tsc")
	tsc.Dir = dir
	if out, err := tsc.CombinedOutput(); err != nil {
		t.Fatalf("tsc failed: %v\n%s", err, out)
	}

	run := exec.Command("node", filepath.Join("dist", "main.js"))
	run.Dir = dir
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("node failed: %v\n%s", err, out)
	}

	return string(out)
}
