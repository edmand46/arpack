package main

import (
	"errors"
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

func parseSchema(t *testing.T, src string) (parser.Schema, string) {
	t.Helper()
	path := writeSchema(t, src)
	schema, err := parser.ParseSchemaFile(path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return schema, path
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
	schema, in := parseSchema(t, structSchema)
	out := t.TempDir()
	files, notices, err := buildOutputs(schema, genRequest{
		in:        in,
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
	schema, in := parseSchema(t, luaUnsupportedSchema)
	out := t.TempDir()
	files, _, err := buildOutputs(schema, genRequest{
		in:        in,
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
	schema, in := parseSchema(t, enumOnlySchema)
	out := t.TempDir()
	files, notices, err := buildOutputs(schema, genRequest{
		in:        in,
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
	schema, in := parseSchema(t, structSchema)
	out := t.TempDir()
	files, notices, err := buildOutputs(schema, genRequest{
		in:    in,
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

func TestWriteOutputs_LaterTempWriteFailureLeavesExistingOutputUnchanged(t *testing.T) {
	dir := t.TempDir()
	firstPath := filepath.Join(dir, "first.gen.go")
	secondPath := filepath.Join(dir, "second.gen.ts")
	if err := os.WriteFile(firstPath, []byte("existing first"), 0644); err != nil {
		t.Fatalf("seed first output: %v", err)
	}

	writeCalls := 0
	errWriteSecond := errors.New("simulated second temp write failure")
	err := writeOutputsWith(
		[]genFile{
			{dir: dir, path: firstPath, data: []byte("new first")},
			{dir: dir, path: secondPath, data: []byte("new second")},
		},
		func(f genFile) (string, error) {
			writeCalls++
			if writeCalls == 2 {
				return "", errWriteSecond
			}
			tmp, err := os.CreateTemp(f.dir, "."+filepath.Base(f.path)+".*.tmp")
			if err != nil {
				return "", err
			}
			if _, err := tmp.Write(f.data); err != nil {
				_ = tmp.Close()
				_ = os.Remove(tmp.Name())
				return "", err
			}
			if err := tmp.Close(); err != nil {
				_ = os.Remove(tmp.Name())
				return "", err
			}
			return tmp.Name(), nil
		},
		os.Rename,
	)
	if !errors.Is(err, errWriteSecond) {
		t.Fatalf("writeOutputsWith error = %v, want %v", err, errWriteSecond)
	}

	got, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatalf("read first output: %v", err)
	}
	if string(got) != "existing first" {
		t.Fatalf("first output = %q, want existing content", got)
	}
	if _, err := os.Stat(secondPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("second output stat error = %v, want not exist", err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".*.tmp"))
	if err != nil {
		t.Fatalf("glob temps: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected temp files to be cleaned up, found %v", matches)
	}
}

func TestWriteOutputs_LaterRenameFailureRollsBackEarlierReplacements(t *testing.T) {
	dir := t.TempDir()
	firstPath := filepath.Join(dir, "first.gen.go")
	secondPath := filepath.Join(dir, "second.gen.ts")
	if err := os.WriteFile(firstPath, []byte("existing first"), 0644); err != nil {
		t.Fatalf("seed first output: %v", err)
	}
	if err := os.WriteFile(secondPath, []byte("existing second"), 0644); err != nil {
		t.Fatalf("seed second output: %v", err)
	}

	tempForFinal := map[string]string{}
	writeTemp := func(f genFile) (string, error) {
		tmp, err := os.CreateTemp(f.dir, "."+filepath.Base(f.path)+".*.tmp")
		if err != nil {
			return "", err
		}
		if _, err := tmp.Write(f.data); err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
			return "", err
		}
		if err := tmp.Close(); err != nil {
			_ = os.Remove(tmp.Name())
			return "", err
		}
		tempForFinal[f.path] = tmp.Name()
		return tmp.Name(), nil
	}

	errReplaceSecond := errors.New("simulated second replace failure")
	rename := func(oldpath, newpath string) error {
		if oldpath == tempForFinal[secondPath] {
			return errReplaceSecond
		}
		return os.Rename(oldpath, newpath)
	}

	err := writeOutputsWith(
		[]genFile{
			{dir: dir, path: firstPath, data: []byte("new first")},
			{dir: dir, path: secondPath, data: []byte("new second")},
		},
		writeTemp,
		rename,
	)
	if !errors.Is(err, errReplaceSecond) {
		t.Fatalf("writeOutputsWith error = %v, want %v", err, errReplaceSecond)
	}

	gotFirst, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatalf("read first output: %v", err)
	}
	if string(gotFirst) != "existing first" {
		t.Fatalf("first output = %q, want rollback to existing content", gotFirst)
	}
	gotSecond, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatalf("read second output: %v", err)
	}
	if string(gotSecond) != "existing second" {
		t.Fatalf("second output = %q, want rollback to existing content", gotSecond)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".*"))
	if err != nil {
		t.Fatalf("glob temp files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected temp/backup files to be cleaned up, found %v", matches)
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

func TestToSnakeCase(t *testing.T) {
	cases := map[string]string{
		"":            "",
		"messages":    "messages",
		"NetMsg":      "net_msg",
		"MoveMessage": "move_message",
		"HTTPServer":  "http_server",
	}
	for in, want := range cases {
		if got := toSnakeCase(in); got != want {
			t.Errorf("toSnakeCase(%q) = %q, want %q", in, got, want)
		}
	}
}
