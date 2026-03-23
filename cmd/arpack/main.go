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
	namespace := flag.String("cs-namespace", "Arpack.Messages", "C# namespace")
	flag.Parse()

	if *in == "" {
		log.Fatal("arpack: -in is required")
	}

	if *outGo == "" && *outCS == "" && *outTS == "" {
		log.Fatal("arpack: at least one of -out-go, -out-cs, or -out-ts is required")
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
}

func toTitle(s string) string {
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}
