//go:build !solution

package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

type Rule struct {
	Endpoint      string   `yaml:"endpoint"`
	ForbUserAg    []string `yaml:"forbidden_user_agents"`
	ForbHeaders   []string `yaml:"forbidden_headers"`
	ReqHeaders    []string `yaml:"required_headers"`
	MaxReqLen     int      `yaml:"max_request_length_bytes"`
	MaxRespLen    int      `yaml:"max_response_length_bytes"`
	ForbRespCodes []int    `yaml:"forbidden_response_codes"`
	ForbReqRe     []string `yaml:"forbidden_request_re"`
	ForbRespRe    []string `yaml:"forbidden_response_re"`
}

type RuleWithRegexp struct {
	ForbUserAg    []*regexp.Regexp
	ForbHeaders   map[string]*regexp.Regexp //[]*regexp.Regexp
	ReqHeaders    []string
	MaxReqLen     int
	MaxRespLen    int
	ForbRespCodes map[int]struct{}
	ForbReqRe     []*regexp.Regexp
	ForbRespRe    []*regexp.Regexp
}

type Config struct {
	Rules []Rule `yaml:"rules"`
}

var rules map[string]RuleWithRegexp
var serviceAddr *url.URL


func ThrowForbidden(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte("Forbidden"))
}

func RequestHandler(w http.ResponseWriter, r *http.Request, endpoint string) ([]byte, error) {
	usrAg := r.UserAgent()
	for _, re := range rules[endpoint].ForbUserAg {
		if re.MatchString(usrAg) {
			ThrowForbidden(w, r)
			return nil, errors.New("forbidden")
		}
	}

	for k, head := range rules[endpoint].ForbHeaders {
		vals, ok := r.Header[k]
		if !ok {
			continue
		}

		for _, val := range vals {
			if head.MatchString(val) {
				ThrowForbidden(w, r)
				return nil, errors.New("forbidden")
			}
		}
	}

	for _, head := range rules[endpoint].ReqHeaders {
		_, ok := r.Header[head]
		if !ok {
			ThrowForbidden(w, r)
			return nil, errors.New("forbidden")
		}
	}

	maxLen := rules[endpoint].MaxReqLen
	var data []byte
	if maxLen > 0 {
		var err error
		limitedReader := io.LimitReader(r.Body, int64(maxLen+1))
		data, err = io.ReadAll(limitedReader)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return nil, errors.New("bad request")
		}
		if len(data) > maxLen {
			ThrowForbidden(w, r)
			return nil, errors.New("forbidden")
		}
	} else {
		var err error
		data, err = io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return nil, errors.New("bad request")
		}
	}

	for _, re := range rules[endpoint].ForbReqRe {
		if re.Match(data) {
			ThrowForbidden(w, r)
			return nil, errors.New("forbidden")
		}
	}

	return data, nil
}

func ResponseHandler(w http.ResponseWriter, resp *http.Response, r *http.Request, endpoint string) ([]byte, error) {
	if _, ok := rules[endpoint].ForbRespCodes[resp.StatusCode]; ok {
		ThrowForbidden(w, r)
		return nil, errors.New("bad request")
	}

	maxLen := rules[endpoint].MaxRespLen
	var data []byte
	if maxLen > 0 {
		var err error
		limitedReader := io.LimitReader(resp.Body, int64(maxLen+1))
		data, err = io.ReadAll(limitedReader)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return nil, errors.New("bad request")
		}
		if len(data) > maxLen {
			ThrowForbidden(w, r)
			return nil, errors.New("forbidden")
		}
	} else {
		var err error
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return nil, errors.New("bad request")
		}
	}

	for _, re := range rules[endpoint].ForbRespRe {
		if re.Match(data) {
			ThrowForbidden(w, r)
			return nil, errors.New("forbidden")
		}
	}

	return data, nil
}

func RegisterHandlers(mux *http.ServeMux, client *http.Client) {
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = serviceAddr.Scheme
			req.URL.Host = serviceAddr.Host
		},
	}

	for endpoint := range rules {
		mux.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()

			b, err := RequestHandler(w, r, endpoint)
			if err != nil {
				return
			}

			newReq := r.Clone(r.Context())
			newReq.Body = io.NopCloser(bytes.NewReader(b))
			newReq.URL.Host = serviceAddr.Host
			newReq.URL.Scheme = serviceAddr.Scheme
			newReq.Host = serviceAddr.Host
			newReq.RequestURI = ""

			resp, err := client.Do(newReq)
			if err != nil {
				w.WriteHeader(http.StatusBadGateway)
				return
			}

			defer resp.Body.Close()

			data, err := ResponseHandler(w, resp, r, endpoint)

			if err != nil {
				return
			}

			for k, vv := range resp.Header {
				for _, v := range vv {
					w.Header().Add(k, v)
				}
			}

			w.WriteHeader(resp.StatusCode)

			_, err = io.Copy(w, bytes.NewReader(data))
			if err != nil {
				log.Printf("Error copying body: %v", err)
			}
		})
	}

	if _, ok := rules["/"]; !ok {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			proxy.ServeHTTP(w, r)
		})
	}
}

func main() {
	serv := flag.String("service-addr", "", "service addr to defend")
	conf := flag.String("conf", "", "path to file to conf")
	addr := flag.String("addr", "", "address to listen on")

	flag.Parse()

	var err error
	serviceAddr, err = url.Parse(*serv)
	if err != nil {
		panic(err)
	}

	file, err := os.ReadFile(*conf)
	if err != nil {
		panic(err)
	}
	var config Config
	err = yaml.Unmarshal(file, &config)
	if err != nil {
		panic(err)
	}

	ForbHeadersRegexp := regexp.MustCompile(`([\w-]+): ([\w/-]+)`)

	rules = make(map[string]RuleWithRegexp, len(config.Rules))
	for _, rule := range config.Rules {
		r := RuleWithRegexp{}
		r.ForbHeaders = make(map[string]*regexp.Regexp)
		for _, u := range rule.ForbUserAg {
			r.ForbUserAg = append(r.ForbUserAg, regexp.MustCompile(u))
		}
		for _, h := range rule.ForbHeaders {
			matches := ForbHeadersRegexp.FindStringSubmatch(h)
			r.ForbHeaders[matches[1]] = regexp.MustCompile(matches[2])
		}

		r.ReqHeaders = rule.ReqHeaders

		r.MaxReqLen = rule.MaxReqLen
		r.MaxRespLen = rule.MaxRespLen
		r.ForbRespCodes = make(map[int]struct{})
		for _, code := range rule.ForbRespCodes {
			r.ForbRespCodes[code] = struct{}{}
		}
		for _, re := range rule.ForbReqRe {
			r.ForbReqRe = append(r.ForbReqRe, regexp.MustCompile(re))
		}
		for _, re := range rule.ForbRespRe {
			r.ForbRespRe = append(r.ForbRespRe, regexp.MustCompile(re))
		}
		rules[rule.Endpoint] = r
	}

	client := &http.Client{
		Transport: &http.Transport{},
	}

	mux := http.NewServeMux()

	RegisterHandlers(mux, client)

	srv := &http.Server{
		Addr:    *addr,
		Handler: mux,
	}

	err = srv.ListenAndServe()
	if err != nil {
		panic(err)
	}

}
