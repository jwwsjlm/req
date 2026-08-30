package util

import (
	"encoding/base64"
	"os"
	"reflect"
	"strings"
)

// IsJSONType method is to check JSON content type or not
func IsJSONType(ct string) bool {
	return strings.Contains(ct, "json")
}

// IsXMLType method is to check XML content type or not
func IsXMLType(ct string) bool {
	return strings.Contains(ct, "xml")
}

// GetPointer return the pointer of the interface.
func GetPointer(v any) any {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		if tt := t.Elem(); tt.Kind() == reflect.Ptr { // pointer of pointer
			if tt.Elem().Kind() == reflect.Ptr {
				panic("pointer of pointer of pointer is not supported")
			}
			el := reflect.ValueOf(v).Elem()
			if el.IsZero() {
				vv := reflect.New(tt.Elem())
				el.Set(vv)
				return vv.Interface()
			} else {
				return el.Interface()
			}
		} else {
			if reflect.ValueOf(v).IsZero() {
				vv := reflect.New(t.Elem())
				return vv.Interface()
			}
			return v
		}
	}
	return reflect.New(t).Interface()
}

// GetType return the underlying type.
func GetType(v any) reflect.Type {
	return reflect.Indirect(reflect.ValueOf(v)).Type()
}

// IsStringEmpty method tells whether given string is empty or not
func IsStringEmpty(str string) bool {
	return len(strings.TrimSpace(str)) == 0
}

// See 2 (end of page 4) https://www.ietf.org/rfc/rfc2617.txt
// "To receive authorization, the client sends the userid and password,
// separated by a single colon (":") character, within a base64
// encoded string in the credentials."
// It is not meant to be urlencoded.
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// BasicAuthHeaderValue return the header of basic auth.
func BasicAuthHeaderValue(username, password string) string {
	return "Basic " + basicAuth(username, password)
}

// CreateDirectory create the directory.
func CreateDirectory(dir string) (err error) {
	if _, err = os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(dir, 0755); err != nil {
				return
			}
		}
	}
	return
}
