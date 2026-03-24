package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
)

func main() {
	urls := []string{"https://golang.org/doc", "https://golang.org/pkg", "https://golang.org/help"}

	client := &http.Client{Transport: &http.Transport{MaxConnsPerHost: 100}}

	var wg sync.WaitGroup
	for _, url := range urls {
		url := url
		wg.Go(func() {
			resp, err := client.Get(url)
			if err != nil {
				fmt.Printf("%s: %s\n", url, err)
				return
			}
			defer resp.Body.Close()
			_, _ = io.Copy(io.Discard, resp.Body)
			fmt.Printf("%s - %d\n", url, resp.StatusCode)
		})
	}

	wg.Wait()
}
