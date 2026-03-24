package main

import (
	"fmt"
	"os"
)

func main() {
	var s, sep string
	for i := range len(os.Args) - 1 {
		s += sep + os.Args[i+1]
		sep = " "
	}
	fmt.Println(s)
}
