package common

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

var s string

func init() {
	s = strings.Repeat("go中国人golang", 1000)
}

func TestUtfToUnicode(t *testing.T) {
	buf, err := UtfToUnicode([]byte(s))
	if err != nil {
		t.Fatalf("%v", err)
	}

	t.Logf(buf.String())
}

func TestUtfToUnicode2(t *testing.T) {
	buf, err := utfToUnicode2([]byte(s))
	if err != nil {
		t.Fatalf("%v", err)
	}

	t.Logf(buf.String())
}

func utfToUnicode2(b []byte) (reader *bytes.Buffer, err error) {
	reader = new(bytes.Buffer)
	for _, v := range s {

		if v < 128 {
			_, err := reader.WriteRune(rune(v))
			if err != nil {
				return nil, err
			}
		} else {
			_, err := reader.WriteString(fmt.Sprintf("%s%04x", `\u`, rune(v)))
			if err != nil {
				return nil, err
			}
		}
	}

	return
}

func BenchmarkUtfToUnicode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		UtfToUnicode([]byte(s))
	}
}

func BenchmarkUtfToUnicode2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		utfToUnicode2([]byte(s))
	}
}
