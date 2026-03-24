//go:build !solution

package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"
)

func make_req(url string, wg *sync.WaitGroup) {
	defer wg.Done()
	resp, err := http.Get(url)

	if err != nil {
		fmt.Println("error")
	} else {
		defer resp.Body.Close()
	}

	fmt.Println(url, "done")

}

func main() {
	urls := os.Args

	if len(urls) <= 1 {
		return
	}

	var wg sync.WaitGroup

	for _, url := range urls[1:] {
		wg.Add(1)
		go make_req(url, &wg)
	}

	wg.Wait()
}
