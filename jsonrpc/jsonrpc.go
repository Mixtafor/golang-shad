//go:build !solution

package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"unsafe"
)

type App struct {
	Mux *http.ServeMux
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.Mux.ServeHTTP(w, r)
}

func CheckRPCMethod(m reflect.Method) bool {
	isValParams := m.Type.NumIn() == 3 && m.Type.NumOut() == 2
	if !isValParams {
		return false
	}

	isCtx := m.Type.In(1).AssignableTo(reflect.TypeFor[context.Context]())
	if !isCtx {
		return false
	}
	isErr := m.Type.Out(1)

	if !isErr.AssignableTo(reflect.TypeFor[error]()) {
		return false
	}

	return true
}

func MakeHandler(service interface{}) http.Handler {
	mux := http.NewServeMux()

	rv := reflect.ValueOf(service)
	if rv.Kind() == reflect.Invalid {
		panic("invalid service")
	}

	for i := range rv.NumMethod() {
		m := rv.Type().Method(i)
		if CheckRPCMethod(m) {
			mux.HandleFunc(fmt.Sprintf("/%s", m.Name), func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				var err error
				defer r.Body.Close()

				req := reflect.New(m.Type.In(2).Elem())
				err = json.NewDecoder(r.Body).Decode(req.Interface())
				if err != nil {
					http.Error(w, "internal err", http.StatusInternalServerError)
					return
				}

				ret := rv.Method(i).Call([]reflect.Value{
					reflect.ValueOf(ctx),
					req,
				})

				errTyped, _ := ret[1].Interface().(error)
				if errTyped != nil {
					http.Error(w, errTyped.Error(), http.StatusInternalServerError)
					return
				}
				json.NewEncoder(w).Encode(ret[0].Interface())
			})
		}
	}

	return &App{
		Mux: mux,
	}
}

func Call(ctx context.Context, endpoint string, method string, req, rsp interface{}) error {
	cli := &http.Client{
		Transport: http.DefaultTransport,
	}

	pr, pw := io.Pipe()
	defer pr.Close()
	go func() {
		err := json.NewEncoder(pw).Encode(req)
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		pw.Close()
	}()

	realEndpoint, _ := url.JoinPath(endpoint, method)
	request, err := http.NewRequestWithContext(ctx, "POST", realEndpoint, pr)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := cli.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode/100 == 5 {
		rspData, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}
		return errors.New(unsafe.String(unsafe.SliceData(rspData), len(rspData)))
	}

	err = json.NewDecoder(response.Body).Decode(rsp)
	if err != nil {
		return err
	}

	return nil
}
