//go:build !solution

package varfmt

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

func Sprintf(format string, args ...any) string {
	sb := strings.Builder{}

	refsToArgsCnt := 0

	for i := 0; i < len(format); {
		if format[i] != '{' {
			sb.WriteByte(format[i])
			i++
			continue
		}

		i++
		endInd := i
		for ; endInd < len(format) && unicode.IsDigit(rune(format[endInd])); endInd++ {
		}

		if i == endInd {
			fmt.Fprint(&sb, args[refsToArgsCnt])
		} else {
			objInd, err := strconv.Atoi(format[i:endInd])
			if err != nil {
				fmt.Println("format error")
				return ""
			}
			fmt.Fprint(&sb, args[objInd])
		}
		refsToArgsCnt++
		i = endInd + 1
	}

	return sb.String()
}
