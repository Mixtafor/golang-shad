//go:build !solution

package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"net/http"
	"sync"
)

type postQuery struct {
	Url string `json:"url"`
}

func main() {
	lenMaxKey := 30
	maxInt := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(lenMaxKey)), nil)

	port := flag.Int("port", 8080, "port to listen on")
	flag.Parse()
	mu := sync.RWMutex{}
	keyToUrl := make(map[string]string)
	urlToKey := make(map[string]string)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /shorten", func(w http.ResponseWriter, r *http.Request) {
		var q postQuery
		err := json.NewDecoder(r.Body).Decode(&q)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		mu.RLock()
		key, ok := urlToKey[q.Url]
		mu.RUnlock()
		if !ok {
			keyGen, err := rand.Int(rand.Reader, maxInt)

			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			key = keyGen.String()

			// if expired
			mu.Lock()
			k, ok := urlToKey[q.Url]
			if !ok {
				keyToUrl[key] = q.Url
				urlToKey[q.Url] = key
				mu.Unlock()
			} else {
				key = k
				mu.Unlock()
			}
		}

		resp := struct {
			Url string `json:"url"`
			Key string `json:"key"`
		}{
			Url: q.Url,
			Key: key,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("GET /go/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")

		mu.RLock()
		path, ok := keyToUrl[key]
		mu.RUnlock()

		if !ok {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.Header().Set("Location", path)
			w.WriteHeader(http.StatusFound)
		}
	})

	serv := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: mux,
	}

	err := serv.ListenAndServe()
	if err != nil {
		fmt.Println(err.Error())
	}
}
