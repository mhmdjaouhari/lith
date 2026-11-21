// Package format writes numbers and words for people to read, the same in
// every demo: 12,345 with a comma between thousands, and a word's singular
// or plural. lith-ui's grouped writes numbers the same way in the browser.
package format

import "strconv"

// Integer is a signed integer type that Grouped writes.
type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

// Grouped writes n with a comma between thousands: 12,345, -1,000.
func Grouped[N Integer](n N) string { return string(AppendGrouped(nil, n)) }

// AppendGrouped appends n to b as Grouped writes it, for a frame written
// by hand into one buffer.
func AppendGrouped[N Integer](b []byte, n N) []byte {
	v := int64(n)
	var digits [20]byte
	s := strconv.AppendInt(digits[:0], v, 10)
	if v < 0 {
		b = append(b, '-')
		s = s[1:]
	}
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b = append(b, ',')
		}
		b = append(b, c)
	}
	return b
}

// Plural is one when n is 1, else many.
func Plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
