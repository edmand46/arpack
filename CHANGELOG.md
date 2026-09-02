# Changelog

## [1.2.0] - 2026-09-02

### Performance

Go, Apple M3 Max, 5 runs (`make bench-stats`). New `ScoreboardMessage` benchmark: `map[string]int32` + `map[uint16]Vector3`, 8 entries each. Protobuf marshals with `Deterministic: true` so both sides sort keys.

| Benchmark | ArPack | Protobuf |
| --- | ---: | ---: |
| Map Marshal | 332-349 ns/op, 128 B, 1 alloc | 2469-2520 ns/op, 960 B, 56 allocs |
| Map Unmarshal | 273-278 ns/op, 64 B, 8 allocs | 1923-1966 ns/op, 1376 B, 57 allocs |

ArPack map marshal allocates once for the sorted key slice; unmarshal allocates one string per key (decoded map reused via `clear`). FlatBuffers has no native map type and is not compared. Existing `MoveMessage` numbers are unchanged within run-to-run variance (Marshal 8.9-9.0 ns/op, Unmarshal 25.0-25.7 ns/op).

### Added

- **`map[K]V` fields** in all targets. Keys: `string`, explicit-width integers, schema enums; values follow slice-element rules (primitives, strings, enums, nested structs). Wire: `uint16` entry count followed by `(key, value)` pairs in ascending key order (numeric for integers/enums, UTF-8 byte order for strings, identical across languages). Deserializers reject unsorted or duplicate keys. Representation: Go `map[K]V` (decode reuses an existing map via `clear`), C# `Dictionary<K, V>`, TypeScript `Map<K, V>`, Lua table, GML `ds_map` (caller owns and destroys).

### Other

- Parser error for nested collections now reads `nested arrays/slices/maps not supported (v1 limitation)` and also covers `map` values, `[]map` and `[N]map`.
- Lua target rejects 64-bit map keys and values, like other 64-bit fields.

## [1.1.0] - 2026-09-02

### Performance

Go, Apple M3 Max, 5 runs (`make bench-stats`):

| Benchmark | 1.0.2 | 1.1.0 |
| --- | ---: | ---: |
| ArPack Marshal | 9.4 ns/op, 0 allocs | 9.0 ns/op, 0 allocs |
| ArPack Unmarshal | 32.8 ns/op, 40 B, 2 allocs | 25.0 ns/op, 40 B, 2 allocs |

Unmarshal ~25% faster: string fields now compare incoming bytes with `m.Field != string(bytes)` (compiler-optimized, no allocation) instead of the emitted `arpackStringEqualBytes` helper. Regenerate Go outputs. Protobuf/FlatBuffers baselines moved ~10-14% in the same run, so part of the gain is run-to-run variance.

### Breaking

- **C# `Serialize`/`Deserialize` take `int length`** and bounds-check every read and write; an undersized buffer throws `ArgumentException` instead of corrupting memory. `msg.Serialize(ptr)` → `msg.Serialize(ptr, buf.Length)`, `X.Deserialize(ptr, out msg)` → `X.Deserialize(ptr, data.Length, out msg)`.
- **C#/TS output filenames** keep internal casing and only uppercase the first letter (`netMsg.go` → `NetMsg.gen.cs`, was `Netmsg.gen.cs`). Delete old-cased files to avoid duplicate definitions.
- **C# enum members** match the schema verbatim; type-prefix stripping removed (it could produce invalid or colliding identifiers).
- **API removed**: `parser.Message.PackageName` (use `parser.Schema.PackageName`), `parser.Field.CSharpTypeName`, namespace parameter of `GenerateTypeScriptSchema`.

### Key changes

- **Wire parity for quantized floats**: all targets use float64 arithmetic with truncation toward zero. C# used 32-bit literals and Lua used `math.floor`, diverging on negative values.
- **GameMaker Language target**: `-out-gml` (PR #1).
- **Multiple schema inputs**: `-in` is repeatable, files may live in different packages; `-name` sets the output base name, defaulting to the first input file (PR #1, #2).
- **Atomic multi-target output**: all targets are generated in memory and written only if every target succeeds.
- **Parser**: rejects nested collections (`[][]T`, `[3][]T`) and embedded struct fields with clear errors; enum-only schemas allowed (Go target skipped, other targets emit enums); named type chains and aliases resolved through `go/types`.
- **Lua bounds checking**: `check_bounds` on all primitive deserialize paths.
- **e2e tests**: quantized boundary values across all targets, non-Go-pivot roundtrips (C#↔TS, C#↔Lua, TS↔Lua), truncated input errors instead of garbage or panics.

### Other

- `Message.MinWireSize()` accounts for bool bit-packing; generators share the single implementation.
- Go: `encoding/binary` imported only when needed.
- `-out-go` directory name validated as a Go package identifier; invalid names fall back to the schema package name.
- TS/Lua: valid identifiers for nested structs inside fixed arrays (`[3]Nested`).
- TS: slices serialized with an index loop; wire format unchanged.
- Parser errors no longer repeat `struct X:` / `field Y:` prefixes.
- `generator.ToSnakeCase` exported, shared by the Lua generator and the CLI.
- CLI writes files with plain `os.WriteFile`; temp-file/backup/rollback machinery removed.
- Generator internals: ~1000 lines of duplicated per-language code removed; generated output unchanged.
- Benchmarks: README numbers refreshed; Unity benchmark regenerated for the new C# API.
