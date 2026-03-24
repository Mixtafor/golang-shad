package main

import (
	"fmt"
	"net/http"
	"sync"
)

func main() {
	urls := []string{"https://golang.org/doc", "https://golang.org/pkg", "https://golang.org/help"}

	var wg sync.WaitGroup

	for _, url := range urls {
		url := url
		wg.Go(func() {
			var client http.Client
			resp, err := client.Get(url)
			if err != nil {
				fmt.Printf("%s: %s\n", url, err)
				return
			}
			fmt.Printf("%s - %d\n", url, resp.StatusCode)
		})
	}

	wg.Wait()
}
