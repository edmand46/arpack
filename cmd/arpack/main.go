package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/edmand46/arpack/generator"
	"github.com/edmand46/arpack/parser"
)

func main() {
	in := flag.String("in", "", "input Go file with struct definitions")
	outGo := flag.String("out-go", "", "output directory for generated Go code")
	outCS := flag.String("out-cs", "", "output directory for generated C# code")
	outTS := flag.String("out-ts", "", "output directory for generated TypeScript code")
	outLua := flag.String("out-lua", "", "output directory for generated Lua code")
	outC := flag.String("out-c", "", "output directory for generated C code")
	namespace := flag.String("cs-namespace", "Arpack.Messages", "C# namespace")
	flag.Parse()

	if *in == "" {
		log.Fatal("arpack: -in is required")
	}

	if *outGo == "" && *outCS == "" && *outTS == "" && *outLua == "" && *outC == "" {
		log.Fatal("arpack: at least one of -out-go, -out-cs, -out-ts, -out-lua, or -out-c is required")
	}

	schema, err := parser.ParseSchemaFile(*in)
	if err != nil {
		log.Fatalf("arpack: parse error: %v", err)
	}
	msgs := schema.Messages
	if len(msgs) == 0 {
		log.Fatalf("arpack: no structs found in %s", *in)
	}

	baseName := strings.TrimSuffix(filepath.Base(*in), ".go")

	if *outGo != "" {
		pkgName := filepath.Base(*outGo)
		if pkgName == "." || pkgName == "" {
			pkgName = msgs[0].PackageName
		}
		// Replace hyphens with underscores for valid Go package names
		pkgName = strings.ReplaceAll(pkgName, "-", "_")

		src, err := generator.GenerateGoSchema(schema, pkgName)
		if err != nil {
			log.Fatalf("arpack: Go generation error: %v", err)
		}

		outPath := filepath.Join(*outGo, baseName+"_gen.go")
		if err := os.MkdirAll(*outGo, 0755); err != nil {
			log.Fatalf("arpack: mkdir %s: %v", *outGo, err)
		}
		if err := os.WriteFile(outPath, src, 0644); err != nil {
			log.Fatalf("arpack: write %s: %v", outPath, err)
		}

		fmt.Printf("arpack: wrote %s\n", outPath)
	}

	if *outCS != "" {
		src, err := generator.GenerateCSharpSchema(schema, *namespace)
		if err != nil {
			log.Fatalf("arpack: C# generation error: %v", err)
		}

		outPath := filepath.Join(*outCS, toTitle(baseName)+".gen.cs")
		if err := os.MkdirAll(*outCS, 0755); err != nil {
			log.Fatalf("arpack: mkdir %s: %v", *outCS, err)
		}
		if err := os.WriteFile(outPath, src, 0644); err != nil {
			log.Fatalf("arpack: write %s: %v", outPath, err)
		}

		fmt.Printf("arpack: wrote %s\n", outPath)
	}

	if *outTS != "" {
		src, err := generator.GenerateTypeScriptSchema(schema, "Arpack.Messages")
		if err != nil {
			log.Fatalf("arpack: TypeScript generation error: %v", err)
		}

		outPath := filepath.Join(*outTS, toTitle(baseName)+".gen.ts")
		if err := os.MkdirAll(*outTS, 0755); err != nil {
			log.Fatalf("arpack: mkdir %s: %v", *outTS, err)
		}
		if err := os.WriteFile(outPath, src, 0644); err != nil {
			log.Fatalf("arpack: write %s: %v", outPath, err)
		}

		fmt.Printf("arpack: wrote %s\n", outPath)
	}

	if *outLua != "" {
		src, err := generator.GenerateLuaSchema(schema, baseName)
		if err != nil {
			log.Fatalf("arpack: Lua generation error: %v", err)
		}

		// Use snake_case filename for Lua require() compatibility
		outPath := filepath.Join(*outLua, toSnakeCase(baseName)+"_gen.lua")
		if err := os.MkdirAll(*outLua, 0755); err != nil {
			log.Fatalf("arpack: mkdir %s: %v", *outLua, err)
		}
		if err := os.WriteFile(outPath, src, 0644); err != nil {
			log.Fatalf("arpack: write %s: %v", outPath, err)
		}

		fmt.Printf("arpack: wrote %s\n", outPath)
	}

	if *outC != "" {
		snakeBase := toSnakeCase(baseName)
		headerSrc, sourceSrc, err := generator.GenerateCSchema(schema, snakeBase)
		if err != nil {
			log.Fatalf("arpack: C generation error: %v", err)
		}

		headerPath := filepath.Join(*outC, snakeBase+".gen.h")
		sourcePath := filepath.Join(*outC, snakeBase+".gen.c")

		if err := os.MkdirAll(*outC, 0755); err != nil {
			log.Fatalf("arpack: mkdir %s: %v", *outC, err)
		}

		if err := os.WriteFile(headerPath, headerSrc, 0644); err != nil {
			log.Fatalf("arpack: write %s: %v", headerPath, err)
		}
		if err := os.WriteFile(sourcePath, sourceSrc, 0644); err != nil {
			log.Fatalf("arpack: write %s: %v", sourcePath, err)
		}

		fmt.Printf("arpack: wrote %s\n", headerPath)
		fmt.Printf("arpack: wrote %s\n", sourcePath)
	}
}

func toTitle(s string) string {
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
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
