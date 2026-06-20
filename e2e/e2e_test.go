package e2e

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/edmand46/arpack/generator"
	"github.com/edmand46/arpack/parser"
)

const samplePath = "../testdata/sample.go"

func TestE2E_CrossLanguage(t *testing.T) {
	schema, err := parser.ParseSchemaFile(samplePath)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	goSrc, err := generator.GenerateGoSchema(schema, "main")
	if err != nil {
		t.Fatalf("GenerateGoSchema: %v", err)
	}

	goDir := buildGoHarness(t, goSrc)

	cases := []struct {
		name    string
		typ     string
		epsilon float64
	}{
		{"Vector3", "Vector3", 0.02},           // quantized float32 → допустимая погрешность
		{"SpawnMessage", "SpawnMessage", 0.02}, // mix: int, nested, []string, []byte
		{"MoveMessage", "MoveMessage", 0.02},   // bool bit packing: Active, Visible, Ghost
		{"EnvelopeMessage", "EnvelopeMessage", 0},
		{"QuantTestMessage", "QuantTestMessage", 0.02},
	}

	t.Run("CS", func(t *testing.T) {
		if _, err := exec.LookPath("dotnet"); err != nil {
			t.Skip("dotnet not found")
		}
		csSrc, err := generator.GenerateCSharpSchema(schema, "Ragono.Messages")
		if err != nil {
			t.Fatalf("GenerateCSharpSchema: %v", err)
		}
		csDir := buildCSHarness(t, csSrc)

		for _, tc := range cases {
			t.Run("Wire/Go_EQ_CS/"+tc.name, func(t *testing.T) {
				goHex := strings.TrimSpace(runHarness(t, goDir, "go", "ser", tc.typ, ""))
				csHex := strings.TrimSpace(runHarness(t, csDir, "cs", "ser", tc.typ, ""))
				if goHex != csHex {
					t.Fatalf("wire drift between Go and C# for %s:\ngo=%s\ncs=%s", tc.typ, goHex, csHex)
				}
			})
		}

		for _, tc := range cases {
			t.Run("Go_to_CS/"+tc.name, func(t *testing.T) {
				hex := runHarness(t, goDir, "go", "ser", tc.typ, "")
				out := runHarness(t, csDir, "cs", "deser", tc.typ, hex)
				checkOutput(t, tc.typ, out, tc.epsilon)
			})
			t.Run("CS_to_Go/"+tc.name, func(t *testing.T) {
				hex := runHarness(t, csDir, "cs", "ser", tc.typ, "")
				out := runHarness(t, goDir, "go", "deser", tc.typ, hex)
				checkOutput(t, tc.typ, out, tc.epsilon)
			})
		}
	})
	// TestE2E_CrossLanguage — TS block

	t.Run("TS", func(t *testing.T) {
		if _, err := exec.LookPath("node"); err != nil {
			t.Skip("node not found")
		}
		tsSrc, err := generator.GenerateTypeScriptSchema(schema)
		if err != nil {
			t.Fatalf("GenerateTypeScriptSchema: %v", err)
		}
		tsDir := buildTSHarness(t, tsSrc)

		for _, tc := range cases {
			t.Run("Wire/Go_EQ_TS/"+tc.name, func(t *testing.T) {
				goHex := strings.TrimSpace(runHarness(t, goDir, "go", "ser", tc.typ, ""))
				tsHex := strings.TrimSpace(runHarness(t, tsDir, "ts", "ser", tc.typ, ""))
				if goHex != tsHex {
					t.Fatalf("wire drift between Go and TS for %s:\ngo=%s\nts=%s", tc.typ, goHex, tsHex)
				}
			})
		}

		for _, tc := range cases {
			t.Run("Go_to_TS/"+tc.name, func(t *testing.T) {
				hex := runHarness(t, goDir, "go", "ser", tc.typ, "")
				out := runHarness(t, tsDir, "ts", "deser", tc.typ, hex)
				checkOutput(t, tc.typ, out, tc.epsilon)
			})
			t.Run("TS_to_Go/"+tc.name, func(t *testing.T) {
				hex := runHarness(t, tsDir, "ts", "ser", tc.typ, "")
				out := runHarness(t, goDir, "go", "deser", tc.typ, hex)
				checkOutput(t, tc.typ, out, tc.epsilon)
			})
		}
	})
	// TestE2E_CrossLanguage — Lua block

	t.Run("Lua", func(t *testing.T) {
		if _, err := exec.LookPath("luajit"); err != nil {
			t.Skip("luajit not found")
		}
		// Use a simpler test schema without int64/uint64 for Lua
		luaSchema := parser.Schema{
			Messages: []parser.Message{
				{
					Name: "Vector3",
					Fields: []parser.Field{
						{Name: "X", Kind: parser.KindPrimitive, Primitive: parser.KindFloat32, Quant: &parser.QuantInfo{Min: -500, Max: 500, Bits: 16}},
						{Name: "Y", Kind: parser.KindPrimitive, Primitive: parser.KindFloat32, Quant: &parser.QuantInfo{Min: -500, Max: 500, Bits: 16}},
						{Name: "Z", Kind: parser.KindPrimitive, Primitive: parser.KindFloat32, Quant: &parser.QuantInfo{Min: -500, Max: 500, Bits: 16}},
					},
				},
				{
					Name: "MoveMessage",
					Fields: []parser.Field{
						{Name: "Position", Kind: parser.KindNested, TypeName: "Vector3"},
						{Name: "Velocity", Kind: parser.KindFixedArray, FixedLen: 3, Elem: &parser.Field{Kind: parser.KindPrimitive, Primitive: parser.KindFloat32}},
						{Name: "Waypoints", Kind: parser.KindSlice, Elem: &parser.Field{Kind: parser.KindNested, TypeName: "Vector3"}},
						{Name: "PlayerID", Kind: parser.KindPrimitive, Primitive: parser.KindUint32},
						{Name: "Active", Kind: parser.KindPrimitive, Primitive: parser.KindBool},
						{Name: "Visible", Kind: parser.KindPrimitive, Primitive: parser.KindBool},
						{Name: "Ghost", Kind: parser.KindPrimitive, Primitive: parser.KindBool},
						{Name: "Name", Kind: parser.KindPrimitive, Primitive: parser.KindString},
					},
				},
				{
					Name: "EnvelopeMessage",
					Fields: []parser.Field{
						{Name: "Code", Kind: parser.KindPrimitive, Primitive: parser.KindUint16},
						{Name: "Counter", Kind: parser.KindPrimitive, Primitive: parser.KindUint8},
					},
				},
			},
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
		}

		luaSrc, err := generator.GenerateLuaSchema(luaSchema, "messages")
		if err != nil {
			t.Fatalf("GenerateLuaSchema: %v", err)
		}
		luaDir := buildLuaHarness(t, luaSrc)

		luaCases := []struct {
			name    string
			typ     string
			epsilon float64
		}{
			{"Vector3", "Vector3", 0.02},
			{"MoveMessage", "MoveMessage", 0.02},
			{"EnvelopeMessage", "EnvelopeMessage", 0},
		}

		for _, tc := range luaCases {
			t.Run("Wire/Go_EQ_Lua/"+tc.name, func(t *testing.T) {
				goHex := strings.TrimSpace(runHarness(t, goDir, "go", "ser", tc.typ, ""))
				luaHex := strings.TrimSpace(runHarness(t, luaDir, "lua", "ser", tc.typ, ""))
				if goHex != luaHex {
					t.Fatalf("wire drift between Go and Lua for %s:\ngo=%s\nlua=%s", tc.typ, goHex, luaHex)
				}
			})
		}

		for _, tc := range luaCases {
			t.Run("Go_to_Lua/"+tc.name, func(t *testing.T) {
				hex := runHarness(t, goDir, "go", "ser", tc.typ, "")
				out := runHarness(t, luaDir, "lua", "deser", tc.typ, hex)
				checkOutput(t, tc.typ, out, tc.epsilon)
			})
			t.Run("Lua_to_Go/"+tc.name, func(t *testing.T) {
				hex := runHarness(t, luaDir, "lua", "ser", tc.typ, "")
				out := runHarness(t, goDir, "go", "deser", tc.typ, hex)
				checkOutput(t, tc.typ, out, tc.epsilon)
			})
		}
	})
	// TestE2E_QuantBoundaryValues — CS block
}

func TestE2E_QuantBoundaryValues(t *testing.T) {
	schema, err := parser.ParseSchemaFile(samplePath)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// Build Go harness
	goSrc, err := generator.GenerateGoSchema(schema, "main")
	if err != nil {
		t.Fatalf("GenerateGoSchema: %v", err)
	}
	goDir := buildGoHarness(t, goSrc)

	// Get reference hex from Go
	goHex := strings.TrimSpace(runHarness(t, goDir, "go", "ser", "QuantTestMessage", ""))
	t.Logf("Go hex: %s (len=%d)", goHex, len(goHex)/2)

	// Expected quantized value for DW_divergenceval = -491.989: 525 (0x020d)
	// Expected quantized value for zero: 32768 (0x8000) — the midpoint
	// Expected quantized value for max (500): 65535 (0xffff)
	// Expected quantized value for min (-500): 0 (0x0000)
	// Expected quantized value for -0.001: 32767 (0x7fff) [one less than midpoint]
	// Expected quantized value for 499.999: 65534 (0xfffe) [one less than max]
	//
	// Wire format: all fields are 16-bit LE, so 6 fields * 2 bytes = 12 bytes total
	expectedLen := 12
	if len(goHex)/2 != expectedLen {
		t.Fatalf("expected %d wire bytes, got %d", expectedLen, len(goHex)/2)
	}
	// Expected quantized value for zero: 32767 (0x7fff) — the midpoint, truncated down
	// Expected quantized value for max (500): 65535 (0xffff)
	// Expected quantized value for min (-500): 0 (0x0000)
	// Expected quantized value for -0.001: 32767 (0x7fff) [same bucket as zero]
	// Expected quantized value for 499.999: 65534 (0xfffe) [one less than max]
	goBytes, _ := hex.DecodeString(goHex)
	if len(goBytes) >= 2 {
		divergenceQuant := binary.LittleEndian.Uint16(goBytes[0:2])
		if divergenceQuant != 525 {
			t.Errorf("DivergenceVal quantized value: expected 525, got %d", divergenceQuant)
		}
	}
	if len(goBytes) >= 4 {
		zeroQuant := binary.LittleEndian.Uint16(goBytes[2:4])
		if zeroQuant != 32767 {
			t.Errorf("ZeroVal quantized value: expected 32767, got %d", zeroQuant)
		}
	}

	// Test C# produces identical output
	t.Run("CS", func(t *testing.T) {
		if _, err := exec.LookPath("dotnet"); err != nil {
			t.Skip("dotnet not found")
		}
		csSrc, err := generator.GenerateCSharpSchema(schema, "Ragono.Messages")
		if err != nil {
			t.Fatalf("GenerateCSharpSchema: %v", err)
		}
		csDir := buildCSHarness(t, csSrc)
		csHex := strings.TrimSpace(runHarness(t, csDir, "cs", "ser", "QuantTestMessage", ""))
		if goHex != csHex {
			t.Errorf("wire drift between Go and C# for QuantTestMessage:\ngo=%s\ncs=%s", goHex, csHex)
		}
		t.Logf("C# hex: %s", csHex)
	})
	// TestE2E_QuantBoundaryValues — TS block

	// Test TS produces identical output
	t.Run("TS", func(t *testing.T) {
		if _, err := exec.LookPath("node"); err != nil {
			t.Skip("node not found")
		}
		tsSrc, err := generator.GenerateTypeScriptSchema(schema)
		if err != nil {
			t.Fatalf("GenerateTypeScriptSchema: %v", err)
		}
		tsDir := buildTSHarness(t, tsSrc)
		tsHex := strings.TrimSpace(runHarness(t, tsDir, "ts", "ser", "QuantTestMessage", ""))
		if goHex != tsHex {
			t.Errorf("wire drift between Go and TS for QuantTestMessage:\ngo=%s\nts=%s", goHex, tsHex)
		}
		t.Logf("TS hex: %s", tsHex)
	})
	// TestE2E_QuantBoundaryValues — Lua block

	// Test Lua produces identical output (if schema is compatible)
	t.Run("Lua", func(t *testing.T) {
		if _, err := exec.LookPath("luajit"); err != nil {
			t.Skip("luajit not found")
		}
		// Use a Lua-compatible schema (no int64/uint64)
		luaSchema := parser.Schema{
			Messages: []parser.Message{
				{Name: "QuantTestMessage", Fields: []parser.Field{
					{Name: "DivergenceVal", Kind: parser.KindPrimitive, Primitive: parser.KindFloat64, Quant: &parser.QuantInfo{Min: -500, Max: 500, Bits: 16}},
					{Name: "ZeroVal", Kind: parser.KindPrimitive, Primitive: parser.KindFloat64, Quant: &parser.QuantInfo{Min: -500, Max: 500, Bits: 16}},
					{Name: "MaxBoundVal", Kind: parser.KindPrimitive, Primitive: parser.KindFloat64, Quant: &parser.QuantInfo{Min: -500, Max: 500, Bits: 16}},
					{Name: "MinBoundVal", Kind: parser.KindPrimitive, Primitive: parser.KindFloat64, Quant: &parser.QuantInfo{Min: -500, Max: 500, Bits: 16}},
					{Name: "NearZeroVal", Kind: parser.KindPrimitive, Primitive: parser.KindFloat64, Quant: &parser.QuantInfo{Min: -500, Max: 500, Bits: 16}},
					{Name: "NearHighVal", Kind: parser.KindPrimitive, Primitive: parser.KindFloat64, Quant: &parser.QuantInfo{Min: -500, Max: 500, Bits: 16}},
				}},
			},
		}
		luaSrc, err := generator.GenerateLuaSchema(luaSchema, "messages_gen")
		if err != nil {
			t.Fatalf("GenerateLuaSchema: %v", err)
		}
		luaDir := buildLuaHarness(t, luaSrc)
		luaHex := strings.TrimSpace(runHarness(t, luaDir, "lua", "ser", "QuantTestMessage", ""))
		if goHex != luaHex {
			t.Errorf("wire drift between Go and Lua for QuantTestMessage:\ngo=%s\nlua=%s", goHex, luaHex)
		}
		t.Logf("Lua hex: %s", luaHex)
	})
	// TestE2E_NonGoPivot — CS block

	// Verify roundtrip in Go (deserialize the wire and check values)
	out := runHarness(t, goDir, "go", "deser", "QuantTestMessage", goHex)
	kv := parseKV(out)
	verifyQuantValue(t, kv, "DivergenceVal", -491.989)
	verifyQuantValue(t, kv, "ZeroVal", 0.0)
	verifyQuantValue(t, kv, "MaxBoundVal", 500.0)
	verifyQuantValue(t, kv, "MinBoundVal", -500.0)
	verifyQuantValue(t, kv, "NearZeroVal", -0.001)
	verifyQuantValue(t, kv, "NearHighVal", 499.999)
}

func TestE2E_NonGoPivot(t *testing.T) {
	schema, err := parser.ParseSchemaFile(samplePath)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// Build Go, C#, TS, Lua harnesses as needed (at parent scope so dirs survive subtests)
	var csDir, tsDir, luaDir string
	hasCS, hasTS, hasLua := false, false, false

	if _, err := exec.LookPath("dotnet"); err == nil {
		csSrc, err := generator.GenerateCSharpSchema(schema, "Ragono.Messages")
		if err != nil {
			t.Fatalf("GenerateCSharpSchema: %v", err)
		}
		csDir = buildCSHarness(t, csSrc)
		hasCS = true
	} else {
		t.Log("dotnet not found, skipping C#")
	}

	if _, err := exec.LookPath("node"); err == nil {
		tsSrc, err := generator.GenerateTypeScriptSchema(schema)
		if err != nil {
			t.Fatalf("GenerateTypeScriptSchema: %v", err)
		}
		tsDir = buildTSHarness(t, tsSrc)
		hasTS = true
	} else {
		t.Log("node not found, skipping TS")
	}

	if _, err := exec.LookPath("luajit"); err == nil {
		luaSchema := parser.Schema{
			Messages: []parser.Message{
				{Name: "Vector3", Fields: []parser.Field{
					{Name: "X", Kind: parser.KindPrimitive, Primitive: parser.KindFloat32, Quant: &parser.QuantInfo{Min: -500, Max: 500, Bits: 16}},
					{Name: "Y", Kind: parser.KindPrimitive, Primitive: parser.KindFloat32, Quant: &parser.QuantInfo{Min: -500, Max: 500, Bits: 16}},
					{Name: "Z", Kind: parser.KindPrimitive, Primitive: parser.KindFloat32, Quant: &parser.QuantInfo{Min: -500, Max: 500, Bits: 16}},
				}},
				{Name: "EnvelopeMessage", Fields: []parser.Field{
					{Name: "Code", Kind: parser.KindPrimitive, Primitive: parser.KindUint16},
					{Name: "Counter", Kind: parser.KindPrimitive, Primitive: parser.KindUint8},
				}},
			},
			Enums: []parser.Enum{
				{Name: "Opcode", Primitive: parser.KindUint16, Values: []parser.EnumValue{
					{Name: "Unknown", Value: "0"},
					{Name: "Join", Value: "1"},
					{Name: "Leave", Value: "2"},
				}},
			},
		}
		luaSrc, err := generator.GenerateLuaSchema(luaSchema, "messages")
		if err != nil {
			t.Fatalf("GenerateLuaSchema: %v", err)
		}
		luaDir = buildLuaHarness(t, luaSrc)
		hasLua = true
	} else {
		t.Log("luajit not found, skipping Lua")
	}
	// TestE2E_TruncatedInput — CS block

	// C# → TS and TS → C# roundtrip
	if hasCS && hasTS {
		for _, name := range []string{"Vector3", "EnvelopeMessage"} {
			t.Run("CS_to_TS/"+name, func(t *testing.T) {
				hex := runHarness(t, csDir, "cs", "ser", name, "")
				out := runHarness(t, tsDir, "ts", "deser", name, hex)
				checkOutput(t, name, out, 0.02)
			})
			t.Run("TS_to_CS/"+name, func(t *testing.T) {
				hex := runHarness(t, tsDir, "ts", "ser", name, "")
				out := runHarness(t, csDir, "cs", "deser", name, hex)
				checkOutput(t, name, out, 0.02)
			})
		}
	}

	// C# → Lua and Lua → C# roundtrip
	if hasCS && hasLua {
		for _, name := range []string{"Vector3", "EnvelopeMessage"} {
			t.Run("CS_to_Lua/"+name, func(t *testing.T) {
				hex := runHarness(t, csDir, "cs", "ser", name, "")
				out := runHarness(t, luaDir, "lua", "deser", name, hex)
				checkOutput(t, name, out, 0.02)
			})
			t.Run("Lua_to_CS/"+name, func(t *testing.T) {
				hex := runHarness(t, luaDir, "lua", "ser", name, "")
				out := runHarness(t, csDir, "cs", "deser", name, hex)
				checkOutput(t, name, out, 0.02)
			})
		}
	}

	// TS → Lua and Lua → TS roundtrip
	if hasTS && hasLua {
		for _, name := range []string{"Vector3", "EnvelopeMessage"} {
			t.Run("TS_to_Lua/"+name, func(t *testing.T) {
				hex := runHarness(t, tsDir, "ts", "ser", name, "")
				out := runHarness(t, luaDir, "lua", "deser", name, hex)
				checkOutput(t, name, out, 0.02)
			})
			t.Run("Lua_to_TS/"+name, func(t *testing.T) {
				hex := runHarness(t, luaDir, "lua", "ser", name, "")
				out := runHarness(t, tsDir, "ts", "deser", name, hex)
				checkOutput(t, name, out, 0.02)
			})
		}
	}
}

func TestE2E_TruncatedInput(t *testing.T) {
	schema, err := parser.ParseSchemaFile(samplePath)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	goSrc, err := generator.GenerateGoSchema(schema, "main")
	if err != nil {
		t.Fatalf("GenerateGoSchema: %v", err)
	}
	goDir := buildGoHarness(t, goSrc)

	// All test types that provide deserialize in the harness
	types := []string{"Vector3", "SpawnMessage", "MoveMessage", "EnvelopeMessage"}

	t.Run("Go/truncated", func(t *testing.T) {
		t.Run("Go/empty", func(t *testing.T) {
			for _, typ := range types {
				out, err := exec.Command(filepath.Join(goDir, "harness"), "deser", typ, "").CombinedOutput()
				if err == nil {
					t.Errorf("%s: expected error for empty input, got output: %s", typ, out)
					continue
				}
				if !bytes.Contains(out, []byte("unmarshal:")) {
					t.Errorf("%s: expected generated Unmarshal error, got: %s", typ, out)
				}
			}
		})

		t.Run("Go/truncated_1byte", func(t *testing.T) {
			for _, typ := range types {
				out, err := exec.Command(filepath.Join(goDir, "harness"), "deser", typ, "00").CombinedOutput()
				if err == nil {
					t.Errorf("%s: expected error for 1-byte input, got output: %s", typ, out)
					continue
				}
				if !bytes.Contains(out, []byte("unmarshal:")) {
					t.Errorf("%s: expected generated Unmarshal error, got: %s", typ, out)
				}
			}
		})
	})
	t.Run("CS", func(t *testing.T) {
		if _, err := exec.LookPath("dotnet"); err != nil {
			t.Skip("dotnet not found")
		}
		csSrc, err := generator.GenerateCSharpSchema(schema, "Ragono.Messages")
		if err != nil {
			t.Fatalf("GenerateCSharpSchema: %v", err)
		}
		csDir := buildCSHarness(t, csSrc)
		t.Run("CS/empty", func(t *testing.T) {
			for _, typ := range types {
				out, err := exec.Command("dotnet", filepath.Join(csDir, "out", "E2EHarness.dll"), "deser", typ, "").CombinedOutput()
				if err == nil {
					t.Errorf("%s: expected error for empty input, got output: %s", typ, out)
				}
			}
		})

		t.Run("CS/truncated_1byte", func(t *testing.T) {
			for _, typ := range types {
				out, err := exec.Command("dotnet", filepath.Join(csDir, "out", "E2EHarness.dll"), "deser", typ, "00").CombinedOutput()
				if err == nil {
					t.Errorf("%s: expected error for 1-byte input, got output: %s", typ, out)
				}
			}
		})
	})
	// TestE2E_TruncatedInput — TS block

	// TS truncated input tests
	t.Run("TS", func(t *testing.T) {
		if _, err := exec.LookPath("node"); err != nil {
			t.Skip("node not found")
		}
		tsSrc, err := generator.GenerateTypeScriptSchema(schema)
		if err != nil {
			t.Fatalf("GenerateTypeScriptSchema: %v", err)
		}
		tsDir := buildTSHarness(t, tsSrc)

		t.Run("TS/empty", func(t *testing.T) {
			for _, typ := range types {
				out, err := exec.Command("node", filepath.Join(tsDir, "dist", "harness.js"), "deser", typ, "").CombinedOutput()
				if err == nil {
					t.Errorf("%s: expected error for empty input, got output: %s", typ, out)
				}
			}
		})

		t.Run("TS/truncated_1byte", func(t *testing.T) {
			for _, typ := range types {
				out, err := exec.Command("node", filepath.Join(tsDir, "dist", "harness.js"), "deser", typ, "00").CombinedOutput()
				if err == nil {
					t.Errorf("%s: expected error for 1-byte input, got output: %s", typ, out)
				}
			}
		})
	})
	// TestE2E_TruncatedInput — Lua block

	// Lua truncated input tests
	t.Run("Lua", func(t *testing.T) {
		if _, err := exec.LookPath("luajit"); err != nil {
			t.Skip("luajit not found")
		}
		luaSchema := parser.Schema{
			Messages: []parser.Message{
				{Name: "Vector3", Fields: []parser.Field{
					{Name: "X", Kind: parser.KindPrimitive, Primitive: parser.KindFloat32, Quant: &parser.QuantInfo{Min: -500, Max: 500, Bits: 16}},
					{Name: "Y", Kind: parser.KindPrimitive, Primitive: parser.KindFloat32, Quant: &parser.QuantInfo{Min: -500, Max: 500, Bits: 16}},
					{Name: "Z", Kind: parser.KindPrimitive, Primitive: parser.KindFloat32, Quant: &parser.QuantInfo{Min: -500, Max: 500, Bits: 16}},
				}},
				{Name: "EnvelopeMessage", Fields: []parser.Field{
					{Name: "Code", Kind: parser.KindPrimitive, Primitive: parser.KindUint16},
					{Name: "Counter", Kind: parser.KindPrimitive, Primitive: parser.KindUint8},
				}},
			},
			Enums: []parser.Enum{
				{Name: "Opcode", Primitive: parser.KindUint16, Values: []parser.EnumValue{
					{Name: "Unknown", Value: "0"},
					{Name: "Join", Value: "1"},
					{Name: "Leave", Value: "2"},
				}},
			},
		}

		luaSrc, err := generator.GenerateLuaSchema(luaSchema, "messages")
		if err != nil {
			t.Fatalf("GenerateLuaSchema: %v", err)
		}
		luaDir := buildLuaHarness(t, luaSrc)

		luaTypes := []string{"Vector3", "EnvelopeMessage"}

		t.Run("Lua/empty", func(t *testing.T) {
			for _, typ := range luaTypes {
				out, err := exec.Command("luajit", filepath.Join(luaDir, "harness.lua"), "deser", typ, "").CombinedOutput()
				if err == nil {
					t.Errorf("%s: expected error for empty input, got output: %s", typ, out)
				}
			}
		})

		t.Run("Lua/truncated_1byte", func(t *testing.T) {
			for _, typ := range luaTypes {
				out, err := exec.Command("luajit", filepath.Join(luaDir, "harness.lua"), "deser", typ, "00").CombinedOutput()
				if err == nil {
					t.Errorf("%s: expected error for 1-byte input, got output: %s", typ, out)
				}
			}
		})
	})
}

func verifyQuantValue(t *testing.T, kv map[string]string, key string, expected float64) {
	t.Helper()
	// One bits=16 quant step over [-500,500] is 1000/65535 ≈ 0.0153; allow a bit more.
	const epsilon = 0.02
	got, err := strconv.ParseFloat(kv[key], 64)
	if err != nil {
		t.Errorf("failed to parse %s=%q: %v", key, kv[key], err)
		return
	}
	diff := got - expected
	if diff < 0 {
		diff = -diff
	}
	if diff > epsilon {
		t.Errorf("%s: expected %v, got %v (diff=%v)", key, expected, got, diff)
	}
}

func buildGoHarness(t *testing.T, generatedSrc []byte) string {
	t.Helper()
	dir := t.TempDir()

	sampleSrc, err := os.ReadFile(samplePath)
	if err != nil {
		t.Fatalf("read sample: %v", err)
	}
	sampleSrc = bytes.Replace(sampleSrc, []byte("package messages"), []byte("package main"), 1)

	write(t, filepath.Join(dir, "messages.go"), sampleSrc)
	write(t, filepath.Join(dir, "messages_arpack.go"), generatedSrc)
	write(t, filepath.Join(dir, "main.go"), []byte(goHarnessSource))
	write(t, filepath.Join(dir, "go.mod"), []byte("module arpack_e2e\n\ngo 1.21\n"))

	mustRun(t, dir, "go", "build", "-o", "harness", ".")
	return dir
}

func buildCSHarness(t *testing.T, generatedSrc []byte) string {
	t.Helper()
	dir := t.TempDir()

	write(t, filepath.Join(dir, "Messages.cs"), generatedSrc)
	write(t, filepath.Join(dir, "Program.cs"), []byte(csHarnessSource))
	write(t, filepath.Join(dir, "E2EHarness.csproj"), []byte(csProjSource))

	mustRun(t, dir, "dotnet", "build", "-c", "Release", "-o", "out")
	return dir
}

func buildTSHarness(t *testing.T, generatedSrc []byte) string {
	t.Helper()
	dir := t.TempDir()

	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", srcDir, err)
	}

	write(t, filepath.Join(srcDir, "messages.gen.ts"), generatedSrc)
	write(t, filepath.Join(srcDir, "harness.ts"), []byte(tsHarnessSource))
	write(t, filepath.Join(dir, "package.json"), []byte(tsPackageSource))
	write(t, filepath.Join(dir, "tsconfig.json"), []byte(tsConfigSource))

	mustRun(t, dir, "npm", "install")
	mustRun(t, dir, "npx", "tsc")

	return dir
}

func buildLuaHarness(t *testing.T, generatedSrc []byte) string {
	t.Helper()
	dir := t.TempDir()

	write(t, filepath.Join(dir, "messages_gen.lua"), generatedSrc)
	write(t, filepath.Join(dir, "harness.lua"), []byte(luaHarnessSource))

	return dir
}

func runHarness(t *testing.T, dir, lang, op, typ, hexInput string) string {
	t.Helper()
	var cmd *exec.Cmd
	switch lang {
	case "go":
		args := []string{op, typ}
		if hexInput != "" {
			args = append(args, hexInput)
		}
		cmd = exec.Command(filepath.Join(dir, "harness"), args...)
	case "cs":
		args := []string{op, typ}
		if hexInput != "" {
			args = append(args, hexInput)
		}
		cmd = exec.Command("dotnet", append([]string{filepath.Join(dir, "out", "E2EHarness.dll")}, args...)...)
	case "ts":
		args := []string{op, typ}
		if hexInput != "" {
			args = append(args, hexInput)
		}
		cmd = exec.Command("node", append([]string{filepath.Join(dir, "dist", "harness.js")}, args...)...)
	case "lua":
		args := []string{filepath.Join(dir, "harness.lua"), op, typ}
		if hexInput != "" {
			args = append(args, hexInput)
		}
		cmd = exec.Command("luajit", args...)
	}
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s harness %s %s failed: %v\n%s", lang, op, typ, err, out)
	}
	return strings.TrimSpace(string(out))
}

func checkOutput(t *testing.T, typ, output string, epsilon float64) {
	t.Helper()
	t.Logf("output for %s:\n%s", typ, output)

	kv := parseKV(output)

	switch typ {
	case "Vector3":
		assertFloat(t, kv, "X", 123.45, epsilon)
		assertFloat(t, kv, "Y", -200.0, epsilon)
		assertFloat(t, kv, "Z", 0.0, epsilon)

	case "SpawnMessage":
		assertInt(t, kv, "EntityID", 42)
		assertFloat(t, kv, "Position.X", 10.0, epsilon)
		assertFloat(t, kv, "Position.Y", 20.0, epsilon)
		assertFloat(t, kv, "Position.Z", 30.0, epsilon)
		assertInt(t, kv, "Health", -100)
		assertStr(t, kv, "Tags[0]", "hero")
		assertStr(t, kv, "Tags[1]", "player")
		assertInt(t, kv, "Data[0]", 1)
		assertInt(t, kv, "Data[1]", 2)
		assertInt(t, kv, "Data[2]", 3)

	case "MoveMessage":
		assertInt(t, kv, "PlayerID", 777)
		assertStr(t, kv, "Active", "true")
		assertStr(t, kv, "Visible", "false")
		assertStr(t, kv, "Ghost", "true")
		assertStr(t, kv, "Name", "TestPlayer")
	case "EnvelopeMessage":
		assertInt(t, kv, "Code", 2)
		assertInt(t, kv, "Counter", 7)
	case "QuantTestMessage":
		assertFloat(t, kv, "DivergenceVal", -491.989, epsilon)
		assertFloat(t, kv, "ZeroVal", 0.0, epsilon)
		assertFloat(t, kv, "MaxBoundVal", 500.0, epsilon)
		assertFloat(t, kv, "MinBoundVal", -500.0, epsilon)
		assertFloat(t, kv, "NearZeroVal", -0.001, epsilon)
		assertFloat(t, kv, "NearHighVal", 499.999, epsilon)
	}
}

func parseKV(s string) map[string]string {
	m := map[string]string{}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if idx := strings.IndexByte(line, '='); idx >= 0 {
			m[line[:idx]] = line[idx+1:]
		}
	}
	return m
}

func assertFloat(t *testing.T, kv map[string]string, key string, want, eps float64) {
	t.Helper()
	s, ok := kv[key]
	if !ok {
		t.Errorf("missing key %q in output", key)
	}
	got, err := strconv.ParseFloat(s, 64)
	if err != nil {
		t.Errorf("%s: cannot parse %q as float: %v", key, s, err)
		return
	}
	if math.Abs(got-want) > eps {
		t.Errorf("%s: got %v, want %v (±%v)", key, got, want, eps)
	}
}

func assertInt(t *testing.T, kv map[string]string, key string, want int64) {
	t.Helper()
	s, ok := kv[key]
	if !ok {
		t.Errorf("missing key %q in output", key)
		return
	}
	got, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		t.Errorf("%s: cannot parse %q as int: %v", key, s, err)
		return
	}
	if got != want {
		t.Errorf("%s: got %d, want %d", key, got, want)
	}
}

func assertStr(t *testing.T, kv map[string]string, key, want string) {
	t.Helper()
	got, ok := kv[key]
	if !ok {
		t.Errorf("missing key %q in output", key)
		return
	}
	if got != want {
		t.Errorf("%s: got %q, want %q", key, got, want)
	}
}

func write(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustRun(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
}

const goHarnessSource = `package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

func mustDecodeHex(s string) []byte {
	data, err := hex.DecodeString(strings.TrimSpace(s))
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode hex: %v\n", err)
		os.Exit(1)
	}
	return data
}

func mustUnmarshal(_ int, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "unmarshal: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	op  := os.Args[1] // ser | deser
	typ := os.Args[2] // Vector3 | SpawnMessage | ...

	switch op + ":" + typ {

	case "ser:Vector3":
		v := Vector3{X: 123.45, Y: -200, Z: 0}
		fmt.Println(hex.EncodeToString(v.Marshal(nil)))

	case "deser:Vector3":
		data := mustDecodeHex(os.Args[3])
		var v Vector3
		mustUnmarshal(v.Unmarshal(data))
		fmt.Printf("X=%v\nY=%v\nZ=%v\n", v.X, v.Y, v.Z)

	case "ser:SpawnMessage":
		msg := SpawnMessage{
			EntityID: 42,
			Position: Vector3{X: 10, Y: 20, Z: 30},
			Health:   -100,
			Tags:     []string{"hero", "player"},
			Data:     []uint8{1, 2, 3},
		}
		fmt.Println(hex.EncodeToString(msg.Marshal(nil)))

	case "deser:SpawnMessage":
		data := mustDecodeHex(os.Args[3])
		var msg SpawnMessage
		mustUnmarshal(msg.Unmarshal(data))
		fmt.Printf("EntityID=%d\n", msg.EntityID)
		fmt.Printf("Position.X=%v\n", msg.Position.X)
		fmt.Printf("Position.Y=%v\n", msg.Position.Y)
		fmt.Printf("Position.Z=%v\n", msg.Position.Z)
		fmt.Printf("Health=%d\n", msg.Health)
		for i, tag := range msg.Tags {
			fmt.Printf("Tags[%d]=%s\n", i, tag)
		}
		for i, b := range msg.Data {
			fmt.Printf("Data[%d]=%d\n", i, b)
		}

	case "ser:MoveMessage":
		msg := MoveMessage{
			Position:  Vector3{X: 50, Y: -100, Z: 0},
			Velocity:  [3]float32{1.5, -2.5, 0},
			Waypoints: []Vector3{{X: 10, Y: 20, Z: 0}},
			PlayerID:  777,
			Active:    true,
			Visible:   false,
			Ghost:     true,
			Name:      "TestPlayer",
		}
		fmt.Println(hex.EncodeToString(msg.Marshal(nil)))

	case "deser:MoveMessage":
		data := mustDecodeHex(os.Args[3])
		var msg MoveMessage
		mustUnmarshal(msg.Unmarshal(data))
		fmt.Printf("PlayerID=%d\n", msg.PlayerID)
		fmt.Printf("Active=%v\n", msg.Active)
		fmt.Printf("Visible=%v\n", msg.Visible)
		fmt.Printf("Ghost=%v\n", msg.Ghost)
		fmt.Printf("Name=%s\n", msg.Name)

	case "ser:EnvelopeMessage":
		msg := EnvelopeMessage{
			Code:    OpcodeJoinRoom,
			Counter: 7,
		}
		fmt.Println(hex.EncodeToString(msg.Marshal(nil)))

	case "deser:EnvelopeMessage":
		data := mustDecodeHex(os.Args[3])
		var msg EnvelopeMessage
		mustUnmarshal(msg.Unmarshal(data))
		fmt.Printf("Code=%d\n", msg.Code)
		fmt.Printf("Counter=%d\n", msg.Counter)

	case "ser:QuantTestMessage":
		msg := QuantTestMessage{
			DivergenceVal: -491.989,
			ZeroVal:       0.0,
			MaxBoundVal:   500.0,
			MinBoundVal:   -500.0,
			NearZeroVal:   -0.001,
			NearHighVal:   499.999,
		}
		fmt.Println(hex.EncodeToString(msg.Marshal(nil)))

	case "deser:QuantTestMessage":
		data := mustDecodeHex(os.Args[3])
		var msg QuantTestMessage
		mustUnmarshal(msg.Unmarshal(data))
		fmt.Printf("DivergenceVal=%v\n", msg.DivergenceVal)
		fmt.Printf("ZeroVal=%v\n", msg.ZeroVal)
		fmt.Printf("MaxBoundVal=%v\n", msg.MaxBoundVal)
		fmt.Printf("MinBoundVal=%v\n", msg.MinBoundVal)
		fmt.Printf("NearZeroVal=%v\n", msg.NearZeroVal)
		fmt.Printf("NearHighVal=%v\n", msg.NearHighVal)

	default:
		fmt.Fprintf(os.Stderr, "unknown op:type %s:%s\n", op, typ)
		os.Exit(1)
	}
}
`

const csHarnessSource = `using System;
using System.Globalization;
using System.Text;
using Ragono.Messages;

unsafe class Program
{
    static void Main(string[] args)
    {
        CultureInfo.DefaultThreadCurrentCulture = CultureInfo.InvariantCulture;
        string op  = args[0]; // ser | deser
        string typ = args[1]; // Vector3 | SpawnMessage | ...

        switch (op + ":" + typ)
        {
        case "ser:Vector3":
            SerVector3();
            break;
        case "deser:Vector3":
            DeserVector3(args[2]);
            break;
        case "ser:SpawnMessage":
            SerSpawnMessage();
            break;
        case "deser:SpawnMessage":
            DeserSpawnMessage(args[2]);
            break;
        case "ser:MoveMessage":
            SerMoveMessage();
            break;
        case "deser:MoveMessage":
            DeserMoveMessage(args[2]);
            break;
        case "ser:EnvelopeMessage":
            SerEnvelopeMessage();
            break;
        case "deser:EnvelopeMessage":
            DeserEnvelopeMessage(args[2]);
            break;
        case "ser:QuantTestMessage":
            SerQuantTestMessage();
            break;
        case "deser:QuantTestMessage":
            DeserQuantTestMessage(args[2]);
            break;
        default:
            Console.Error.WriteLine($"unknown op:type {op}:{typ}");
            Environment.Exit(1);
            break;
        }
    }

    static unsafe void SerVector3()
    {
        var msg = new Vector3 { X = 123.45f, Y = -200.0f, Z = 0.0f };
        byte[] buf = new byte[64];
        int n = msg.Serialize(buf);
        Console.WriteLine(Convert.ToHexString(buf, 0, n).ToLower());
    }

    static unsafe void DeserVector3(string hexStr)
    {
        byte[] data = Convert.FromHexString(hexStr);
        Vector3.Deserialize(data, out Vector3 msg);
        Console.WriteLine($"X={msg.X:G9}");
        Console.WriteLine($"Y={msg.Y:G9}");
        Console.WriteLine($"Z={msg.Z:G9}");
    }

    static unsafe void SerSpawnMessage()
    {
        var msg = new SpawnMessage
        {
            EntityID = 42,
            Position = new Vector3 { X = 10.0f, Y = 20.0f, Z = 30.0f },
            Health   = -100,
            Tags     = new string[] { "hero", "player" },
            Data     = new byte[] { 1, 2, 3 },
        };
        byte[] buf = new byte[512];
        int n = msg.Serialize(buf);
        Console.WriteLine(Convert.ToHexString(buf, 0, n).ToLower());
    }

    static unsafe void DeserSpawnMessage(string hexStr)
    {
        byte[] data = Convert.FromHexString(hexStr);
        SpawnMessage.Deserialize(data, out SpawnMessage msg);
        Console.WriteLine($"EntityID={msg.EntityID}");
        Console.WriteLine($"Position.X={msg.Position.X:G9}");
        Console.WriteLine($"Position.Y={msg.Position.Y:G9}");
        Console.WriteLine($"Position.Z={msg.Position.Z:G9}");
        Console.WriteLine($"Health={msg.Health}");
        if (msg.Tags != null)
            for (int i = 0; i < msg.Tags.Length; i++)
                Console.WriteLine($"Tags[{i}]={msg.Tags[i]}");
        if (msg.Data != null)
            for (int i = 0; i < msg.Data.Length; i++)
                Console.WriteLine($"Data[{i}]={msg.Data[i]}");
    }

    static unsafe void SerMoveMessage()
    {
        var msg = new MoveMessage
        {
            Position  = new Vector3 { X = 50.0f, Y = -100.0f, Z = 0.0f },
            Velocity  = new float[] { 1.5f, -2.5f, 0.0f },
            Waypoints = new Vector3[] { new Vector3 { X = 10.0f, Y = 20.0f, Z = 0.0f } },
            PlayerID  = 777,
            Active    = true,
            Visible   = false,
            Ghost     = true,
            Name      = "TestPlayer",
        };
        byte[] buf = new byte[512];
        int n = msg.Serialize(buf);
        Console.WriteLine(Convert.ToHexString(buf, 0, n).ToLower());
    }

    static unsafe void DeserMoveMessage(string hexStr)
    {
        byte[] data = Convert.FromHexString(hexStr);
        MoveMessage.Deserialize(data, out MoveMessage msg);
        Console.WriteLine($"PlayerID={msg.PlayerID}");
        Console.WriteLine($"Active={msg.Active.ToString().ToLower()}");
        Console.WriteLine($"Visible={msg.Visible.ToString().ToLower()}");
        Console.WriteLine($"Ghost={msg.Ghost.ToString().ToLower()}");
        Console.WriteLine($"Name={msg.Name}");
    }

    static unsafe void SerEnvelopeMessage()
    {
        var msg = new EnvelopeMessage
        {
            Code = Opcode.OpcodeJoinRoom,
            Counter = 7,
        };
        byte[] buf = new byte[64];
        int n = msg.Serialize(buf);
        Console.WriteLine(Convert.ToHexString(buf, 0, n).ToLower());
    }

    static unsafe void DeserEnvelopeMessage(string hexStr)
    {
        byte[] data = Convert.FromHexString(hexStr);
        EnvelopeMessage.Deserialize(data, out EnvelopeMessage msg);
        Console.WriteLine($"Code={(ushort)msg.Code}");
        Console.WriteLine($"Counter={msg.Counter}");
    }

    static unsafe void SerQuantTestMessage()
    {
        var msg = new QuantTestMessage
        {
            DivergenceVal = -491.989,
            ZeroVal = 0.0,
            MaxBoundVal = 500.0,
            MinBoundVal = -500.0,
            NearZeroVal = -0.001,
            NearHighVal = 499.999,
        };
        byte[] buf = new byte[64];
        int n = msg.Serialize(buf);
        Console.WriteLine(Convert.ToHexString(buf, 0, n).ToLower());
    }

    static unsafe void DeserQuantTestMessage(string hexStr)
    {
        byte[] data = Convert.FromHexString(hexStr);
        QuantTestMessage.Deserialize(data, out QuantTestMessage msg);
        Console.WriteLine($"DivergenceVal={msg.DivergenceVal:G}");
        Console.WriteLine($"ZeroVal={msg.ZeroVal:G}");
        Console.WriteLine($"MaxBoundVal={msg.MaxBoundVal:G}");
        Console.WriteLine($"MinBoundVal={msg.MinBoundVal:G}");
        Console.WriteLine($"NearZeroVal={msg.NearZeroVal:G}");
        Console.WriteLine($"NearHighVal={msg.NearHighVal:G}");
    }
}
`

var csProjSource = `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>net9.0</TargetFramework>
    <AllowUnsafeBlocks>true</AllowUnsafeBlocks>
    <Nullable>enable</Nullable>
    <ImplicitUsings>disable</ImplicitUsings>
  </PropertyGroup>
</Project>
`

const tsPackageSource = `{
  "name": "arpack-e2e-harness",
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "build": "tsc"
  },
  "devDependencies": {
    "typescript": "^5.3.0",
    "@types/node": "^20.0.0"
  }
}
`

const tsConfigSource = `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "lib": ["ES2022", "DOM"],
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "outDir": "./dist",
    "rootDir": "./src"
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist"]
}
`

const tsHarnessSource = `import { readFileSync } from 'fs';
import { argv } from 'process';

import { Vector3, MoveMessage, SpawnMessage, EnvelopeMessage, Opcode, QuantTestMessage } from './messages.gen.js';

// Hex encoding/decoding utilities
function encodeHex(data: Uint8Array): string {
  return Array.from(data)
    .map(b => b.toString(16).padStart(2, '0'))
    .join('');
}

function decodeHex(hex: string): Uint8Array {
  const bytes = new Uint8Array(hex.length / 2);
  for (let i = 0; i < hex.length; i += 2) {
    bytes[i / 2] = parseInt(hex.substr(i, 2), 16);
  }
  return bytes;
}

// Main harness
function main() {
  const args = argv.slice(2);
  const op = args[0]; // 'ser' or 'deser'
  const typ = args[1]; // message type
  const hexInput = args[2]; // for deser

  switch (` + "`${op}:${typ}`" + `) {
    case 'ser:Vector3': {
      const msg = new Vector3();
      msg.x = 123.45;
      msg.y = -200.0;
      msg.z = 0.0;
      const buf = new Uint8Array(64);
      const n = msg.serialize(buf);
      const bytes = buf.subarray(0, n);
      console.log(encodeHex(bytes));
      break;
    }

    case 'deser:Vector3': {
      const data = decodeHex(hexInput);
      const [msg] = Vector3.deserialize(data);
      console.log(` + "`X=${msg.x.toPrecision(9)}`" + `);
      console.log(` + "`Y=${msg.y.toPrecision(9)}`" + `);
      console.log(` + "`Z=${msg.z.toPrecision(9)}`" + `);
      break;
    }

    case 'ser:SpawnMessage': {
      const msg = new SpawnMessage();
      msg.entityID = 42n;
      msg.position = new Vector3();
      msg.position.x = 10.0;
      msg.position.y = 20.0;
      msg.position.z = 30.0;
      msg.health = -100;
      msg.tags = ['hero', 'player'];
      msg.data = [1, 2, 3];
      const buf = new Uint8Array(512);
      const n = msg.serialize(buf);
      const bytes = buf.subarray(0, n);
      console.log(encodeHex(bytes));
      break;
    }

    case 'deser:SpawnMessage': {
      const data = decodeHex(hexInput);
      const [msg] = SpawnMessage.deserialize(data);
      console.log(` + "`EntityID=${msg.entityID.toString()}`" + `);
      console.log(` + "`Position.X=${msg.position.x.toPrecision(9)}`" + `);
      console.log(` + "`Position.Y=${msg.position.y.toPrecision(9)}`" + `);
      console.log(` + "`Position.Z=${msg.position.z.toPrecision(9)}`" + `);
      console.log(` + "`Health=${msg.health}`" + `);
      for (let i = 0; i < msg.tags.length; i++) {
        console.log(` + "`Tags[${i}]=${msg.tags[i]}`" + `);
      }
      for (let i = 0; i < msg.data.length; i++) {
        console.log(` + "`Data[${i}]=${msg.data[i]}`" + `);
      }
      break;
    }

    case 'ser:MoveMessage': {
      const msg = new MoveMessage();
      msg.position = new Vector3();
      msg.position.x = 50.0;
      msg.position.y = -100.0;
      msg.position.z = 0.0;
      msg.velocity = [1.5, -2.5, 0.0];
      msg.waypoints = [new Vector3()];
      msg.waypoints[0].x = 10.0;
      msg.waypoints[0].y = 20.0;
      msg.waypoints[0].z = 0.0;
      msg.playerID = 777;
      msg.active = true;
      msg.visible = false;
      msg.ghost = true;
      msg.name = 'TestPlayer';
      const buf = new Uint8Array(512);
      const n = msg.serialize(buf);
      const bytes = buf.subarray(0, n);
      console.log(encodeHex(bytes));
      break;
    }

    case 'deser:MoveMessage': {
      const data = decodeHex(hexInput);
      const [msg] = MoveMessage.deserialize(data);
      console.log(` + "`PlayerID=${msg.playerID}`" + `);
      console.log(` + "`Active=${msg.active.toString().toLowerCase()}`" + `);
      console.log(` + "`Visible=${msg.visible.toString().toLowerCase()}`" + `);
      console.log(` + "`Ghost=${msg.ghost.toString().toLowerCase()}`" + `);
      console.log(` + "`Name=${msg.name}`" + `);
      break;
    }

    case 'ser:EnvelopeMessage': {
      const msg = new EnvelopeMessage();
      msg.code = 2; // Opcode.JoinRoom
      msg.counter = 7;
      const buf = new Uint8Array(64);
      const n = msg.serialize(buf);
      const bytes = buf.subarray(0, n);
      console.log(encodeHex(bytes));
      break;
    }

    case 'deser:EnvelopeMessage': {
      const data = decodeHex(hexInput);
      const [msg] = EnvelopeMessage.deserialize(data);
      console.log(` + "`Code=${msg.code}`" + `);
      console.log(` + "`Counter=${msg.counter}`" + `);
      break;
    }

    case 'ser:QuantTestMessage': {
      const msg = new QuantTestMessage();
      msg.divergenceVal = -491.989;
      msg.zeroVal = 0.0;
      msg.maxBoundVal = 500.0;
      msg.minBoundVal = -500.0;
      msg.nearZeroVal = -0.001;
      msg.nearHighVal = 499.999;
      const buf = new Uint8Array(64);
      const n = msg.serialize(buf);
      const bytes = buf.subarray(0, n);
      console.log(encodeHex(bytes));
      break;
    }

    case 'deser:QuantTestMessage': {
      const data = decodeHex(hexInput);
      const [msg] = QuantTestMessage.deserialize(data);
      console.log(` + "`DivergenceVal=${msg.divergenceVal.toPrecision(9)}`" + `);
      console.log(` + "`ZeroVal=${msg.zeroVal.toPrecision(9)}`" + `);
      console.log(` + "`MaxBoundVal=${msg.maxBoundVal.toPrecision(9)}`" + `);
      console.log(` + "`MinBoundVal=${msg.minBoundVal.toPrecision(9)}`" + `);
      console.log(` + "`NearZeroVal=${msg.nearZeroVal.toPrecision(9)}`" + `);
      console.log(` + "`NearHighVal=${msg.nearHighVal.toPrecision(9)}`" + `);
      break;
    }


    default:
      console.error(` + "`Unknown op:type ${op}:${typ}`" + `);
      process.exit(1);
  }
}

main();
`

const luaHarnessSource = `-- Lua E2E Harness
-- Usage: luajit harness.lua <op> <type> [hex_input]
-- op: 'ser' or 'deser'
-- type: message type name

local messages = require("messages_gen")

local function hexToBytes(hex)
    local bytes = {}
    for i = 1, #hex, 2 do
        local byte = tonumber(hex:sub(i, i+1), 16)
        table.insert(bytes, string.char(byte))
    end
    return table.concat(bytes)
end

local function bytesToHex(data)
    local hex = {}
    for i = 1, #data do
        table.insert(hex, string.format("%02x", string.byte(data, i)))
    end
    return table.concat(hex)
end

local function serializeVector3()
    local msg = messages.new_vector3()
    msg.x = 123.45
    msg.y = -200.0
    msg.z = 0.0
    return bytesToHex(messages.serialize_vector3(msg))
end

local function deserializeVector3(hex)
    local data = hexToBytes(hex)
    local msg = messages.deserialize_vector3(data)
    print(string.format("X=%.10g", msg.x))
    print(string.format("Y=%.10g", msg.y))
    print(string.format("Z=%.10g", msg.z))
end

local function serializeMoveMessage()
    local msg = messages.new_move_message()
    msg.position = messages.new_vector3()
    msg.position.x = 50.0
    msg.position.y = -100.0
    msg.position.z = 0.0
    msg.velocity = {1.5, -2.5, 0.0}
    msg.waypoints = {}
    local wp = messages.new_vector3()
    wp.x = 10.0
    wp.y = 20.0
    wp.z = 0.0
    table.insert(msg.waypoints, wp)
    msg.player_id = 777
    msg.active = true
    msg.visible = false
    msg.ghost = true
    msg.name = "TestPlayer"
    return bytesToHex(messages.serialize_move_message(msg))
end

local function deserializeMoveMessage(hex)
    local data = hexToBytes(hex)
    local msg = messages.deserialize_move_message(data)
    print(string.format("PlayerID=%d", msg.player_id))
    print(string.format("Active=%s", tostring(msg.active)))
    print(string.format("Visible=%s", tostring(msg.visible)))
    print(string.format("Ghost=%s", tostring(msg.ghost)))
    print(string.format("Name=%s", msg.name))
end

local function serializeEnvelopeMessage()
    local msg = messages.new_envelope_message()
    msg.code = 2  -- Join
    msg.counter = 7
    return bytesToHex(messages.serialize_envelope_message(msg))
end

local function deserializeEnvelopeMessage(hex)
    local data = hexToBytes(hex)
    local msg = messages.deserialize_envelope_message(data)
    print(string.format("Code=%d", msg.code))
    print(string.format("Counter=%d", msg.counter))
end


local function serializeQuantTestMessage()
    local msg = messages.new_quant_test_message()
    msg.divergence_val = -491.989
    msg.zero_val = 0.0
    msg.max_bound_val = 500.0
    msg.min_bound_val = -500.0
    msg.near_zero_val = -0.001
    msg.near_high_val = 499.999
    return bytesToHex(messages.serialize_quant_test_message(msg))
end

local function deserializeQuantTestMessage(hex)
    local data = hexToBytes(hex)
    local msg = messages.deserialize_quant_test_message(data)
    print(string.format("DivergenceVal=%.10g", msg.divergence_val))
    print(string.format("ZeroVal=%.10g", msg.zero_val))
    print(string.format("MaxBoundVal=%.10g", msg.max_bound_val))
    print(string.format("MinBoundVal=%.10g", msg.min_bound_val))
    print(string.format("NearZeroVal=%.10g", msg.near_zero_val))
    print(string.format("NearHighVal=%.10g", msg.near_high_val))
end

local op = arg[1]
local typ = arg[2]
local hexInput = arg[3]

local key = op .. ":" .. typ

if key == "ser:Vector3" then
    print(serializeVector3())
elseif key == "deser:Vector3" then
    deserializeVector3(hexInput)
elseif key == "ser:MoveMessage" then
    print(serializeMoveMessage())
elseif key == "deser:MoveMessage" then
    deserializeMoveMessage(hexInput)
elseif key == "ser:EnvelopeMessage" then
    print(serializeEnvelopeMessage())
elseif key == "deser:EnvelopeMessage" then
    deserializeEnvelopeMessage(hexInput)
elseif key == "ser:QuantTestMessage" then
    print(serializeQuantTestMessage())
elseif key == "deser:QuantTestMessage" then
    deserializeQuantTestMessage(hexInput)
else
    error("Unknown op:type " .. key)
end
`
