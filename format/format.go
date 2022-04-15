package format

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

func BytesFormat(data []byte) string {
	dataLen := len(data)
	parts := []string{}
	for i := 0; i < dataLen; i++ {
		if i != 0 && i%4 == 0 {
			parts = append(parts, "")
		}
		parts = append(parts, "%02x")
	}

	return fmt.Sprintf(strings.Join(parts, " "), bytesToAny(data)...)
}

func bytesToAny(input []byte) []any {
	output := make([]any, len(input))
	for i, b := range input {
		output[i] = b
	}

	return output
}

var ascii = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0x0030, 0x0039, 1},
		{0x0041, 0x005a, 1},
		{0x0061, 0x007a, 1},
	},
	LatinOffset: 3,
}

func BytesToAscii(data []byte) string {
	buf := bytes.NewBuffer(nil)
	dataLen := len(data)
	offset := 0
	for {
		if offset >= dataLen {
			break
		}
		r, size := utf8.DecodeRune(data[offset:])
		if unicode.Is(ascii, r) {
			buf.WriteRune(r)
		} else {
			buf.WriteRune('_')
		}
		offset += size
	}

	return buf.String()
}
