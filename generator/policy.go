package generator

import (
	"fmt"

	"github.com/edmand46/arpack/parser"
)

func lengthContext(f parser.Field) string {
	switch {
	case f.Kind == parser.KindSlice:
		if f.Name != "" {
			return "slice length for " + f.Name
		}
		return "slice length"
	case f.Kind == parser.KindPrimitive && f.Primitive == parser.KindString:
		if f.Name != "" {
			return "string length for " + f.Name
		}
		return "string length"
	default:
		return "length"
	}
}

func quantContext(f parser.Field) string {
	if f.Name != "" {
		return f.Name
	}
	return "value"
}

func schemaNeedsLengthGuards(messages []parser.Message) bool {
	for _, msg := range messages {
		for _, f := range msg.Fields {
			if fieldNeedsLengthGuard(f) {
				return true
			}
		}
	}
	return false
}

func fieldNeedsLengthGuard(f parser.Field) bool {
	switch f.Kind {
	case parser.KindPrimitive:
		return f.Primitive == parser.KindString
	case parser.KindFixedArray, parser.KindSlice:
		if f.Kind == parser.KindSlice {
			return true
		}
		if f.Elem != nil {
			return fieldNeedsLengthGuard(*f.Elem)
		}
	}
	return false
}

func schemaNeedsQuantRangeGuards(messages []parser.Message) bool {
	for _, msg := range messages {
		for _, f := range msg.Fields {
			if fieldNeedsQuantRangeGuard(f) {
				return true
			}
		}
	}
	return false
}

func fieldNeedsQuantRangeGuard(f parser.Field) bool {
	switch f.Kind {
	case parser.KindPrimitive:
		return f.Quant != nil
	case parser.KindFixedArray, parser.KindSlice:
		if f.Elem != nil {
			return fieldNeedsQuantRangeGuard(*f.Elem)
		}
	}
	return false
}

// quantizeExpr returns the arithmetic expression string that converts a float64 value
// to its quantized integer, using float64 arithmetic and truncation toward zero.
// lang: "go", "cs", "ts", or "lua".
// valueExpr: source-level expression evaluating to the float64 value (e.g. "m.PosX").
// The caller is responsible for the surrounding buffer write and variable assignment.
func quantizeExpr(lang, valueExpr string, q *parser.QuantInfo, bits int) string {
	var inner string
	switch lang {
	case "go":
		inner = fmt.Sprintf("(float64(%s) - (%g)) / (%g - (%g)) * %g", valueExpr, q.Min, q.Max, q.Min, q.MaxUint())
		if bits == 8 {
			return fmt.Sprintf("uint8(%s)", inner)
		}
		return fmt.Sprintf("uint16(%s)", inner)
	case "cs":
		inner = fmt.Sprintf("(((double)(%s) - (%g)) / (%g - (%g)) * %g)", valueExpr, q.Min, q.Max, q.Min, q.MaxUint())
		if bits == 8 {
			return fmt.Sprintf("(byte)(%s)", inner)
		}
		return fmt.Sprintf("(ushort)(%s)", inner)
	case "ts":
		inner = fmt.Sprintf("(%s - (%g)) / (%g - (%g)) * %g", valueExpr, q.Min, q.Max, q.Min, q.MaxUint())
		return fmt.Sprintf("Math.trunc(%s)", inner)
	case "lua":
		inner = fmt.Sprintf("(%s - (%g)) / (%g - (%g)) * %g", valueExpr, q.Min, q.Max, q.Min, q.MaxUint())
		return fmt.Sprintf("(math.modf(%s))", inner)
	default:
		panic("unsupported language: " + lang)
	}
}

// dequantizeExpr returns the arithmetic expression string that reconstructs a float64
// from a quantized wire value, using float64 arithmetic.
// lang: "go", "cs", "ts", or "lua".
// rawExpr: source-level expression for the raw wire integer (e.g. "data[offset]").
// For float32 targets, the caller wraps the result in a float32 cast after dequant.
func dequantizeExpr(lang, rawExpr string, q *parser.QuantInfo, primKind parser.PrimitiveKind) string {
	var inner string
	switch lang {
	case "go":
		// Cast rawExpr to float64 first to avoid integer division overflow
		inner = fmt.Sprintf("(float64(%s) / %g) * (%g - (%g)) + (%g)", rawExpr, q.MaxUint(), q.Max, q.Min, q.Min)
		if primKind == parser.KindFloat32 {
			return fmt.Sprintf("float32(%s)", inner)
		}
		return inner
	case "cs":
		// Cast rawExpr to double first to avoid integer division
		inner = fmt.Sprintf("((double)(%s) / %g) * (%g - (%g)) + (%g)", rawExpr, q.MaxUint(), q.Max, q.Min, q.Min)
		if primKind == parser.KindFloat32 {
			return fmt.Sprintf("(float)(%s)", inner)
		}
		return fmt.Sprintf("(double)(%s)", inner)
	case "ts":
		return fmt.Sprintf("(%s / %g) * (%g - (%g)) + (%g)", rawExpr, q.MaxUint(), q.Max, q.Min, q.Min)
	case "lua":
		return fmt.Sprintf("(%s / %g) * (%g - (%g)) + (%g)", rawExpr, q.MaxUint(), q.Max, q.Min, q.Min)
	default:
		panic("unsupported language: " + lang)
	}
}

func needsBinaryImport(messages []parser.Message) bool {
	for _, msg := range messages {
		for _, f := range msg.Fields {
			if fieldNeedsBinary(f) {
				return true
			}
		}
	}
	return false
}

func fieldNeedsBinary(f parser.Field) bool {
	switch f.Kind {
	case parser.KindPrimitive:
		// bool, int8, uint8, and 8-bit quant use append(); everything else needs binary
		if f.Quant != nil {
			return f.Quant.Bits == 16
		}
		return f.Primitive != parser.KindBool &&
			f.Primitive != parser.KindInt8 &&
			f.Primitive != parser.KindUint8
	case parser.KindSlice:
		return true // length prefix uses binary.LittleEndian.AppendUint16
	case parser.KindFixedArray:
		if f.Elem != nil {
			return fieldNeedsBinary(*f.Elem)
		}
	}
	return false
}
