//go:build !solution

package jsonlist

import (
	"encoding/json"
	"io"
	"reflect"
)

func Marshal(w io.Writer, slice any) error {
	rv := reflect.ValueOf(slice)
	if rv.Kind() != reflect.Slice {
		return &json.UnsupportedTypeError{Type: reflect.TypeOf(slice)}
	}

	enc := json.NewEncoder(w)
	var err error
	for i := range rv.Len() {
		err = enc.Encode(rv.Index(i).Interface())
		if err != nil {
			return err
		}
	}
	return nil
}

func Unmarshal(r io.Reader, slice any) error { //{"A": 1} {"B": 2} {"C": 3}
	rvPtr := reflect.ValueOf(slice)
	if rvPtr.Kind() != reflect.Pointer {
		return &json.UnsupportedTypeError{Type: reflect.TypeOf(slice)}
	}

	rv := rvPtr.Elem()
	if rv.Kind() != reflect.Slice {
		return &json.UnsupportedTypeError{Type: reflect.TypeOf(slice)}
	}

	dec := json.NewDecoder(r)
	for dec.More() {
		zeroPtr := reflect.New(rv.Type().Elem())
		err := dec.Decode(zeroPtr.Interface())
		if err != nil && err != io.EOF {
			return err
		}

		rv.Set(reflect.Append(rv, zeroPtr.Elem()))
	}
	return nil
}
