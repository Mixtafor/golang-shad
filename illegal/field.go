//go:build !solution

package illegal

import (
	"reflect"
	"unsafe"
)

type eface struct {
	_type unsafe.Pointer
	data  unsafe.Pointer
}

func SetPrivateField(obj interface{}, name string, value interface{}) {
	dataPtr := (*eface)(unsafe.Pointer(&obj)).data
	fieldTypeInfo, _ := reflect.ValueOf(obj).Elem().Type().FieldByName(name)
	fieldPtr := unsafe.Add(dataPtr, fieldTypeInfo.Offset)
	reflect.NewAt(fieldTypeInfo.Type, fieldPtr).Elem().Set(reflect.ValueOf(value))
}
