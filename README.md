<p align="center">
  <img src="images/logo.png" alt="arpack logo" width="240">
</p>

# ArPack

Binary serialization code generator for Go and C#. Define messages once as Go structs — get zero-allocation `Marshal`/`Unmarshal` for Go and `unsafe` pointer-based `Serialize`/`Deserialize` for C#.

## Features

- **Single source of truth** — define messages in Go, generate both Go and C# code
- **Float quantization** — compress `float32`/`float64` to 8 or 16 bits with a `pack` struct tag
- **Boolean packing** — consecutive `bool` fields are packed into single bytes (up to 8 per byte)
- **Enums** — `type Opcode uint16` + `const` block becomes a C# `enum`
- **Nested types, fixed arrays, slices** — full support for complex message structures
- **Cross-language binary compatibility** — Go and C# produce identical wire formats

## When to use

ArPack is designed for real-time multiplayer games and other latency-sensitive systems where a Go backend talks to a C# client over a binary protocol.

Typical setups:

- **[Nakama](https://heroiclabs.com/nakama/) + Unity** — define all network messages in Go, generate C# structs for Unity. Both sides share the exact same wire format with no reflection or boxing.
- **Custom Go game server + Unity** — roll your own server without pulling in a serialization framework. ArPack generates plain `Marshal`/`Unmarshal` methods with zero allocations on the hot path.
- **Any Go service + .NET client** — works anywhere you control both ends and want a compact binary protocol without Protobuf's runtime overhead or code-gen complexity.

ArPack is a poor fit if you need schema evolution (adding/removing fields without redeploying both sides) — use Protobuf or FlatBuffers instead.

## Installation

```bash
go install github.com/edmand46/arpack/cmd/arpack@latest
```

## Usage

```bash
arpack -in messages.go -out-go ./gen -out-cs ../Unity/Assets/Scripts
```

| Flag | Description |
|---|---|
| `-in` | Input Go file with struct definitions (required) |
| `-out-go` | Output directory for generated Go code |
| `-out-cs` | Output directory for generated C# code |
| `-cs-namespace` | C# namespace (default: `Arpack.Messages`) |

**Output files:**
- Go: `{name}_gen.go`
- C#: `{Name}.gen.cs`

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

| Type | Wire Size |
|---|---|
| `bool` (packed) | 1 bit (up to 8 per byte) |
| `int8`, `uint8` | 1 byte |
| `int16`, `uint16` | 2 bytes |
| `int32`, `uint32`, `float32` | 4 bytes |
| `int64`, `uint64`, `float64` | 8 bytes |
| `string` | 2-byte length prefix + UTF-8 |
| `[N]T` | N × sizeof(T) |
| `[]T` | 2-byte length prefix + N × sizeof(T) |

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

## Wire Format

- Little-endian byte order
- No message framing — fields are written in declaration order
- Variable-length fields (`string`, `[]T`) prefixed with `uint16` length
- Booleans packed as bitfields (LSB first, up to 8 per byte)
- Quantized floats stored as `uint8` or `uint16`

## Running Tests

```bash
# Unit tests 
go test ./parser/... ./generator/...

# End-to-end cross-language tests
go test ./e2e/...
```
