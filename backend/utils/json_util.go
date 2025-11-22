package utils

import (
	"reflect"
	"strings"
)

func buildJSONTagMap(obj any) (jsonTagMap map[string]string) {
	jsonTagMap = map[string]string{}

	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" && jsonTag != "-" {
			parts := strings.Split(jsonTag, ",")
			jsonTagMap[field.Name] = parts[0]
		}
	}
	return
}
