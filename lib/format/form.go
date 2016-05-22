package format

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
)

// FormValues takes a struct with `form:` annotations, and transforms them
// into a url.Values map. It can handle both base types, as well as structs.
func FormValues(f interface{}) url.Values {
	form := url.Values{}
	val := reflect.Indirect(reflect.ValueOf(f))

	for i := 0; i < val.NumField(); i++ {
		valueField := val.Field(i)
		typeField := val.Type().Field(i)
		tag := typeField.Tag

		if valueField.Interface() == reflect.Zero(valueField.Type()).Interface() {
			continue
		}

		formKey := tag.Get("form")
		if formKey == "" {
			continue
		}

		var v string

		switch valueField.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			v = strconv.FormatInt(valueField.Int(), 10)
			form.Set(formKey, v)
		case reflect.Bool:
			v = strconv.FormatBool(valueField.Bool())
			form.Set(formKey, v)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			v = strconv.FormatUint(valueField.Uint(), 10)
			form.Set(formKey, v)
		case reflect.Float32:
			v = strconv.FormatFloat(valueField.Float(), 'f', 4, 32)
			form.Set(formKey, v)
		case reflect.Float64:
			v = strconv.FormatFloat(valueField.Float(), 'f', 4, 64)
			form.Set(formKey, v)
		case reflect.Slice:
			// Assume that this is a slice of bytes. Will panic if not.
			v = string(valueField.Bytes())
			form.Set(formKey, v)
		case reflect.String:
			v = valueField.String()
			form.Set(formKey, v)
		default:
			subVals := FormValues(valueField.Interface())
			for key, vals := range subVals {
				for _, val := range vals {
					form.Set(fmt.Sprintf("%s[%s]", formKey, key), val)
				}
			}
		}
	}

	return form
}
