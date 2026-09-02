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
	case f.Kind == parser.KindMap:
		return "map length for " + f.Name
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
	if pred(f) || (f.Key != nil && pred(*f.Key)) {
		return true
	}
	return f.Elem != nil && hasField(*f.Elem, pred)
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

func isMap(f parser.Field) bool {
	return f.Kind == parser.KindMap
}

func isStringKeyMap(f parser.Field) bool {
	return f.Kind == parser.KindMap && isString(*f.Key)
}

func schemaNeedsLengthGuards(messages []parser.Message) bool {
	return anyField(messages, func(f parser.Field) bool {
		return isString(f) || f.Kind == parser.KindSlice || f.Kind == parser.KindMap
	})
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
	case "gml":
		inner = fmt.Sprintf("(%s - (%g)) / (%g - (%g)) * %g", valueExpr, q.Min, q.Max, q.Min, q.MaxUint())
		return fmt.Sprintf("(floor(%s))", inner)
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
	case "lua", "gml":
		return fmt.Sprintf("(%s / %g) * (%g - (%g)) + (%g)", rawExpr, q.MaxUint(), q.Max, q.Min, q.Min)
	default:
		panic("unsupported language: " + lang)
	}
}

func needsBinaryImport(messages []parser.Message) bool {
	return anyField(messages, func(f parser.Field) bool {
		switch f.Kind {
		case parser.KindSlice, parser.KindMap:
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
