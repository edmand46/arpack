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
	case isString(f):
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

func hasField(f parser.Field, pred func(parser.Field) bool) bool {
	for p := &f; p != nil; p = p.Elem {
		if pred(*p) {
			return true
		}
	}
	return false
}

func anyField(messages []parser.Message, pred func(parser.Field) bool) bool {
	for _, msg := range messages {
		for _, f := range msg.Fields {
			if hasField(f, pred) {
				return true
			}
		}
	}
	return false
}

func isString(f parser.Field) bool {
	return f.Kind == parser.KindPrimitive && f.Primitive == parser.KindString
}

func schemaNeedsLengthGuards(messages []parser.Message) bool {
	return anyField(messages, func(f parser.Field) bool { return isString(f) || f.Kind == parser.KindSlice })
}

func schemaNeedsQuantRangeGuards(messages []parser.Message) bool {
	return anyField(messages, func(f parser.Field) bool { return f.Quant != nil })
}

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

func dequantizeExpr(lang, rawExpr string, q *parser.QuantInfo, primKind parser.PrimitiveKind) string {
	var inner string
	switch lang {
	case "go":
		inner = fmt.Sprintf("(float64(%s) / %g) * (%g - (%g)) + (%g)", rawExpr, q.MaxUint(), q.Max, q.Min, q.Min)
		if primKind == parser.KindFloat32 {
			return fmt.Sprintf("float32(%s)", inner)
		}
		return inner
	case "cs":
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
	return anyField(messages, func(f parser.Field) bool {
		switch f.Kind {
		case parser.KindSlice:
			return true
		case parser.KindPrimitive:
			if f.Quant != nil {
				return f.Quant.Bits == 16
			}
			return f.Primitive != parser.KindBool && f.Primitive != parser.KindInt8 && f.Primitive != parser.KindUint8
		}
		return false
	})
}
