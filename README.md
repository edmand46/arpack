<p align="center">
  <img src="images/logo.png" alt="arpack logo" width="240">
</p>

# ArPack

![Tests](https://github.com/edmand46/arpack/actions/workflows/tests.yml/badge.svg)
![GitHub Tag](https://img.shields.io/github/v/tag/edmand46/arpack)
![GitHub License](https://img.shields.io/github/license/edmand46/arpack)


Binary serialization code generator for Go, C#, TypeScript, Lua, and C. Define messages once as Go structs — get zero-allocation `Marshal`/`Unmarshal` for Go, `unsafe` pointer-based `Serialize`/`Deserialize` for C#, `DataView`-based serialization for TypeScript/browser, pure Lua implementation for Defold/LuaJIT, and explicit encode/decode functions for C.

## Features

- **Single source of truth** — define messages in Go, generate code for Go, C#, TypeScript, Lua, and C
- **Float quantization** — compress `float32`/`float64` to 8 or 16 bits with a `pack` struct tag
- **Boolean packing** — consecutive `bool` fields are packed into single bytes (up to 8 per byte)
- **Enums** — `type Opcode uint16` + `const` block becomes C#/TypeScript enums
- **Nested types, fixed arrays, slices** — full support for complex message structures
- **Cross-language binary compatibility** — Go, C#, TypeScript, Lua, and C produce identical wire formats
- **Browser support** — TypeScript target uses native DataView API for zero-dependency serialization

## When to use

ArPack is designed for real-time multiplayer games and other latency-sensitive systems where a Go backend talks to a C# client over a binary protocol.

Typical setups:

- **[Nakama](https://heroiclabs.com/nakama/) + Unity** — define all network messages in Go, generate C# structs for Unity. Both sides share the exact same wire format with no reflection or boxing.
- **Custom Go game server + Unity** — roll your own server without pulling in a serialization framework. ArPack generates plain `Marshal`/`Unmarshal` methods with zero allocations on the hot path.
- **Any Go service + .NET client** — works anywhere you control both ends and want a compact binary protocol without Protobuf's runtime overhead or code-gen complexity.
- **Go backend + Browser/WebSocket** — generate TypeScript classes for browser-based clients. Uses native DataView API with zero dependencies.
- **Go backend + Defold/Lua** — generate Lua modules for Defold game engine. Pure Lua implementation compatible with LuaJIT.
- **Go backend + Defold/C** — generate C code for Defold native extensions. Maximum performance for Defold games with C extensions.

## Installation

```bash
go install github.com/edmand46/arpack/cmd/arpack@latest
```

## Usage

```bash
# Generate Go + C# + TypeScript
arpack -in messages.go -out-go ./gen -out-cs ../Unity/Assets/Scripts -out-ts ./web/src/messages

# Generate only TypeScript
arpack -in messages.go -out-ts ./web/src/messages

# Generate only Lua (for Defold)
arpack -in messages.go -out-lua ./defold/scripts/messages

# Generate C for Defold native extension
arpack -in messages.go -out-c ./defold/extension/src
```

| Flag | Description |
|---|---|
| `-in` | Input Go file with struct definitions (required) |
| `-out-go` | Output directory for generated Go code |
| `-out-cs` | Output directory for generated C# code |
| `-out-ts` | Output directory for generated TypeScript code |
| `-out-lua` | Output directory for generated Lua code |
| `-out-c` | Output directory for generated C code (for Defold native extensions) |
| `-cs-namespace` | C# namespace (default: `Arpack.Messages`) |

**Output files:**
- Go: `{name}_gen.go`
- C#: `{Name}.gen.cs`
- TypeScript: `{Name}.gen.ts`
- Lua: `{name}_gen.lua` (snake_case for Lua `require()` compatibility)
- C: `{name}.gen.h` and `{name}.gen.c` (snake_case for C conventions)

## Schema Definition

Messages are defined as Go structs in a single `.go` file:

```go
package messages

// Quantized 3D vector — 6 bytes instead of 12
type Vector3 struct {
    X float32 `pack:"min=-500,max=500,bits=16"`
    Y float32 `pack:"min=-500,max=500,bits=16"`
    Z float32 `pack:"min=-500,max=500,bits=16"`
}

// Enum
type Opcode uint16

const (
    OpcodeUnknown   Opcode = iota
    OpcodeAuthorize
    OpcodeJoinRoom
)

type MoveMessage struct {
    Position  Vector3    // nested type
    Velocity  [3]float32 // fixed-length array
    Waypoints []Vector3  // variable-length slice
    PlayerID  uint32
    Active    bool       // 3 consecutive bools →
    Visible   bool       //   packed into 1 byte
    Ghost     bool
    Name      string
}
```

### Supported Types

| Type | Wire Size | Lua Support |
|---|---|---|
| `bool` (packed) | 1 bit (up to 8 per byte) | ✓ (uses BitOp library) |
| `int8`, `uint8` | 1 byte | ✓ |
| `int16`, `uint16` | 2 bytes | ✓ |
| `int32`, `uint32`, `float32` | 4 bytes | ✓ |
| `int64`, `uint64` | 8 bytes | ✗ (LuaJIT limitation) |
| `float64` | 8 bytes | ✓ |
| `string` | 2-byte length prefix + UTF-8 | ✓ |
| `[N]T` | N × sizeof(T) | ✓ |
| `[]T` | 2-byte length prefix + N × sizeof(T) | ✓ |

**Note:** `int64`/`uint64` are not supported in Lua target. LuaJIT (used by Defold) represents numbers as double-precision floats, which can only safely represent integers up to 2^53. Use `int32`/`uint32` instead.

### Float Quantization

Use the `pack` struct tag to compress floats:

```go
X float32 `pack:"min=-500,max=500,bits=16"`  // 2 bytes instead of 4
Y float32 `pack:"min=0,max=1,bits=8"`        // 1 byte instead of 4
```

| Parameter | Description |
|---|---|
| `min` | Minimum expected value |
| `max` | Maximum expected value |
| `bits` | Target size: `8` (uint8) or `16` (uint16) |

Values are linearly mapped: `encoded = (value - min) / (max - min) * maxUint`.

## Generated Code

### Go

```go
func (m *MoveMessage) Marshal(buf []byte) []byte
func (m *MoveMessage) Unmarshal(data []byte) (int, error)
```

`Marshal` appends to the buffer and returns it. `Unmarshal` reads from the buffer and returns bytes consumed.

### C#

```csharp
public unsafe int Serialize(byte* buffer)
public static unsafe int Deserialize(byte* buffer, out MoveMessage msg)
```

Uses unsafe pointers for zero-copy serialization. Returns bytes written/consumed.

### TypeScript

```typescript
export class MoveMessage {
  position: Vector3 = new Vector3();
  velocity: number[] = new Array<number>(3).fill(0);
  waypoints: Vector3[] = [];
  playerId: number = 0;
  active: boolean = false;
  visible: boolean = false;
  ghost: boolean = false;
  name: string = "";

  serialize(view: DataView, offset: number): number
  static deserialize(view: DataView, offset: number): [MoveMessage, number]
}
```

Uses native DataView API for browser-compatible serialization with zero dependencies. Returns bytes written/consumed.

**Note:** TypeScript field names are converted to camelCase (e.g., `PlayerID` → `playerId`).

### Lua

```lua
local messages = require("messages.messages_gen")

-- Create message
local msg = messages.new_move_message()
msg.player_id = 123
msg.active = true

-- Serialize
local data = messages.serialize_move_message(msg)

-- Deserialize
local decoded, bytes_read = messages.deserialize_move_message(data, 1)
```

Uses pure Lua with inline helper functions for byte manipulation. Compatible with LuaJIT (Defold). All identifiers use snake_case (e.g., `MoveMessage` → `move_message`, `PlayerID` → `player_id`).

**Requirements:** The generated Lua code requires the [BitOp library](https://bitop.luajit.org/) for bit manipulation. This library is included in LuaJIT (used by Defold).

**Limitations:**
- Lua target does not support `int64`/`uint64` types. Use `int32`/`uint32` instead. This is because LuaJIT represents numbers as double-precision floats, which can only safely represent integers up to 2^53.
- Variable-length fields use `uint16` length prefixes, so `string` byte length and `[]T` element count must not exceed `65535`. Serialization raises an error if the limit is exceeded.
- Deserialization raises Lua errors on malformed or truncated input. If you need a recoverable boundary, wrap decode calls in `pcall(...)`.
- Generated file uses snake_case naming (e.g., `messages_gen.lua`) for proper Lua `require()` resolution.

### C

```c
#include "messages.gen.h"

// Fixed-size message (no context needed)
sample_envelope_message msg = {
    .code = sample_opcode_authorize,
    .counter = 42
};

uint8_t buf[64];
size_t written;
arpack_status status = sample_envelope_message_encode(&msg, buf, sizeof(buf), &written);

// Variable-length message (requires decode context)
sample_spawn_message_decode_ctx ctx = {
    .tags_data = tags_buffer,
    .tags_cap = MAX_TAGS
};
sample_spawn_message decoded;
status = sample_spawn_message_decode(&decoded, buf, buf_len, &ctx, &read);
```

Generates two files: `{name}.gen.h` (declarations) and `{name}.gen.c` (implementations). Uses explicit encode/decode functions with bounds checking. All symbols are prefixed with `{name}_` to avoid collisions.

**API Shape:**
- Fixed-size messages: `{name}_{msg}_min_size()`, `{name}_{msg}_encode()`, `{name}_{msg}_decode()`
- Variable-length messages: Additional `{name}_{msg}_size()` and decode context struct
- Strings and byte slices are views into the input buffer (zero-copy)
- Other slices require caller-provided storage via decode context

**Limitations:**
- C11 standard required
- Variable-length slice fields require caller-provided storage (no hidden allocations)
- Wire format is not a packed C struct — use the generated encode/decode functions

## Wire Format

- Little-endian byte order
- No message framing — fields are written in declaration order
- Variable-length fields (`string`, `[]T`) prefixed with `uint16` length
- Booleans packed as bitfields (LSB first, up to 8 per byte)
- Quantized floats stored as `uint8` or `uint16`

## Benchmarks 

### Go Results (M3 Max)
```
BenchmarkArPack_Marshal-16        382568360    9.5 ns/op    5065 MB/s    0 B/op    0 allocs/op
BenchmarkArPack_Unmarshal-16       98895892   34.6 ns/op    1388 MB/s   40 B/op    2 allocs/op
BenchmarkProto_Marshal-16          21989466  163.6 ns/op     416 MB/s    0 B/op    0 allocs/op
BenchmarkProto_Unmarshal-16        13950333  256.9 ns/op     265 MB/s  248 B/op    7 allocs/op
BenchmarkFlatBuffers_Marshal-16    16297458  221.4 ns/op     687 MB/s    0 B/op    0 allocs/op
BenchmarkFlatBuffers_Unmarshal-16  56095480   64.8 ns/op    2345 MB/s   24 B/op    1 allocs/op
```

| Format | Size |
|---|---|
| ArPack | 48 bytes |
| Protobuf | 68 bytes |
| FlatBuffers | 152 bytes |

```bash
go test ./benchmarks/... -bench=. -benchmem
```

### Unity Mono (M3 Max)

```
ArPack Serialize:           96.7 ns/op |    0 B/op
ArPack Deserialize:        205.4 ns/op |    0 B/op
Proto Serialize (alloc):   930.2 ns/op |    0 B/op
Proto Deserialize (alloc): 1621.2 ns/op |   29 B/op
Proto Serialize (reuse):   652.7 ns/op |    0 B/op
```

ArPack serialize is ~10× faster than Protobuf in Unity. Protobuf deserialize allocates on every call — a GC pressure source in hot game loops. ArPack deserialize is zero-alloc.

```bash
make gen-unity
# then attach BenchmarkRunner to any GameObject in SampleScene and press Play
```

## Running Tests

```bash
# Unit tests
go test ./parser/... ./generator/...

# End-to-end cross-language tests
go test ./e2e/...
```
