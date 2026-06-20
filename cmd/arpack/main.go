package main

import (
	"errors"
	"flag"
	"fmt"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/edmand46/arpack/generator"
	"github.com/edmand46/arpack/parser"
)

type genRequest struct {
	in        string
	outGo     string
	outCS     string
	outTS     string
	outLua    string
	namespace string
}

type genFile struct {
	dir  string
	path string
	data []byte
}

func main() {
	in := flag.String("in", "", "input Go file with struct definitions")
	outGo := flag.String("out-go", "", "output directory for generated Go code")
	outCS := flag.String("out-cs", "", "output directory for generated C# code")
	outTS := flag.String("out-ts", "", "output directory for generated TypeScript code")
	outLua := flag.String("out-lua", "", "output directory for generated Lua code")
	namespace := flag.String("cs-namespace", "Arpack.Messages", "C# namespace")
	flag.Parse()

	if *in == "" {
		log.Fatal("arpack: -in is required")
	}

	if *outGo == "" && *outCS == "" && *outTS == "" && *outLua == "" {
		log.Fatal("arpack: at least one of -out-go, -out-cs, -out-ts, or -out-lua is required")
	}

	schema, err := parser.ParseSchemaFile(*in)
	if err != nil {
		log.Fatalf("arpack: parse error: %v", err)
	}
	if len(schema.Messages) == 0 && len(schema.Enums) == 0 {
		log.Fatalf("arpack: no structs or enums found in %s", *in)
	}

	files, notices, err := buildOutputs(schema, genRequest{
		in:        *in,
		outGo:     *outGo,
		outCS:     *outCS,
		outTS:     *outTS,
		outLua:    *outLua,
		namespace: *namespace,
	})
	for _, n := range notices {
		log.Printf("arpack: %s", n)
	}
	if err != nil {
		log.Fatalf("arpack: %v", err)
	}

	if err := writeOutputs(files); err != nil {
		log.Fatalf("arpack: %v", err)
	}
	for _, f := range files {
		fmt.Printf("arpack: wrote %s\n", f.path)
	}
}

// buildOutputs generates every requested target in memory. Callers write the
// returned files only when all targets succeed, so a failing target (e.g. an
// int64 field with -out-lua) never leaves a partially updated codegen tree.
func buildOutputs(schema parser.Schema, req genRequest) (files []genFile, notices []string, err error) {
	msgs := schema.Messages

	schemaPkg := schema.PackageName
	if len(msgs) > 0 {
		schemaPkg = msgs[0].PackageName
	}
	if schemaPkg == "" {
		schemaPkg = "main"
	}

	baseName := strings.TrimSuffix(filepath.Base(req.in), ".go")

	if req.outGo != "" {
		if len(msgs) == 0 {
			notices = append(notices, "skipping Go output: schema has no messages (enums are already declared in the Go schema source)")
		} else {
			pkgName := filepath.Base(req.outGo)
			if pkgName == "." || pkgName == "" {
				pkgName = schemaPkg
			}
			// Replace hyphens with underscores for valid Go package names
			pkgName = strings.ReplaceAll(pkgName, "-", "_")

			if !token.IsIdentifier(pkgName) || token.IsKeyword(pkgName) {
				notices = append(notices, fmt.Sprintf("warning: -out-go directory name %q is not a valid Go package name; using %q", pkgName, schemaPkg))
				pkgName = schemaPkg
				if !token.IsIdentifier(pkgName) || token.IsKeyword(pkgName) {
					return nil, notices, fmt.Errorf("schema package name %q is also not a valid Go package name", pkgName)
				}
			}

			src, genErr := generator.GenerateGoSchema(schema, pkgName)
			if genErr != nil {
				return nil, notices, fmt.Errorf("generating Go: %w", genErr)
			}
			files = append(files, genFile{dir: req.outGo, path: filepath.Join(req.outGo, baseName+"_gen.go"), data: src})
		}
	}

	if req.outCS != "" {
		src, genErr := generator.GenerateCSharpSchema(schema, req.namespace)
		if genErr != nil {
			return nil, notices, fmt.Errorf("generating C#: %w", genErr)
		}
		files = append(files, genFile{dir: req.outCS, path: filepath.Join(req.outCS, toTitle(baseName)+".gen.cs"), data: src})
	}

	if req.outTS != "" {
		src, genErr := generator.GenerateTypeScriptSchema(schema)
		if genErr != nil {
			return nil, notices, fmt.Errorf("generating TypeScript: %w", genErr)
		}
		files = append(files, genFile{dir: req.outTS, path: filepath.Join(req.outTS, toTitle(baseName)+".gen.ts"), data: src})
	}

	if req.outLua != "" {
		src, genErr := generator.GenerateLuaSchema(schema, baseName)
		if genErr != nil {
			return nil, notices, fmt.Errorf("generating Lua: %w", genErr)
		}
		// Use snake_case filename for Lua require() compatibility
		files = append(files, genFile{dir: req.outLua, path: filepath.Join(req.outLua, toSnakeCase(baseName)+"_gen.lua"), data: src})
	}

	return files, notices, nil
}

type tempOutputWriter func(genFile) (string, error)
type outputRenamer func(oldpath, newpath string) error

type stagedOutput struct {
	finalPath string
	tempPath  string
}

type replacedOutput struct {
	finalPath  string
	backupPath string
	existed    bool
}

func writeOutputs(files []genFile) error {
	return writeOutputsWith(files, writeTempOutput, os.Rename)
}

func writeOutputsWith(files []genFile, writeTemp tempOutputWriter, rename outputRenamer) error {
	for _, f := range files {
		if err := os.MkdirAll(f.dir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", f.dir, err)
		}
	}

	staged := make([]stagedOutput, 0, len(files))
	defer func() {
		for _, f := range staged {
			if f.tempPath != "" {
				_ = os.Remove(f.tempPath)
			}
		}
	}()

	for _, f := range files {
		tempPath, err := writeTemp(f)
		if err != nil {
			return err
		}
		staged = append(staged, stagedOutput{finalPath: f.path, tempPath: tempPath})
	}

	replaced := make([]replacedOutput, 0, len(staged))
	committed := false
	defer func() {
		if !committed {
			rollbackReplacements(replaced)
		}
		for _, f := range replaced {
			if f.backupPath != "" {
				_ = os.Remove(f.backupPath)
			}
		}
	}()

	for i := range staged {
		repl, err := replaceWithBackup(staged[i], rename)
		if err != nil {
			return err
		}
		staged[i].tempPath = ""
		replaced = append(replaced, repl)
	}

	committed = true
	return nil
}

func replaceWithBackup(staged stagedOutput, rename outputRenamer) (replacedOutput, error) {
	repl := replacedOutput{finalPath: staged.finalPath}
	if _, err := os.Stat(staged.finalPath); err == nil {
		backupPath, err := tempPathInDir(filepath.Dir(staged.finalPath), "."+filepath.Base(staged.finalPath)+".*.bak")
		if err != nil {
			return repl, fmt.Errorf("create backup path for %s: %w", staged.finalPath, err)
		}
		if err := rename(staged.finalPath, backupPath); err != nil {
			return repl, fmt.Errorf("backup %s: %w", staged.finalPath, err)
		}
		repl.existed = true
		repl.backupPath = backupPath
	} else if !errors.Is(err, os.ErrNotExist) {
		return repl, fmt.Errorf("stat %s: %w", staged.finalPath, err)
	}

	if err := rename(staged.tempPath, staged.finalPath); err != nil {
		if repl.existed {
			_ = rename(repl.backupPath, repl.finalPath)
			repl.backupPath = ""
		}
		return repl, fmt.Errorf("replace %s: %w", staged.finalPath, err)
	}

	return repl, nil
}

func tempPathInDir(dir, pattern string) (string, error) {
	tmp, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	path := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	if err := os.Remove(path); err != nil {
		return "", err
	}
	return path, nil
}

func rollbackReplacements(replaced []replacedOutput) {
	for i := len(replaced) - 1; i >= 0; i-- {
		f := replaced[i]
		if f.existed {
			_ = os.Remove(f.finalPath)
			if f.backupPath != "" {
				_ = os.Rename(f.backupPath, f.finalPath)
			}
		} else {
			_ = os.Remove(f.finalPath)
		}
	}
}

func writeTempOutput(f genFile) (string, error) {
	name := "." + filepath.Base(f.path) + ".*.tmp"
	tmp, err := os.CreateTemp(f.dir, name)
	if err != nil {
		return "", fmt.Errorf("create temp for %s: %w", f.path, err)
	}

	tempPath := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = tmp.Close()
			_ = os.Remove(tempPath)
		}
	}()

	if _, err := tmp.Write(f.data); err != nil {
		return "", fmt.Errorf("write temp for %s: %w", f.path, err)
	}
	if err := tmp.Chmod(0644); err != nil {
		return "", fmt.Errorf("chmod temp for %s: %w", f.path, err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("close temp for %s: %w", f.path, err)
	}

	ok = true
	return tempPath, nil
}

func toTitle(s string) string {
	if s == "" {
		return ""
	}

	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func toSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	var prevUpper bool

	for i, c := range s {
		isUpper := c >= 'A' && c <= 'Z'

		if i > 0 && isUpper {
			nextLower := false
			if i+1 < len(s) {
				nextChar := rune(s[i+1])
				nextLower = nextChar >= 'a' && nextChar <= 'z'
			}

			if !prevUpper || nextLower {
				b.WriteByte('_')
			}
		}

		b.WriteRune(c)
		prevUpper = isUpper
	}

	return strings.ToLower(b.String())
}
