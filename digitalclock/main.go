//go:build !solution

package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func DrawTimeImage(t time.Time, k int) image.Image {
	timeStr := t.Format("15:04:05")

	symbolMap := map[rune]string{
		'0': Zero, '1': One, '2': Two, '3': Three, '4': Four,
		'5': Five, '6': Six, '7': Seven, '8': Eight, '9': Nine,
		':': Colon,
	}

	zeroLines := strings.Split(Zero, "\n")
	h := len(zeroLines)
	w := len(zeroLines[0])
	wColon := len(strings.Split(Colon, "\n")[0])

	imgW := (6*w + 2*wColon) * k
	imgH := h * k
	img := image.NewRGBA(image.Rect(0, 0, imgW, imgH))

	white := color.RGBA{255, 255, 255, 255}

	currentX := 0
	for _, char := range timeStr {
		symbolLines := strings.Split(symbolMap[char], "\n")
		symbolW := len(symbolLines[0])

		for y, line := range symbolLines {
			for x, pixel := range line {
				var paint color.Color = white
				if pixel == '1' {
					paint = Cyan
				}

				for ik := 0; ik < k; ik++ {
					for jk := 0; jk < k; jk++ {
						img.Set((currentX+x)*k+ik, y*k+jk, paint)
					}
				}
			}
		}
		currentX += symbolW
	}

	return img
}

func SendDrawImage(t time.Time, k int, w http.ResponseWriter) {
	img := DrawTimeImage(t, k)
	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
	_ = png.Encode(w, img)
}

func ParseKParam(params url.Values, w http.ResponseWriter) (int, error) {
	val, ok := params["k"]
	if !ok || len(val) != 1 {
		w.WriteHeader(http.StatusBadRequest)
		return 0, errors.New("k param is required")
	}

	k, err := strconv.Atoi(val[0])
	if err != nil || !(1 <= k && k <= 30) {
		w.WriteHeader(http.StatusBadRequest)
		return 0, errors.New("bad format")
	}
	return k, nil
}

func ParseTimeParam(params url.Values, w http.ResponseWriter) (time.Time, error) {
	val, ok := params["time"]
	if !ok || len(val) != 1 {
		w.WriteHeader(http.StatusBadRequest)
		return time.Time{}, errors.New("time param is required")
	}

	t, err := time.Parse("15:04:05", val[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return time.Time{}, errors.New("bad format")
	}
	return t, nil
}

func main() {
	port := flag.Int("port", 8080, "port to listen on")
	flag.Parse()

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()

		switch len(params) {
		case 0:
			SendDrawImage(time.Now(), 0, w)
		case 1:
			k, err := ParseKParam(params, w)
			if err != nil {
				return
			}

			SendDrawImage(time.Now(), k, w)
		case 2:
			k, err := ParseKParam(params, w)
			if err != nil {
				return
			}

			t, err := ParseTimeParam(params, w)
			if err != nil {
				return
			}

			SendDrawImage(t, k, w)
		default:
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	})

	serv := &http.Server{
		Addr: fmt.Sprintf("localhost:%d", *port),
		Handler: mux,
	}

	err := serv.ListenAndServe()
	if err != nil {
		fmt.Println(err.Error())
	}
}
