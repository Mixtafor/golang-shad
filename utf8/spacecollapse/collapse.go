//go:build !solution

package spacecollapse

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func CollapseSpaces(input string) string {
	ansLen := 0
	sb := strings.Builder{}

	var prev rune
	for _, char := range input {
		if unicode.IsSpace(char) && unicode.IsSpace(prev) {
			continue
		}

		if unicode.IsSpace(char) {
			ansLen += utf8.RuneLen(' ')
		} else {
			ansLen += utf8.RuneLen(char)
		}

		prev = char
	}

	sb.Grow(ansLen)

	prev = 0
	for _, char := range input {
		if unicode.IsSpace(char) && unicode.IsSpace(prev) {
			continue
		}

		if unicode.IsSpace(char) {
			sb.WriteRune(' ')
		} else {
			sb.WriteRune(char)
		}

		prev = char
	}

	return sb.String()
}
