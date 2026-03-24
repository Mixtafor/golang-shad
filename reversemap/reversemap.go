//go:build !solution

package reversemap

import "reflect"

func ReverseMap(forward any) any {
	rv := reflect.ValueOf(forward)
	if rv.Kind() != reflect.Map {
		panic("ReverseMap: argument must be a map")
	}

	revMap := reflect.MakeMapWithSize(reflect.MapOf(rv.Type().Elem(), rv.Type().Key()), len(rv.MapKeys()))

	iter := rv.MapRange()
	for iter.Next() {
		revMap.SetMapIndex(iter.Value(), iter.Key())
	}

	return revMap.Interface()
}

//func ReverseMap[K, V comparable](forward map[K]V) map[V]K {
//	revMap := make(map[V]K, len(forward))
//	for k, v := range forward {
//		revMap[v] = k
//	}
//	return revMap
//}
