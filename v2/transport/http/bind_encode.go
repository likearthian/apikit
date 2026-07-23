package http

import (
	"encoding"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
)

func encodeData(q url.Values, ptr interface{}, tag string) error {
	if ptr == nil {
		return nil
	}
	val := reflect.ValueOf(ptr)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return fmt.Errorf("encoded element must not be a nil pointer")
		}
		val = val.Elem()
	}
	if !val.IsValid() {
		return fmt.Errorf("encoded element must be valid")
	}
	typ := val.Type()

	// Map
	if val.Kind() == reflect.Map {
		if val.Type().Key().Kind() != reflect.String {
			return fmt.Errorf("map object should have string keys to be encoded")
		}

		iter := val.MapRange()
		for iter.Next() {
			k := iter.Key().String()
			v := iter.Value()
			if str, ok := marshalField(v.Type().Kind(), v); ok {
				q.Add(k, str)
				continue
			}
			if v.Kind() == reflect.Slice && v.Len() > 0 {
				for i := 0; i < v.Len(); i++ {
					sval := v.Index(i)
					if str := setToString(v.Type().Elem().Kind(), sval); str != "" {
						q.Add(k, str)
					}
				}
			} else if str := setToString(v.Kind(), v); str != "" {
				q.Add(k, str)
			}
		}

		return nil
	}

	// !struct
	if typ.Kind() != reflect.Struct {
		return fmt.Errorf("encoded element must be a struct. got %s", typ.Kind().String())
	}

	for i := 0; i < typ.NumField(); i++ {
		typeField := typ.Field(i)
		if typeField.PkgPath != "" {
			continue
		}
		structField := val.Field(i)
		if !structField.IsValid() || !structField.CanInterface() {
			continue
		}

		structFieldKind := structField.Kind()
		inputFieldName := typeField.Tag.Get(tag)

		if inputFieldName == "" {
			inputFieldName = typeField.Name
			// If tag is nil, we inspect if the field is a struct.
			if structFieldKind == reflect.Struct {
				if err := encodeData(q, structField.Interface(), tag); err != nil {
					return err
				}
				continue
			}
		}

		// Call this first, in case we're dealing with an alias to an array type
		if str, ok := marshalField(typeField.Type.Kind(), structField); ok {
			if str != "" {
				q.Add(inputFieldName, str)
			}
			continue
		}

		if structFieldKind == reflect.Slice {
			numElems := structField.Len()
			if numElems == 0 {
				continue
			}

			sliceOf := structField.Type().Elem().Kind()
			for j := 0; j < numElems; j++ {
				if str := setToString(sliceOf, structField.Index(j)); str != "" {
					q.Add(inputFieldName, str)
				}
			}
		} else if str := setToString(structField.Kind(), structField); str != "" {
			q.Add(inputFieldName, str)
		}
	}
	return nil
}

func marshalField(valueKind reflect.Kind, value reflect.Value) (string, bool) {
	if !value.IsValid() {
		return "", false
	}
	switch valueKind {
	case reflect.Ptr:
		return marshalFieldPtr(value)
	default:
		return marshalFieldNonPtr(value)
	}
}

func marshalFieldNonPtr(value reflect.Value) (string, bool) {
	if !value.IsValid() || !value.CanInterface() {
		return "", false
	}
	fieldIValue := value.Interface()
	fieldValue := reflect.ValueOf(fieldIValue)
	if fieldValue.IsValid() {
		switch fieldValue.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
			if fieldValue.IsNil() {
				return "", false
			}
		}
	}
	if marshaler, ok := fieldIValue.(encoding.TextMarshaler); ok {
		if val, err := marshaler.MarshalText(); err == nil {
			return string(val), true
		}
	}
	return "", false
}

func marshalFieldPtr(value reflect.Value) (string, bool) {
	if !value.IsValid() || value.IsNil() {
		return "", false
	}
	return marshalFieldNonPtr(value.Elem())
}

func setToString(valueKind reflect.Kind, value reflect.Value) string {
	if !value.IsValid() {
		return ""
	}
	if (value.Kind() == reflect.Ptr || value.Kind() == reflect.Interface) && value.IsNil() {
		return ""
	}

	if str, ok := marshalField(valueKind, value); ok {
		return str
	}

	switch value.Kind() {
	case reflect.Ptr, reflect.Interface:
		elem := value.Elem()
		return setToString(elem.Kind(), elem)
	case reflect.Int:
		return strconv.FormatInt(value.Int(), 10)
	case reflect.Int8:
		return strconv.FormatInt(value.Int(), 10)
	case reflect.Int16:
		return strconv.FormatInt(value.Int(), 10)
	case reflect.Int32:
		return strconv.FormatInt(value.Int(), 10)
	case reflect.Int64:
		return strconv.FormatInt(value.Int(), 10)
	case reflect.Uint:
		return strconv.FormatUint(value.Uint(), 10)
	case reflect.Uint8:
		return strconv.FormatUint(value.Uint(), 10)
	case reflect.Uint16:
		return strconv.FormatUint(value.Uint(), 10)
	case reflect.Uint32:
		return strconv.FormatUint(value.Uint(), 10)
	case reflect.Uint64:
		return strconv.FormatUint(value.Uint(), 10)
	case reflect.Bool:
		return strconv.FormatBool(value.Bool())
	case reflect.Float32:
		return strconv.FormatFloat(value.Float(), 'f', -1, 32)
	case reflect.Float64:
		return strconv.FormatFloat(value.Float(), 'f', -1, 64)
	case reflect.String:
		return value.String()
	}
	return ""
}
