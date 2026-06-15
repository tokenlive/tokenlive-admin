package json

import (
	"fmt"
	"reflect"

	jsoniter "github.com/json-iterator/go"
)

var (
	json          = jsoniter.ConfigCompatibleWithStandardLibrary
	Marshal       = json.Marshal
	Unmarshal     = json.Unmarshal
	MarshalIndent = json.MarshalIndent
	NewDecoder    = json.NewDecoder
	NewEncoder    = json.NewEncoder
)

func MarshalToString(v interface{}) *string {
	if v == nil {
		return nil
	}
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		if val.IsNil() {
			return nil
		}
	}
	s, err := jsoniter.MarshalToString(v)
	if err != nil {
		fmt.Println("Failed to marshal json string: " + err.Error())
		return nil
	}
	return &s
}

func UnMarshalToObject(v string, obj interface{}) {
	err := jsoniter.UnmarshalFromString(v, obj)
	if err != nil {
		fmt.Println("Failed to marshal json string: " + err.Error())
	}
}

func UnMarshalToMap(v string) (map[string]string, error) {
	m := make(map[string]string)
	err := json.Unmarshal([]byte(v), &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}
