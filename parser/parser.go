package parser

import (
	"fmt"
	"go/ast"
	"go/importer"
	goparser "go/parser"
	"go/token"
	"go/types"
	"reflect"
	"strconv"
	"strings"
)

func ParseFile(path string) ([]Message, error) {
	schema, err := ParseSchemaFile(path)
	if err != nil {
		return nil, err
	}
	return schema.Messages, nil
}

func ParseSource(src string) ([]Message, error) {
	schema, err := ParseSchemaSource(src)
	if err != nil {
		return nil, err
	}
	return schema.Messages, nil
}

func ParseSchemaFile(path string) (Schema, error) {
	fset := token.NewFileSet()

	f, err := goparser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return Schema{}, fmt.Errorf("parse %s: %w", path, err)
	}

	return parseASTFile(fset, f)
}

func ParseSchemaSource(src string) (Schema, error) {
	fset := token.NewFileSet()

	f, err := goparser.ParseFile(fset, "source.go", src, 0)
	if err != nil {
		return Schema{}, fmt.Errorf("parse source: %w", err)
	}

	return parseASTFile(fset, f)
}

func parseASTFile(fset *token.FileSet, f *ast.File) (Schema, error) {
	pkgName := f.Name.Name

	knownStructs := map[string]bool{}
	namedPrimitives := map[string]PrimitiveKind{}
	var enumOrder []string

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			switch t := typeSpec.Type.(type) {
			case *ast.StructType:
				knownStructs[typeSpec.Name.Name] = true
			case *ast.Ident:
				primKind, isPrimitive := goPrimitiveKind(t.Name)
				if !isPrimitive {
					continue
				}
				namedPrimitives[typeSpec.Name.Name] = primKind
				if IsIntegralPrimitive(primKind) {
					enumOrder = append(enumOrder, typeSpec.Name.Name)
				}
			}
		}
	}

	info, err := typeCheckFile(fset, f)
	if err != nil {
		return Schema{}, err
	}

	schema := Schema{PackageName: pkgName}
	enumIndex := make(map[string]int, len(enumOrder))
	for _, name := range enumOrder {
		enumIndex[name] = len(schema.Enums)
		schema.Enums = append(schema.Enums, Enum{
			Name:      name,
			Primitive: namedPrimitives[name],
		})
	}

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		switch genDecl.Tok {
		case token.TYPE:
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}

				msg, err := parseStruct(pkgName, typeSpec.Name.Name, structType, knownStructs, namedPrimitives)
				if err != nil {
					return Schema{}, fmt.Errorf("struct %s: %w", typeSpec.Name.Name, err)
				}

				schema.Messages = append(schema.Messages, msg)
			}
		case token.CONST:
			if err := parseConstDecls(genDecl, info, enumIndex, &schema); err != nil {
				return Schema{}, err
			}
		}
	}

	return schema, nil
}

func typeCheckFile(fset *token.FileSet, f *ast.File) (*types.Info, error) {
	info := &types.Info{
		Defs: make(map[*ast.Ident]types.Object),
	}

	cfg := &types.Config{
		Importer: importer.Default(),
	}

	if _, err := cfg.Check(f.Name.Name, fset, []*ast.File{f}, info); err != nil {
		return nil, fmt.Errorf("typecheck %s: %w", f.Name.Name, err)
	}

	return info, nil
}

func parseConstDecls(genDecl *ast.GenDecl, info *types.Info, enumIndex map[string]int, schema *Schema) error {
	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		for _, name := range valueSpec.Names {
			obj, ok := info.Defs[name].(*types.Const)
			if !ok {
				continue
			}

			named, ok := obj.Type().(*types.Named)
			if !ok {
				continue
			}

			idx, ok := enumIndex[named.Obj().Name()]
			if !ok {
				continue
			}

			schema.Enums[idx].Values = append(schema.Enums[idx].Values, EnumValue{
				Name:  name.Name,
				Value: obj.Val().String(),
			})
		}
	}

	return nil
}

func parseStruct(
	pkg string,
	name string,
	st *ast.StructType,
	knownStructs map[string]bool,
	namedPrimitives map[string]PrimitiveKind,
) (Message, error) {
	msg := Message{PackageName: pkg, Name: name}

	for _, astField := range st.Fields.List {
		if len(astField.Names) == 0 {
			continue
		}

		var rawTag string
		if astField.Tag != nil {
			tag := reflect.StructTag(strings.Trim(astField.Tag.Value, "`"))
			rawTag = tag.Get("pack")
		}

		for _, fieldName := range astField.Names {
			field, err := parseFieldType(fieldName.Name, astField.Type, rawTag, knownStructs, namedPrimitives)
			if err != nil {
				return Message{}, fmt.Errorf("field %s: %w", fieldName.Name, err)
			}

			msg.Fields = append(msg.Fields, field)
		}
	}
	return msg, nil
}

func parseFieldType(
	name string,
	expr ast.Expr,
	rawTag string,
	knownStructs map[string]bool,
	namedPrimitives map[string]PrimitiveKind,
) (Field, error) {
	switch t := expr.(type) {
	case *ast.Ident:
		return parsePrimitiveOrNested(name, t.Name, rawTag, knownStructs, namedPrimitives)

	case *ast.ArrayType:
		if t.Len == nil {
			elem, err := parseFieldType("", t.Elt, rawTag, knownStructs, namedPrimitives)
			if err != nil {
				return Field{}, fmt.Errorf("slice element: %w", err)
			}

			return Field{
				Name: name,
				Kind: KindSlice,
				Elem: &elem,
			}, nil
		}

		n, err := parseArrayLen(t.Len)
		if err != nil {
			return Field{}, fmt.Errorf("array length: %w", err)
		}

		elem, err := parseFieldType("", t.Elt, rawTag, knownStructs, namedPrimitives)
		if err != nil {
			return Field{}, fmt.Errorf("array element: %w", err)
		}

		return Field{
			Name:     name,
			Kind:     KindFixedArray,
			Elem:     &elem,
			FixedLen: n,
		}, nil

	case *ast.StarExpr:
		return Field{}, fmt.Errorf("pointer types not supported")

	case *ast.SelectorExpr:
		return Field{}, fmt.Errorf("external package types not supported (use only types from the same file)")
	}

	return Field{}, fmt.Errorf("unsupported type expression %T", expr)
}

func parsePrimitiveOrNested(
	name string,
	typeName string,
	rawTag string,
	knownStructs map[string]bool,
	namedPrimitives map[string]PrimitiveKind,
) (Field, error) {
	primKind, isPrimitive := goPrimitiveKind(typeName)
	if !isPrimitive {
		if namedPrimitive, ok := namedPrimitives[typeName]; ok {
			return buildPrimitiveField(name, typeName, namedPrimitive, rawTag)
		}

		if !knownStructs[typeName] {
			return Field{}, fmt.Errorf("unknown type %q (not a primitive and not defined in the same file)", typeName)
		}
		return Field{
			Name:     name,
			Kind:     KindNested,
			TypeName: typeName,
		}, nil
	}

	return buildPrimitiveField(name, "", primKind, rawTag)
}

func buildPrimitiveField(name, namedType string, primKind PrimitiveKind, rawTag string) (Field, error) {
	field := Field{
		Name:      name,
		Kind:      KindPrimitive,
		Primitive: primKind,
		NamedType: namedType,
	}

	if rawTag != "" {
		if primKind != KindFloat32 && primKind != KindFloat64 {
			typeLabel := field.GoTypeName()
			return Field{}, fmt.Errorf("arpack tag can only be applied to float32/float64, got %s", typeLabel)
		}
		quant, err := parseQuantTag(rawTag)
		if err != nil {
			return Field{}, fmt.Errorf("arpack tag: %w", err)
		}
		field.Quant = quant
	}

	return field, nil
}

func parseQuantTag(tag string) (*QuantInfo, error) {
	info := &QuantInfo{Bits: 16}
	parts := strings.Split(tag, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid tag part %q (expected key=value)", p)
		}

		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])

		switch key {
		case "min":
			v, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return nil, fmt.Errorf("min: %w", err)
			}

			info.Min = v
		case "max":
			v, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return nil, fmt.Errorf("max: %w", err)
			}

			info.Max = v
		case "bits":
			v, err := strconv.Atoi(val)
			if err != nil || (v != 8 && v != 16) {
				return nil, fmt.Errorf("bits must be 8 or 16, got %q", val)
			}

			info.Bits = v
		default:
			return nil, fmt.Errorf("unknown tag key %q", key)
		}
	}

	if info.Max <= info.Min {
		return nil, fmt.Errorf("max (%.6g) must be greater than min (%.6g)", info.Max, info.Min)
	}

	return info, nil
}

func parseArrayLen(expr ast.Expr) (int, error) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok {
		return 0, fmt.Errorf("array length must be a literal integer constant")
	}

	n, err := strconv.Atoi(lit.Value)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("array length must be a positive integer, got %q", lit.Value)
	}

	return n, nil
}

func goPrimitiveKind(name string) (PrimitiveKind, bool) {
	switch name {
	case "float32":
		return KindFloat32, true
	case "float64":
		return KindFloat64, true
	case "int8":
		return KindInt8, true
	case "int16":
		return KindInt16, true
	case "int32", "int":
		return KindInt32, true
	case "int64":
		return KindInt64, true
	case "uint8", "byte":
		return KindUint8, true
	case "uint16":
		return KindUint16, true
	case "uint32", "uint":
		return KindUint32, true
	case "uint64":
		return KindUint64, true
	case "bool":
		return KindBool, true
	case "string":
		return KindString, true
	}
	return 0, false
}
