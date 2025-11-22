package utils

import (
	"strings"

	"github.com/beego/beego/v2/core/validation"
)

func ValidateField(obj any) (ok bool, errors map[string][]string, err error) {
	errors = map[string][]string{}
	v := validation.Validation{}

	if ok, err = v.Valid(obj); err != nil {
		return
	} else if !ok {
		jsonTagMap := buildJSONTagMap(obj)
		for _, e := range v.Errors {
			field := e.Field
			if jsonTag, exists := jsonTagMap[field]; exists {
				field = jsonTag
			}

			message := e.Message
			if strings.Contains(message, "empty") {
				errors["missingField"] = append(errors["missingField"], field)
			} else {
				errors["invalidField"] = append(errors["invalidField"], field)
			}
		}
		return
	}
	return
}
