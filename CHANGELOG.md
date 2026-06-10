# Changelog

## [Unreleased]

### Changed

- **Quant float64 canonicalization**: All four languages (Go, C#, TypeScript, Lua) now use identical float64 arithmetic with truncation toward zero for quantized float values. Previously, C# used `float` (32-bit) literals and Lua used `math.floor` (rounds toward negative infinity for negative values), causing wire divergence for negative quantized values.
- **C# Deserialize signature** (breaking): The `Deserialize` method now takes `int length` parameter for explicit bounds checking. Update call sites: `X.Deserialize(ptr, out msg)` → `X.Deserialize(ptr, data.Length, out msg)`.
- **C# Serialize signature** (breaking): `Serialize` now takes `int length` and performs explicit write bounds checks — an undersized buffer throws `ArgumentException` instead of silently corrupting memory. Update call sites: `msg.Serialize(ptr)` → `msg.Serialize(ptr, buf.Length)`.
- **C#/TS output filenames** (breaking): output filenames now preserve internal casing and only uppercase the first letter (`netMsg.go` → `NetMsg.gen.cs`; previously `Netmsg.gen.cs`). Delete previously generated files with the old casing to avoid duplicate definitions.
- **TypeScript namespace parameter removed**: `GenerateTypeScriptSchema` no longer takes a namespace parameter. (Existing change, now documented.)
- **Parser now rejects nested collections**: Arrays/slices inside arrays/slices (`[][]float64`, `[3][]int32`) are now rejected with a clear error.
- **Parser now rejects embedded struct fields**: Anonymous/embedded fields in structs now produce a clear error.
- **Enum-only schemas allowed**: Schemas with only enums and no structs no longer cause a fatal error. The Go target is skipped for enum-only schemas with a notice (enum declarations already live in the Go schema source); C#/TS/Lua emit the enums.
- **Atomic multi-target output**: the CLI now generates all requested targets in memory and writes files only if every target succeeds. A failing target (e.g. `int64` with `-out-lua`) no longer leaves partially updated files on disk.
- **`Message.MinWireSize()`**: now accounts for bool bit-packing, matching the layout generators emit; the generators use this single implementation.
- **Parser error messages**: removed duplicated `struct X:` / `field Y:` prefixes.
- **Conditional `encoding/binary` import**: Go generated code only imports `encoding/binary` when needed (quantized 16-bit fields, slices, non-byte primitives).
- **Go package name validation**: The `-out-go` directory name is now validated as a valid Go package identifier. Invalid names (e.g., keywords, names with dots) fall back to the schema's package name.
- **TS/Lua nested struct variable naming**: Fixed invalid TypeScript destructuring identifiers and Lua variable names when nested structs appear inside fixed arrays (e.g., `[3]NestedStruct`).
- **C# enum value names**: enum-type-prefix stripping removed entirely — C# enum member names now always match the schema (and TypeScript) verbatim. Previously stripping could produce invalid identifiers (leading digits) or colliding members.
- **Lua bounds checking**: `check_bounds` calls are now wired into all primitive deserialize paths for defense-in-depth.

### Added

- **Boundary-value e2e tests**: Added `QuantTestMessage` to test schemas and `TestE2E_QuantBoundaryValues` that verifies all four languages produce identical wire output for edge-case quantized values.
- **Non-Go-pivot e2e tests**: Added `TestE2E_NonGoPivot` for C#↔TS, C#↔Lua, and TS↔Lua roundtrip tests.
- **Truncated-input e2e tests**: Added `TestE2E_TruncatedInput` that verifies each language's deserializer errors on empty/truncated input rather than producing garbage or panicking.
