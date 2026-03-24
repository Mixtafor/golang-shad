//go:build !solution

package main

import (
	"bufio"
	"fmt"
	"os"
)

var mapp map[string]int

func main() {
	mapp = make(map[string]int)
	for i := 1; i < len(os.Args); i++ {
		f, err := os.Open(os.Args[i])
		scanner := bufio.NewScanner(f)
		if err != nil {
			fmt.Println("Ошибка :", err)
			return
		}

		defer f.Close()

		for scanner.Scan() {
			line := scanner.Text()
			mapp[line]++
		}

	}

	for key, val := range mapp {
		if val >= 2 {
			fmt.Printf("%d\t%v\n", val, key)
		}
	}

}
