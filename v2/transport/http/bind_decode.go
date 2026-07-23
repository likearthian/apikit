package http

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

func bindData(ptr interface{}, data map[string][]string, tag string) error {
	if ptr == nil || len(data) == 0 {
		return nil
	}
	typ := reflect.TypeOf(ptr)
	if typ.Kind() != reflect.Ptr {
		return errors.New("destination is not a pointer to struct")
	}
	ptrVal := reflect.ValueOf(ptr)
	if ptrVal.IsNil() {
		return errors.New("destination is a nil pointer")
	}
	typ = typ.Elem()
	val := ptrVal.Elem()

	// Map
	if typ.Kind() == reflect.Map {
		if typ != reflect.TypeOf(map[string]string{}) {
			return errors.New("binding map must be map[string]string")
		}
		if val.IsNil() {
			val.Set(reflect.MakeMap(typ))
		}
		for k, v := range data {
			if len(v) == 0 {
				continue
			}
			val.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v[0]))
		}
		return nil
	}

	// !struct
	if typ.Kind() != reflect.Struct {
		return fmt.Errorf("binding element must be a struct. got %s", typ.Kind().String())
	}

	for i := 0; i < typ.NumField(); i++ {
		typeField := typ.Field(i)
		structField := val.Field(i)
		if !structField.CanSet() {
			continue
		}
		structFieldKind := structField.Kind()
		inputFieldName := typeField.Tag.Get(tag)

		if inputFieldName == "" {
			inputFieldName = typeField.Name
			// If tag is nil, we inspect if the field is a struct.
			if structFieldKind == reflect.Struct {
				if err := bindData(structField.Addr().Interface(), data, tag); err != nil {
					return err
				}
				continue
			}
			//if _, ok := structField.Addr().Interface().(BindUnmarshaler); !ok && structFieldKind == reflect.Struct {
			//	if err := bindData(structField.Addr().Interface(), data, tag); err != nil {
			//		return err
			//	}
			//	continue
			//}
		}

		rawInputValue, exists, err := lookupInputValue(data, inputFieldName)
		if err != nil {
			return err
		}

		if !exists {
			continue
		}

		//this part is to handle comma separated value
		var inputValue []string
		for _, val := range rawInputValue {
			strSlice := strings.Split(val, ",")
			inputValue = append(inputValue, strSlice...)
		}

		if inputValue == nil {
			continue
		}

		// Call this first, in case we're dealing with an alias to an array type
		if ok, err := unmarshalField(typeField.Type.Kind(), inputValue[0], structField); ok {
			if err != nil {
				return &BindingError{Field: inputFieldName, Value: inputValue[0], Err: err}
			}
			continue
		}

		numElems := len(inputValue)
		if structFieldKind == reflect.Slice && numElems > 0 {
			sliceOf := structField.Type().Elem().Kind()
			slice := reflect.MakeSlice(structField.Type(), numElems, numElems)
			for j := 0; j < numElems; j++ {
				if err := setWithProperType(sliceOf, inputValue[j], slice.Index(j)); err != nil {
					return &BindingError{Field: inputFieldName, Value: inputValue[j], Err: err}
				}
			}
			val.Field(i).Set(slice)
		} else if err := setWithProperType(typeField.Type.Kind(), inputValue[0], structField); err != nil {
			return &BindingError{Field: inputFieldName, Value: inputValue[0], Err: err}
		}
	}
	return nil
}

func lookupInputValue(data map[string][]string, field string) ([]string, bool, error) {
	if value, exists := data[field]; exists {
		return value, true, nil
	}

	var matchingKeys []string
	for key := range data {
		if strings.EqualFold(key, field) {
			matchingKeys = append(matchingKeys, key)
		}
	}
	sort.Strings(matchingKeys)

	switch len(matchingKeys) {
	case 0:
		return nil, false, nil
	case 1:
		return data[matchingKeys[0]], true, nil
	default:
		return nil, false, fmt.Errorf(
			"ambiguous input keys for field %q: %s",
			field,
			strings.Join(matchingKeys, ", "),
		)
	}
}
