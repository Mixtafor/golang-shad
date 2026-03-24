//go:build !solution

package structtags

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

var cache sync.Map

type FieldInfo struct {
	Ind       int
	IsSlice   bool
	InnerType reflect.Type // if IsSlice == true
	// refers to slice elem if IsSlice == true and to Object otherwise
	ValKind int // 0 - String 1 - int 2 - bool 3 - unsupported
}

func AddFieldInfo(ptr interface{}) map[string]FieldInfo {
	fields := make(map[string]FieldInfo)
	v := reflect.TypeOf(ptr).Elem()
	for i := 0; i < v.NumField(); i++ {
		fieldInfo := v.Field(i)
		tag := fieldInfo.Tag
		name := tag.Get("http")
		if name == "" {
			name = strings.ToLower(fieldInfo.Name)
		}
		field := FieldInfo{Ind: i}

		if fieldInfo.Type.Kind() == reflect.Slice {
			field.IsSlice = true
			field.InnerType = fieldInfo.Type.Elem()
		} else {
			field.IsSlice = false
			field.InnerType = fieldInfo.Type
		}
		field = populate(field.InnerType, field)
		fields[name] = field
	}

	cache.Store(v, fields)
	return fields
}

func populate(v reflect.Type, fieldInfo FieldInfo) FieldInfo {
	switch v.Kind() {
	case reflect.String:
		fieldInfo.ValKind = 0

	case reflect.Int:
		fieldInfo.ValKind = 1

	case reflect.Bool:
		fieldInfo.ValKind = 2

	default:
		fieldInfo.ValKind = 3
	}
	return fieldInfo
}

func cachedPopulate(v reflect.Value, value string, fieldInfo FieldInfo) error {
	switch fieldInfo.ValKind {
	case 0:
		v.SetString(value)

	case 1:
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		v.SetInt(i)

	case 2:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		v.SetBool(b)

	default:
		return fmt.Errorf("unsupported kind %s", v.Type())
	}
	return nil
}

func Unpack(req *http.Request, ptr interface{}) error {
	if err := req.ParseForm(); err != nil {
		return err
	}

	var fields map[string]FieldInfo
	cachedFields, hasInfo := cache.Load(reflect.TypeOf(ptr).Elem())
	if hasInfo {
		fields = cachedFields.(map[string]FieldInfo)
	} else {
		fields = AddFieldInfo(ptr)
	}

	for name, values := range req.Form {
		fieldInfo, ok := fields[name]
		if !ok {
			continue
		}

		f := reflect.ValueOf(ptr).Elem().Field(fieldInfo.Ind)

		for _, value := range values {
			if fieldInfo.IsSlice {
				elem := reflect.New(fieldInfo.InnerType).Elem()
				if err := cachedPopulate(elem, value, fieldInfo); err != nil {
					return fmt.Errorf("%s: %v", name, err)
				}
				f.Set(reflect.Append(f, elem))
			} else {
				if err := cachedPopulate(f, value, fieldInfo); err != nil {
					return fmt.Errorf("%s: %v", name, err)
				}
			}
		}
	}
	return nil
}
