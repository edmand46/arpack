package e2e

import (
	"bytes"
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

// TestE2E_CrossLanguage гоняет сериализацию в обе стороны: Go → C# / C# → Go / Go → TS / TS → Go.
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
	}

	// C# tests (if dotnet is available)
	if _, err := exec.LookPath("dotnet"); err == nil {
		csSrc, err := generator.GenerateCSharpSchema(schema, "Ragono.Messages")
		if err != nil {
			t.Fatalf("GenerateCSharpSchema: %v", err)
		}
		csDir := buildCSHarness(t, csSrc)

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
	} else {
		t.Log("dotnet not found, skipping C# cross-language e2e tests")
	}

	// TypeScript tests (if node and tsx are available)
	if _, err := exec.LookPath("node"); err == nil {
		tsSrc, err := generator.GenerateTypeScriptSchema(schema, "Arpack.Messages")
		if err != nil {
			t.Fatalf("GenerateTypeScriptSchema: %v", err)
		}
		tsDir := buildTSHarness(t, tsSrc)

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
	} else {
		t.Log("node not found, skipping TypeScript cross-language e2e tests")
	}
}

// --- Build helpers ---

func buildGoHarness(t *testing.T, generatedSrc []byte) string {
	t.Helper()
	dir := t.TempDir()

	// Читаем sample.go и меняем package на main
	sampleSrc, err := os.ReadFile(samplePath)
	if err != nil {
		t.Fatalf("read sample: %v", err)
	}
	sampleSrc = bytes.Replace(sampleSrc, []byte("package messages"), []byte("package main"), 1)

	// Generated код уже имеет package main (мы передали "main" в GenerateGo)
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

	// Create src directory
	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", srcDir, err)
	}

	// Write generated messages
	write(t, filepath.Join(srcDir, "messages.gen.ts"), generatedSrc)

	// Write harness
	write(t, filepath.Join(srcDir, "harness.ts"), []byte(tsHarnessSource))

	// Write package.json
	write(t, filepath.Join(dir, "package.json"), []byte(tsPackageSource))

	// Write tsconfig.json
	write(t, filepath.Join(dir, "tsconfig.json"), []byte(tsConfigSource))

	// Install dependencies and build
	mustRun(t, dir, "npm", "install")
	mustRun(t, dir, "npx", "tsc")

	return dir
}

// --- Harness runners ---

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
	}
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s harness %s %s failed: %v\n%s", lang, op, typ, err, out)
	}
	return strings.TrimSpace(string(out))
}

// --- Output verification ---

// checkOutput парсит key=value вывод и сравнивает с ожидаемыми значениями.
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
		return
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

// --- Utilities ---

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

// --- Go harness source ---

const goHarnessSource = `package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

func main() {
	op  := os.Args[1] // ser | deser
	typ := os.Args[2] // Vector3 | SpawnMessage | ...

	switch op + ":" + typ {

	case "ser:Vector3":
		v := Vector3{X: 123.45, Y: -200, Z: 0}
		fmt.Println(hex.EncodeToString(v.Marshal(nil)))

	case "deser:Vector3":
		data, _ := hex.DecodeString(strings.TrimSpace(os.Args[3]))
		var v Vector3
		v.Unmarshal(data)
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
		data, _ := hex.DecodeString(strings.TrimSpace(os.Args[3]))
		var msg SpawnMessage
		msg.Unmarshal(data)
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
		data, _ := hex.DecodeString(strings.TrimSpace(os.Args[3]))
		var msg MoveMessage
		msg.Unmarshal(data)
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
		data, _ := hex.DecodeString(strings.TrimSpace(os.Args[3]))
		var msg EnvelopeMessage
		msg.Unmarshal(data)
		fmt.Printf("Code=%d\n", msg.Code)
		fmt.Printf("Counter=%d\n", msg.Counter)

	default:
		fmt.Fprintf(os.Stderr, "unknown op:type %s:%s\n", op, typ)
		os.Exit(1)
	}
}
`

// --- C# harness source ---

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
        fixed (byte* ptr = buf)
        {
            int n = msg.Serialize(ptr);
            Console.WriteLine(Convert.ToHexString(buf, 0, n).ToLower());
        }
    }

    static unsafe void DeserVector3(string hexStr)
    {
        byte[] data = Convert.FromHexString(hexStr);
        fixed (byte* ptr = data)
        {
            Vector3.Deserialize(ptr, out Vector3 msg);
            Console.WriteLine($"X={msg.X:G9}");
            Console.WriteLine($"Y={msg.Y:G9}");
            Console.WriteLine($"Z={msg.Z:G9}");
        }
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
        fixed (byte* ptr = buf)
        {
            int n = msg.Serialize(ptr);
            Console.WriteLine(Convert.ToHexString(buf, 0, n).ToLower());
        }
    }

    static unsafe void DeserSpawnMessage(string hexStr)
    {
        byte[] data = Convert.FromHexString(hexStr);
        fixed (byte* ptr = data)
        {
            SpawnMessage.Deserialize(ptr, out SpawnMessage msg);
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
        fixed (byte* ptr = buf)
        {
            int n = msg.Serialize(ptr);
            Console.WriteLine(Convert.ToHexString(buf, 0, n).ToLower());
        }
    }

    static unsafe void DeserMoveMessage(string hexStr)
    {
        byte[] data = Convert.FromHexString(hexStr);
        fixed (byte* ptr = data)
        {
            MoveMessage.Deserialize(ptr, out MoveMessage msg);
            Console.WriteLine($"PlayerID={msg.PlayerID}");
            Console.WriteLine($"Active={msg.Active.ToString().ToLower()}");
            Console.WriteLine($"Visible={msg.Visible.ToString().ToLower()}");
            Console.WriteLine($"Ghost={msg.Ghost.ToString().ToLower()}");
            Console.WriteLine($"Name={msg.Name}");
        }
    }

    static unsafe void SerEnvelopeMessage()
    {
        var msg = new EnvelopeMessage
        {
            Code = Opcode.JoinRoom,
            Counter = 7,
        };
        byte[] buf = new byte[64];
        fixed (byte* ptr = buf)
        {
            int n = msg.Serialize(ptr);
            Console.WriteLine(Convert.ToHexString(buf, 0, n).ToLower());
        }
    }

    static unsafe void DeserEnvelopeMessage(string hexStr)
    {
        byte[] data = Convert.FromHexString(hexStr);
        fixed (byte* ptr = data)
        {
            EnvelopeMessage.Deserialize(ptr, out EnvelopeMessage msg);
            Console.WriteLine($"Code={(ushort)msg.Code}");
            Console.WriteLine($"Counter={msg.Counter}");
        }
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

// --- TypeScript harness source ---

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

// Import generated messages
import { Vector3, MoveMessage, SpawnMessage, EnvelopeMessage, Opcode } from './messages.gen.js';

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
      const buf = new ArrayBuffer(64);
      const view = new DataView(buf);
      const n = msg.serialize(view, 0);
      const bytes = new Uint8Array(buf, 0, n);
      console.log(encodeHex(bytes));
      break;
    }

    case 'deser:Vector3': {
      const data = decodeHex(hexInput);
      const view = new DataView(data.buffer, data.byteOffset, data.byteLength);
      const [msg] = Vector3.deserialize(view, 0);
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
      const buf = new ArrayBuffer(512);
      const view = new DataView(buf);
      const n = msg.serialize(view, 0);
      const bytes = new Uint8Array(buf, 0, n);
      console.log(encodeHex(bytes));
      break;
    }

    case 'deser:SpawnMessage': {
      const data = decodeHex(hexInput);
      const view = new DataView(data.buffer, data.byteOffset, data.byteLength);
      const [msg] = SpawnMessage.deserialize(view, 0);
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
      const buf = new ArrayBuffer(512);
      const view = new DataView(buf);
      const n = msg.serialize(view, 0);
      const bytes = new Uint8Array(buf, 0, n);
      console.log(encodeHex(bytes));
      break;
    }

    case 'deser:MoveMessage': {
      const data = decodeHex(hexInput);
      const view = new DataView(data.buffer, data.byteOffset, data.byteLength);
      const [msg] = MoveMessage.deserialize(view, 0);
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
      const buf = new ArrayBuffer(64);
      const view = new DataView(buf);
      const n = msg.serialize(view, 0);
      const bytes = new Uint8Array(buf, 0, n);
      console.log(encodeHex(bytes));
      break;
    }

    case 'deser:EnvelopeMessage': {
      const data = decodeHex(hexInput);
      const view = new DataView(data.buffer, data.byteOffset, data.byteLength);
      const [msg] = EnvelopeMessage.deserialize(view, 0);
      console.log(` + "`Code=${msg.code}`" + `);
      console.log(` + "`Counter=${msg.counter}`" + `);
      break;
    }

    default:
      console.error(` + "`Unknown op:type ${op}:${typ}`" + `);
      process.exit(1);
  }
}

main();
`
