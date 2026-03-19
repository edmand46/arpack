package generator

import "edmand46/arpack/parser"

// segment — либо группа bool (1–8 полей → 1 байт), либо одиночное поле.
type segment struct {
	bools  []parser.Field // non-empty: bool-группа
	single *parser.Field  // non-nil: любое не-bool поле
}

// isBoolField возвращает true если поле — нативный bool (не массив, не слайс).
func isBoolField(f parser.Field) bool {
	return f.Kind == parser.KindPrimitive && f.Primitive == parser.KindBool
}

// segmentFields разбивает поля структуры на сегменты.
// Последовательные bool-поля группируются по 8 в один сегмент.
func segmentFields(fields []parser.Field) []segment {
	var segs []segment
	i := 0
	for i < len(fields) {
		if !isBoolField(fields[i]) {
			f := fields[i]
			segs = append(segs, segment{single: &f})
			i++
			continue
		}
		// Собираем последовательные bool-поля группами по 8
		for i < len(fields) && isBoolField(fields[i]) {
			var group []parser.Field
			for i < len(fields) && isBoolField(fields[i]) && len(group) < 8 {
				group = append(group, fields[i])
				i++
			}
			segs = append(segs, segment{bools: group})
		}
	}
	return segs
}

// packedMinWireSize вычисляет минимальный размер буфера с учётом упаковки bool.
func packedMinWireSize(fields []parser.Field) int {
	total := 0
	for _, seg := range segmentFields(fields) {
		if seg.single != nil {
			s := seg.single.WireSize()
			if s == -1 {
				total += 2
			} else {
				total += s
			}
		} else {
			// Группа bool → 1 байт
			total += 1
		}
	}
	return total
}
