package generator

import "github.com/edmand46/arpack/parser"

type segment struct {
	bools  []parser.Field
	single *parser.Field
}

func isBoolField(f parser.Field) bool {
	return f.Kind == parser.KindPrimitive && f.Primitive == parser.KindBool
}

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
			total += 1
		}
	}
	return total
}
