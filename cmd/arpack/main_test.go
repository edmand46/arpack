package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edmand46/arpack/parser"
)

func writeSchema(t *testing.T, src string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "messages.go")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func parseSchema(t *testing.T, src string) parser.Schema {
	t.Helper()
	path := writeSchema(t, src)
	schema, err := parser.ParseSchemaFile(path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return schema
}

const structSchema = `package messages

type Ping struct {
	ID uint32
}
`

const enumOnlySchema = `package messages

type Opcode uint16

const (
	OpcodeUnknown Opcode = iota
	OpcodeJoin
)
`

const luaUnsupportedSchema = `package messages

type Big struct {
	EntityID uint64
}
`

func TestBuildOutputs_AllTargets(t *testing.T) {
	schema := parseSchema(t, structSchema)
	out := t.TempDir()
	files, notices, err := buildOutputs(schema, genRequest{
		name:      "messages",
		outGo:     filepath.Join(out, "messages"),
		outCS:     filepath.Join(out, "cs"),
		outTS:     filepath.Join(out, "ts"),
		outLua:    filepath.Join(out, "lua"),
		namespace: "Arpack.Messages",
	})
	if err != nil {
		t.Fatalf("buildOutputs: %v", err)
	}
	if len(notices) != 0 {
		t.Fatalf("unexpected notices: %v", notices)
	}
	if len(files) != 4 {
		t.Fatalf("expected 4 files, got %d", len(files))
	}
	wantSuffixes := []string{"messages_gen.go", "Messages.gen.cs", "Messages.gen.ts", "messages_gen.lua"}
	for i, suffix := range wantSuffixes {
		if !strings.HasSuffix(files[i].path, suffix) {
			t.Errorf("file %d: expected suffix %q, got %q", i, suffix, files[i].path)
		}
	}
}

func TestBuildOutputs_FailingTargetProducesNoFiles(t *testing.T) {
	// Lua rejects uint64; the Go/C#/TS targets would succeed, but a failing
	// target must abort the whole run with zero files (all-or-nothing).
	schema := parseSchema(t, luaUnsupportedSchema)
	out := t.TempDir()
	files, _, err := buildOutputs(schema, genRequest{
		name:      "messages",
		outGo:     filepath.Join(out, "messages"),
		outCS:     filepath.Join(out, "cs"),
		outTS:     filepath.Join(out, "ts"),
		outLua:    filepath.Join(out, "lua"),
		namespace: "Arpack.Messages",
	})
	if err == nil {
		t.Fatal("expected error for uint64 with -out-lua, got nil")
	}
	if !strings.Contains(err.Error(), "generating Lua") {
		t.Fatalf("expected Lua generation error, got %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected no files on failure, got %d", len(files))
	}
}

func TestBuildOutputs_EnumOnlySkipsGo(t *testing.T) {
	schema := parseSchema(t, enumOnlySchema)
	out := t.TempDir()
	files, notices, err := buildOutputs(schema, genRequest{
		name:      "messages",
		outGo:     filepath.Join(out, "messages"),
		outCS:     filepath.Join(out, "cs"),
		namespace: "Arpack.Messages",
	})
	if err != nil {
		t.Fatalf("buildOutputs: %v", err)
	}
	if len(files) != 1 || !strings.HasSuffix(files[0].path, ".gen.cs") {
		t.Fatalf("expected only the C# file, got %v", files)
	}
	if len(notices) != 1 || !strings.Contains(notices[0], "skipping Go output") {
		t.Fatalf("expected a skipping-Go notice, got %v", notices)
	}
	if !strings.Contains(string(files[0].data), "enum Opcode") {
		t.Fatalf("expected C# enum output, got:\n%s", files[0].data)
	}
}

func TestBuildOutputs_KeywordDirFallsBackToSchemaPackage(t *testing.T) {
	schema := parseSchema(t, structSchema)
	out := t.TempDir()
	files, notices, err := buildOutputs(schema, genRequest{
		name:  "messages",
		outGo: filepath.Join(out, "go"), // "go" is a Go keyword
	})
	if err != nil {
		t.Fatalf("buildOutputs: %v", err)
	}
	if len(notices) != 1 || !strings.Contains(notices[0], "not a valid Go package name") {
		t.Fatalf("expected invalid-package notice, got %v", notices)
	}
	if !strings.Contains(string(files[0].data), "package messages") {
		t.Fatalf("expected fallback to schema package, got:\n%s", files[0].data)
	}
}

func TestPickBaseName(t *testing.T) {
	base := pickBaseName("", []string{"/x/y/net_messages.go"})
	if base != "net_messages" {
		t.Fatalf("single input: got %q", base)
	}
	base = pickBaseName("custom", []string{"/x/y/a.go", "/x/y/b.go"})
	if base != "custom" {
		t.Fatalf("name flag: got %q", base)
	}
	base = pickBaseName("", []string{"/x/y/a.go", "/x/y/b.go"})
	if base != "a" {
		t.Fatalf("multiple inputs without -name: got %q, want first input base", base)
	}
}

func TestBuildOutputs_MultiInputDifferentPackages(t *testing.T) {
	dir := t.TempDir()
	framework := filepath.Join(dir, "framework.go")
	game := filepath.Join(dir, "game.go")
	for path, src := range map[string]string{
		framework: `package framework

type Heartbeat struct {
	Time uint32
}
`,
		game: `package game

type PlayerMove struct {
	X float32
	Y float32
}
`,
	} {
		if err := os.WriteFile(path, []byte(src), 0644); err != nil {
			t.Fatal(err)
		}
	}

	schema, err := parser.ParseSchemaFiles([]string{framework, game})
	if err != nil {
		t.Fatalf("ParseSchemaFiles: %v", err)
	}

	out := t.TempDir()
	files, notices, err := buildOutputs(schema, genRequest{
		name:      "all",
		outGo:     filepath.Join(out, "messages"),
		outCS:     filepath.Join(out, "cs"),
		namespace: "Arpack.Messages",
	})
	if err != nil {
		t.Fatalf("buildOutputs: %v", err)
	}
	if len(notices) != 0 {
		t.Fatalf("unexpected notices: %v", notices)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files (one per target), got %d", len(files))
	}
	goData := string(files[0].data)
	if !strings.Contains(goData, "Heartbeat") || !strings.Contains(goData, "PlayerMove") {
		t.Fatalf("expected merged Go output with both messages, got:\n%s", goData)
	}
	csData := string(files[1].data)
	if !strings.Contains(csData, "Heartbeat") || !strings.Contains(csData, "PlayerMove") {
		t.Fatalf("expected merged C# output with both messages, got:\n%s", csData)
	}
}

func TestToTitle(t *testing.T) {
	cases := map[string]string{
		"":         "",
		"messages": "Messages",
		"netMsg":   "NetMsg",
		"NPC":      "NPC",
		"ёжик":     "Ёжик",
	}
	for in, want := range cases {
		if got := toTitle(in); got != want {
			t.Errorf("toTitle(%q) = %q, want %q", in, got, want)
		}
	}
}
