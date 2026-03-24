//go:build !solution

package reverse

import (
	"strings"
	"unicode/utf8"
)

func Reverse(input string) string {
	sb := strings.Builder{}
	var totalLen int

	for _, char := range input {
		totalLen += utf8.RuneLen(char)
	}

	sb.Grow(totalLen)

	for i := len(input); i > 0; {
		var char, size = utf8.DecodeLastRuneInString(input[0:i])
		sb.WriteRune(char)
		i -= size
	}

	return sb.String()
}
