package generator

import "github.com/edmand46/arpack/parser"

const maxUint16Len = 65535

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
