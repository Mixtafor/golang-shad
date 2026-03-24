//go:build !solution

package speller

import (
	"fmt"
	"io"
	"strings"
)

var numbers2spell = map[int64]string{
	1000000000: "billion",
	1000000:    "million",
	1000:       "thousand",
	100:        "hundred",
	90:         "ninety",
	80:         "eighty",
	70:         "seventy",
	60:         "sixty",
	50:         "fifty",
	40:         "forty",
	30:         "thirty",
	20:         "twenty",
	19:         "nineteen",
	18:         "eighteen",
	17:         "seventeen",
	16:         "sixteen",
	15:         "fifteen",
	14:         "fourteen",
	13:         "thirteen",
	12:         "twelve",
	11:         "eleven",
	10:         "ten",
	9:          "nine",
	8:          "eight",
	7:          "seven",
	6:          "six",
	5:          "five",
	4:          "four",
	3:          "three",
	2:          "two",
	1:          "one",
}

var numbers2spellScales = []int64{
	1000000000,
	1000000, 1000, 100,
	90, 80, 70, 60, 50, 40, 30, 20,
	19, 18, 17, 16, 15, 14, 13, 12, 11, 10,
	9, 8, 7, 6, 5, 4, 3, 2, 1,
}

func Spell(n int64) string {
	if n == 0 {
		return "zero"
	}
	sb := strings.Builder{}

	SpellHelper(&sb, n)
	str := sb.String()
	if len(str) > 0 && str[len(str)-1] == ' ' {
		str = str[:len(str)-1]
	}
	return str
}

func SmallNumsSpell(w io.Writer, n int64) (is_ok bool) {
	switch {
	case 0 <= n && n <= 20:
		w.Write([]byte(numbers2spell[n]))
		return true
	case 21 <= n && n <= 99 && n%10 == 0:
		fmt.Fprintf(w, "%s", numbers2spell[n])
		return true
	case 21 <= n && n <= 99 && n%10 != 0:
		fmt.Fprintf(w, "%s-%s", numbers2spell[n-n%10], numbers2spell[n%10])
		return true
	}

	return false
}

func SpellHelper(w io.Writer, n int64) {
	if SmallNumsSpell(w, n) {
		return
	}

	if n < 0 {
		n = -n
		w.Write([]byte("minus "))
	}

	for _, num := range numbers2spellScales {
		if n < 1 || SmallNumsSpell(w, n) {
			break
		}

		if n/num != 0 {
			SpellHelper(w, n/num)
			w.Write([]byte(" "))
			w.Write([]byte(numbers2spell[num]))
			w.Write([]byte(" "))

			n -= (n / num) * num
		}
	}

}
