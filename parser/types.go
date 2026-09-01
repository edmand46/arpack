package parser

import "strconv"

type PrimitiveKind int

const (
	KindFloat32 PrimitiveKind = iota
	KindFloat64
	KindInt8
	KindInt16
	KindInt32
	KindInt64
	KindUint8
	KindUint16
	KindUint32
	KindUint64
	KindBool
	KindString
)

type FieldKind int

const (
	KindPrimitive FieldKind = iota
	KindNested
	KindFixedArray
	KindSlice
)

type QuantInfo struct {
	Min  float64
	Max  float64
	Bits int
}

func (q *QuantInfo) MaxUint() float64 {
	if q.Bits == 8 {
		return 255
	}
	return 65535
}

func (q *QuantInfo) WireBytes() int {
	return q.Bits / 8
}

type Field struct {
	Name string
	Kind FieldKind

	Primitive PrimitiveKind
	NamedType string
	Quant     *QuantInfo

	TypeName string

	Elem     *Field
	FixedLen int
}

func (f *Field) WireSize() int {
	switch f.Kind {
	case KindPrimitive:
		if f.Quant != nil {
			return f.Quant.WireBytes()
		}
		return primitiveWireSize(f.Primitive)
	case KindNested:
		return -1
	case KindFixedArray:
		elemSize := f.Elem.WireSize()
		if elemSize == -1 {
			return -1
		}
		return f.FixedLen * elemSize
	case KindSlice:
		return -1
	}
	return 0
}

func primitiveWireSize(k PrimitiveKind) int {
	switch k {
	case KindFloat32, KindInt32, KindUint32:
		return 4
	case KindFloat64, KindInt64, KindUint64:
		return 8
	case KindInt16, KindUint16:
		return 2
	case KindInt8, KindUint8, KindBool:
		return 1
	case KindString:
		return -1
	}
	return 0
}

func IsIntegralPrimitive(k PrimitiveKind) bool {
	switch k {
	case KindInt8, KindInt16, KindInt32, KindInt64, KindUint8, KindUint16, KindUint32, KindUint64:
		return true
	}
	return false
}

func (f *Field) GoTypeName() string {
	switch f.Kind {
	case KindPrimitive:
		if f.NamedType != "" {
			return f.NamedType
		}
		return primitiveGoName(f.Primitive)
	case KindNested:
		return f.TypeName
	case KindFixedArray:
		return "[" + strconv.Itoa(f.FixedLen) + "]" + f.Elem.GoTypeName()
	case KindSlice:
		return "[]" + f.Elem.GoTypeName()
	}
	return "unknown"
}

func primitiveGoName(k PrimitiveKind) string {
	switch k {
	case KindFloat32:
		return "float32"
	case KindFloat64:
		return "float64"
	case KindInt8:
		return "int8"
	case KindInt16:
		return "int16"
	case KindInt32:
		return "int32"
	case KindInt64:
		return "int64"
	case KindUint8:
		return "uint8"
	case KindUint16:
		return "uint16"
	case KindUint32:
		return "uint32"
	case KindUint64:
		return "uint64"
	case KindBool:
		return "bool"
	case KindString:
		return "string"
	}
	return "unknown"
}

func (f *Field) GoPrimitiveTypeName() string {
	return primitiveGoName(f.Primitive)
}

func primitiveCSharpName(k PrimitiveKind) string {
	switch k {
	case KindFloat32:
		return "float"
	case KindFloat64:
		return "double"
	case KindInt8:
		return "sbyte"
	case KindInt16:
		return "short"
	case KindInt32:
		return "int"
	case KindInt64:
		return "long"
	case KindUint8:
		return "byte"
	case KindUint16:
		return "ushort"
	case KindUint32:
		return "uint"
	case KindUint64:
		return "ulong"
	case KindBool:
		return "bool"
	case KindString:
		return "string"
	}
	return "unknown"
}

func (f *Field) CSharpPrimitiveTypeName() string {
	return primitiveCSharpName(f.Primitive)
}

type Message struct {
	Name   string
	Fields []Field
}

type EnumValue struct {
	Name  string
	Value string
}

type Enum struct {
	Name      string
	Primitive PrimitiveKind
	Values    []EnumValue
}

type Schema struct {
	PackageName string
	Messages    []Message
	Enums       []Enum
}

func (m *Message) MinWireSize() int {
	total := 0
	i := 0
	for i < len(m.Fields) {
		f := m.Fields[i]
		if f.Kind == KindPrimitive && f.Primitive == KindBool {
			run := 0
			for i < len(m.Fields) && m.Fields[i].Kind == KindPrimitive && m.Fields[i].Primitive == KindBool {
				run++
				i++
			}
			total += (run + 7) / 8
			continue
		}
		s := f.WireSize()
		if s == -1 {
			total += 2
		} else {
			total += s
		}
		i++
	}
	return total
}

func (m *Message) HasVariableFields() bool {
	for _, f := range m.Fields {
		if f.WireSize() == -1 {
			return true
		}
	}
	return false
}
