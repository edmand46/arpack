package main

import (
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

type multiFlag []string

func (f *multiFlag) String() string { return strings.Join(*f, ",") }
func (f *multiFlag) Set(v string) error {
	*f = append(*f, v)
	return nil
}

type genRequest struct {
	name      string
	outGo     string
	outCS     string
	outTS     string
	outLua    string
	outGML    string
	namespace string
}

type genFile struct {
	dir  string
	path string
	data []byte
}

func main() {
	var ins multiFlag
	name := flag.String("name", "", "base name for generated output files (defaults to the first -in file's name)")
	outGo := flag.String("out-go", "", "output directory for generated Go code")
	outCS := flag.String("out-cs", "", "output directory for generated C# code")
	outTS := flag.String("out-ts", "", "output directory for generated TypeScript code")
	outLua := flag.String("out-lua", "", "output directory for generated Lua code")
	outGML := flag.String("out-gml", "", "output directory for generated GameMaker Language (GML) code")
	namespace := flag.String("cs-namespace", "Arpack.Messages", "C# namespace")
	flag.Var(&ins, "in", "input Go file with struct definitions (repeatable)")
	flag.Parse()

	if len(ins) == 0 {
		log.Fatal("arpack: -in is required")
	}

	if *outGo == "" && *outCS == "" && *outTS == "" && *outLua == "" && *outGML == "" {
		log.Fatal("arpack: at least one of -out-go, -out-cs, -out-ts, -out-lua, or -out-gml is required")
	}

	baseName := pickBaseName(*name, ins)

	schema, err := parser.ParseSchemaFiles(ins)
	if err != nil {
		log.Fatalf("arpack: parse error: %v", err)
	}
	if len(schema.Messages) == 0 && len(schema.Enums) == 0 {
		log.Fatalf("arpack: no structs or enums found in %s", strings.Join(ins, ", "))
	}

	files, notices, err := buildOutputs(schema, genRequest{
		name:      baseName,
		outGo:     *outGo,
		outCS:     *outCS,
		outTS:     *outTS,
		outLua:    *outLua,
		outGML:    *outGML,
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

func buildOutputs(schema parser.Schema, req genRequest) (files []genFile, notices []string, err error) {
	msgs := schema.Messages
	schemaPkg := schema.PackageName

	baseName := req.name

	if req.outGo != "" {
		if len(msgs) == 0 {
			notices = append(notices, "skipping Go output: schema has no messages (enums are already declared in the Go schema source)")
		} else {
			pkgName := filepath.Base(req.outGo)
			if pkgName == "." || pkgName == "" {
				pkgName = schemaPkg
			}

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

		files = append(files, genFile{dir: req.outLua, path: filepath.Join(req.outLua, generator.ToSnakeCase(baseName)+"_gen.lua"), data: src})
	}

	if req.outGML != "" {
		src, genErr := generator.GenerateGMLSchema(schema)
		if genErr != nil {
			return nil, notices, fmt.Errorf("generating GML: %w", genErr)
		}

		gmlName := filepath.Base(req.outGML)
		files = append(files, genFile{dir: req.outGML, path: filepath.Join(req.outGML, gmlName+".gml"), data: src})
	}

	return files, notices, nil
}

func writeOutputs(files []genFile) error {
	for _, f := range files {
		if err := os.MkdirAll(f.dir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", f.dir, err)
		}
		if err := os.WriteFile(f.path, f.data, 0644); err != nil {
			return fmt.Errorf("write %s: %w", f.path, err)
		}
	}
	return nil
}

func pickBaseName(nameFlag string, inputs []string) string {
	if nameFlag != "" {
		return nameFlag
	}
	return strings.TrimSuffix(filepath.Base(inputs[0]), ".go")
}

func toTitle(s string) string {
	if s == "" {
		return ""
	}

	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
