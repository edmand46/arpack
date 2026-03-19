package parser

// PrimitiveKind — конкретный примитивный тип.
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

// FieldKind — категория поля.
type FieldKind int

const (
	KindPrimitive  FieldKind = iota // float, int, uint, bool, string
	KindNested                      // ссылка на другой Message
	KindFixedArray                  // [N]T
	KindSlice                       // []T
)

// QuantInfo описывает квантизацию float → uint8/uint16.
type QuantInfo struct {
	Min  float64
	Max  float64
	Bits int // 8 или 16, default 16
}

// MaxUint — максимальное целое значение для данного числа бит.
func (q *QuantInfo) MaxUint() float64 {
	if q.Bits == 8 {
		return 255
	}
	return 65535
}

// WireBytes — размер на проводе в байтах.
func (q *QuantInfo) WireBytes() int {
	return q.Bits / 8
}

// Field — одно поле структуры-сообщения.
type Field struct {
	Name string
	Kind FieldKind

	// KindPrimitive
	Primitive PrimitiveKind
	NamedType string
	Quant     *QuantInfo // nil если нет квантизации

	// KindNested
	TypeName string

	// KindFixedArray / KindSlice
	Elem     *Field
	FixedLen int // >0 только для KindFixedArray
}

// WireSize — размер в байтах на проводе.
// Возвращает -1 для полей переменного размера.
func (f *Field) WireSize() int {
	switch f.Kind {
	case KindPrimitive:
		if f.Quant != nil {
			return f.Quant.WireBytes()
		}
		return primitiveWireSize(f.Primitive)
	case KindNested:
		return -1 // зависит от конкретного типа, узнаём через Message.MinWireSize
	case KindFixedArray:
		elemSize := f.Elem.WireSize()
		if elemSize == -1 {
			return -1
		}
		return f.FixedLen * elemSize
	case KindSlice:
		return -1 // uint16 len + переменная часть
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

// IsIntegralPrimitive — подходит ли тип как базовый для enum.
func IsIntegralPrimitive(k PrimitiveKind) bool {
	switch k {
	case KindInt8, KindInt16, KindInt32, KindInt64, KindUint8, KindUint16, KindUint32, KindUint64:
		return true
	}
	return false
}

// GoTypeName — имя типа в Go.
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
		return "[" + itoa(f.FixedLen) + "]" + f.Elem.GoTypeName()
	case KindSlice:
		return "[]" + f.Elem.GoTypeName()
	}
	return "unknown"
}

// CSharpTypeName — имя типа в C#.
func (f *Field) CSharpTypeName() string {
	switch f.Kind {
	case KindPrimitive:
		if f.NamedType != "" {
			return f.NamedType
		}
		return primitiveCSharpName(f.Primitive)
	case KindNested:
		return f.TypeName
	case KindFixedArray:
		return f.Elem.CSharpTypeName() + "[]"
	case KindSlice:
		return f.Elem.CSharpTypeName() + "[]"
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

// GoPrimitiveTypeName — базовый примитивный тип поля в Go.
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

// CSharpPrimitiveTypeName — базовый примитивный тип поля в C#.
func (f *Field) CSharpPrimitiveTypeName() string {
	return primitiveCSharpName(f.Primitive)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}

// Message — описание одной структуры-сообщения.
type Message struct {
	PackageName string
	Name        string
	Fields      []Field
}

// EnumValue — одно именованное значение enum.
type EnumValue struct {
	Name  string
	Value string
}

// Enum — enum-подобный тип на основе именованного примитива.
type Enum struct {
	Name      string
	Primitive PrimitiveKind
	Values    []EnumValue
}

// Schema — полная модель входного файла.
type Schema struct {
	PackageName string
	Messages    []Message
	Enums       []Enum
}

// MinWireSize — минимальный гарантированный размер в байтах.
// Для вложенных типов считается только если размер известен заранее.
// Строки и слайсы считаются как 2 байта (length prefix).
func (m *Message) MinWireSize() int {
	total := 0
	for _, f := range m.Fields {
		s := f.WireSize()
		if s == -1 {
			total += 2 // минимум: length prefix
		} else {
			total += s
		}
	}
	return total
}

// HasVariableFields — есть ли поля переменного размера.
func (m *Message) HasVariableFields() bool {
	for _, f := range m.Fields {
		if f.WireSize() == -1 {
			return true
		}
	}
	return false
}
