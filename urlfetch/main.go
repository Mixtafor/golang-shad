//go:build !solution

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func make_req(url string) {
	resp, err := http.Get(url)

	if err != nil {
		fmt.Println("error")
		os.Exit(1)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error")
		os.Exit(1)
	}

	fmt.Println(string(body))

}

func main() {
	urls := os.Args

	if len(urls) <= 1 {
		return
	}

	for _, url := range urls[1:] {
		make_req(url)
	}
}
