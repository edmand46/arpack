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
				Name: "PrimitiveMessage",
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

	// Check serialize overloads exist
	if !strings.Contains(code, "serialize(buffer: Uint8Array): number;") {
		t.Error("Missing Uint8Array serialize overload")
	}
	if !strings.Contains(code, "serialize(view: DataView, offset: number): number") {
		t.Error("Missing DataView serialize overload")
	}

	// Check deserialize overloads exist
	if !strings.Contains(code, "static deserialize(data: Uint8Array): [PrimitiveMessage, number];") {
		t.Error("Missing Uint8Array deserialize overload")
	}
	if !strings.Contains(code, "static deserialize(view: DataView, offset: number): [PrimitiveMessage, number]") {
		t.Error("Missing DataView deserialize overload")
	}
}

func TestGenerateTypeScript_QuantizedFloats(t *testing.T) {
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
				Name: "BoolMessage",
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
				Name: "Inner",
				Fields: []parser.Field{
					{Name: "Value", Kind: parser.KindPrimitive, Primitive: parser.KindInt32},
				},
			},
			{
				Name: "Outer",
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
				Name: "SliceMessage",
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

func TestGenerateTypeScript_RejectsUint64Enums(t *testing.T) {
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
		Messages: []parser.Message{
			{
				Name: "EnumMessage",
				Fields: []parser.Field{
					{
						Name:      "Wide",
						Kind:      parser.KindPrimitive,
						Primitive: parser.KindUint64,
						NamedType: "Wide",
					},
				},
			},
		},
	}

	_, err := GenerateTypeScriptSchema(schema)
	if err == nil {
		t.Fatal("expected uint64 enum rejection")
	}
	if !strings.Contains(err.Error(), "int64/uint64 enum Wide") {
		t.Fatalf("expected clear uint64 enum error, got %v", err)
	}
}

func TestGenerateTypeScript_Strings(t *testing.T) {
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
	if !strings.Contains(code, `arpackEnsureWritable(view, pos, 2 + _slen`) {
		t.Error("Missing string write bounds guard in serialize")
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
				Name: "LengthAndQuant",
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
				Name: "Inner",
				Fields: []parser.Field{
					{Name: "Value", Kind: parser.KindPrimitive, Primitive: parser.KindInt32},
				},
			},
			{
				Name: "Holder",
				Fields: []parser.Field{
					{
						Name:     "Items",
						Kind:     parser.KindFixedArray,
						FixedLen: 2,
						Elem: &parser.Field{
							Kind:     parser.KindNested,
							TypeName: "Inner",
						},
					},
				},
			},
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
		},
	}

	src, err := GenerateTypeScriptSchema(schema)
	if err != nil {
		t.Fatalf("GenerateTypeScriptSchema: %v", err)
	}

	out := runGeneratedTypeScriptProgram(t, src, `
import { Guarded, Holder, Inner } from "./messages.gen";

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
  Guarded.deserialize(new Uint8Array(0));
});

emit("LEN", () => {
  const msg = new Guarded();
  msg.name = "a".repeat(65536);
  msg.serialize(new Uint8Array(2));
});

emit("QUANT", () => {
  const msg = new Guarded();
  msg.ratio = 2;
  msg.serialize(new Uint8Array(4));
});

emit("SLICE", () => {
  const backing = new Uint8Array(8);
  backing.fill(0x7f);
  const before = Array.from(backing).join(",");
  const msg = new Guarded();
  msg.name = "abc";
  try {
    msg.serialize(new DataView(backing.buffer, 2, 2), 0);
  } finally {
    const after = Array.from(backing).join(",");
    if (after !== before) {
      throw new Error("backing bytes changed: " + after);
    }
  }
});

emit("UINT8", () => {
  const msg = new Guarded();
  msg.name = "ok";
  msg.ratio = 0.5;
  const data = new Uint8Array(16);
  const n = msg.serialize(data);
  const [decoded, consumed] = Guarded.deserialize(data.subarray(0, n));
  if (consumed !== n || decoded.name !== "ok") {
    throw new Error("Uint8Array roundtrip failed");
  }
});

emit("DEFAULTS", () => {
  const holder = new Holder();
  if (holder.items[0] === holder.items[1]) {
    throw new Error("shared nested default");
  }
  holder.items[0].value = 7;
  if (holder.items[1].value === 7) {
    throw new Error("nested default mutation leaked");
  }
});

emit("FIXED_SHORT", () => {
  const holder = new Holder();
  holder.items = [new Inner()];
  holder.serialize(new Uint8Array(16));
});

emit("FIXED_LONG", () => {
  const holder = new Holder();
  holder.items = [new Inner(), new Inner(), new Inner()];
  holder.serialize(new Uint8Array(16));
});
`)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 8 {
		t.Fatalf("expected 8 output lines, got %d: %q", len(lines), out)
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
	if !strings.Contains(lines[3], "RangeError:arpack: buffer too short for string data for Name") {
		t.Fatalf("expected DataView-slice write guard, got %q", lines[3])
	}
	if lines[4] != "UINT8:OK" {
		t.Fatalf("expected Uint8Array roundtrip, got %q", lines[4])
	}
	if lines[5] != "DEFAULTS:OK" {
		t.Fatalf("expected distinct nested fixed-array defaults, got %q", lines[5])
	}
	if !strings.Contains(lines[6], "RangeError:arpack: fixed array for Items length mismatch: expected 2, got 1") {
		t.Fatalf("expected fixed-array short guard, got %q", lines[6])
	}
	if !strings.Contains(lines[7], "RangeError:arpack: fixed array for Items length mismatch: expected 2, got 3") {
		t.Fatalf("expected fixed-array long guard, got %q", lines[7])
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

func TestGenerateTypeScript_Map(t *testing.T) {
	schema, err := parser.ParseSchemaSource(`package messages

type MapMessage struct {
	ByName map[string]int32
	ByID   map[uint16]int32
	ByBig  map[int64]int32
}
`)
	if err != nil {
		t.Fatalf("ParseSchemaSource: %v", err)
	}
	src, err := GenerateTypeScriptSchema(schema)
	if err != nil {
		t.Fatalf("GenerateTypeScriptSchema: %v", err)
	}
	code := string(src)
	for _, want := range []string{
		"byName: Map<string, number> = new Map();",
		"byBig: Map<bigint, number> = new Map();",
		"function arpackCompareBytes(a: Uint8Array, b: Uint8Array): number",
		".sort((a, b) => arpackCompareBytes(a[0], b[0]));",
		"Array.from(this.byBig.keys()).sort((a, b) => (a < b ? -1 : a > b ? 1 : 0));",
		"let _prevByBig: bigint = 0n;",
		"arpack: map keys out of order for ByName",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("missing %q in:\n%s", want, code)
		}
	}

	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not found")
	}
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm not found")
	}

	out := runGeneratedTypeScriptProgram(t, src, `
import { MapMessage } from "./messages.gen";

function emit(label: string, fn: () => void) {
  try {
    fn();
    console.log(label + ":OK");
  } catch (err) {
    console.log(label + ":" + (err as Error).message);
  }
}

function hex(data: Uint8Array): string {
  return Array.from(data).map(b => b.toString(16).padStart(2, "0")).join("");
}

emit("ROUNDTRIP", () => {
  const msg = new MapMessage();
  msg.byName = new Map([["b", 2], ["a", 1], ["ab", 3]]);
  msg.byID = new Map([[2, 20], [1, 10]]);
  msg.byBig = new Map([[2n, 2], [-1n, -1], [10n, 10]]);
  const buf = new Uint8Array(128);
  const n = msg.serialize(buf);
  const want = "0300" + "010061" + "01000000" + "02006162" + "03000000" + "010062" + "02000000"
    + "0200" + "0100" + "0a000000" + "0200" + "14000000"
    + "0300" + "ffffffffffffffff" + "ffffffff" + "0200000000000000" + "02000000" + "0a00000000000000" + "0a000000";
  if (hex(buf.subarray(0, n)) !== want) throw new Error("wire mismatch: " + hex(buf.subarray(0, n)));
  const [decoded, consumed] = MapMessage.deserialize(buf.subarray(0, n));
  if (consumed !== n || decoded.byName.get("ab") !== 3 || decoded.byID.get(1) !== 10 || decoded.byBig.get(-1n) !== -1) throw new Error("roundtrip mismatch");
  const bigKeys = Array.from(decoded.byBig.keys()).join(",");
  if (bigKeys !== "-1,2,10") throw new Error("bigint order: " + bigKeys);
});

emit("ORDER", () => {
  const msg = new MapMessage();
  msg.byName = new Map([["\u{1F600}", 5], ["！", 4]]);
  const buf = new Uint8Array(64);
  const n = msg.serialize(buf);
  if (buf[4] !== 0xef) throw new Error("UTF-16 order leaked: " + hex(buf.subarray(0, n)));
  const [decoded] = MapMessage.deserialize(buf.subarray(0, n));
  if (decoded.byName.get("！") !== 4 || decoded.byName.get("\u{1F600}") !== 5) throw new Error("roundtrip mismatch");
});

emit("UNSORTED", () => {
  MapMessage.deserialize(new Uint8Array([2, 0, 1, 0, 0x62, 1, 0, 0, 0, 1, 0, 0x61, 2, 0, 0, 0, 0, 0, 0, 0]));
});

emit("DUPLICATE", () => {
  MapMessage.deserialize(new Uint8Array([2, 0, 1, 0, 0x61, 1, 0, 0, 0, 1, 0, 0x61, 2, 0, 0, 0, 0, 0, 0, 0]));
});

emit("INT_UNSORTED", () => {
  MapMessage.deserialize(new Uint8Array([0, 0, 2, 0, 2, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0]));
});
`)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 output lines, got %d: %q", len(lines), out)
	}
	if lines[0] != "ROUNDTRIP:OK" {
		t.Fatalf("roundtrip failed: %q", lines[0])
	}
	if lines[1] != "ORDER:OK" {
		t.Fatalf("utf-8 key order failed: %q", lines[1])
	}
	for i, want := range []string{
		"UNSORTED:arpack: map keys out of order for ByName",
		"DUPLICATE:arpack: map keys out of order for ByName",
		"INT_UNSORTED:arpack: map keys out of order for ByID",
	} {
		if lines[2+i] != want {
			t.Fatalf("line %d: expected %q, got %q", 2+i, want, lines[2+i])
		}
	}
}
