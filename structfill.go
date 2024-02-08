package structfill

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func Fill(structType any, inputMap map[string]any, _typeRegistry ...map[string]func() any) error {
	typeRegistry := make(map[string]func() any)
	if len(_typeRegistry) > 0 {
		typeRegistry = _typeRegistry[0]
	}

	structVal := reflect.ValueOf(structType)
	if structVal.Kind() != reflect.Ptr || structVal.Elem().Kind() != reflect.Struct {
		return errors.New("provided type must be a pointer to a struct")
	}
	structVal = structVal.Elem()
	structTypeVal := structVal.Type()

	for i := 0; i < structVal.NumField(); i++ {
		field := structVal.Field(i)
		fieldType := structTypeVal.Field(i)

		if !field.CanSet() {
			continue
		}

		if fieldType.Anonymous && field.Kind() == reflect.Struct {
			// Recursively fill embedded structs
			err := Fill(field.Addr().Interface(), inputMap, typeRegistry)
			if err != nil {
				return err
			}
		} else {
			err := fillStructField(field, fieldType, inputMap, typeRegistry)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func fillStructField(field reflect.Value, fieldType reflect.StructField, inputMap map[string]any, typeRegistry map[string]func() any) error {
	fieldName := fieldType.Name
	tag := fieldType.Tag
	inputValue, ok := inputMap[strings.ToLower(fieldName)]

	if field.Kind() == reflect.Struct && !fieldType.Anonymous {
		// Handle nested (non-embedded) structs
		if ok {
			nestedMap, ok := inputValue.(map[string]any)
			if !ok {
				return fmt.Errorf("invalid type for field %s, expected map[string]any for nested struct", fieldName)
			}
			err := Fill(field.Addr().Interface(), nestedMap, typeRegistry)
			if err != nil {
				return err
			}
		} else {
			// Set default values for nested structs if not in input map
			setDefaultValues(field, tag)
		}
		return nil
	}

	if !ok {
		// Field name not in map, set default value if specified
		setDefaultValues(field, tag)
		return nil // Skip further processing
	}

	switch field.Kind() {
	case reflect.String:
		if val, ok := inputValue.(string); ok {
			field.SetString(val)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(fmt.Sprintf("%v", inputValue), 10, field.Type().Bits())
		if err != nil {
			return err
		}
		if err := validateIntField(tag, intVal); err != nil {
			return err
		}
		field.SetInt(intVal)
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(fmt.Sprintf("%v", inputValue))
		if err != nil {
			return err
		}
		field.SetBool(boolVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(fmt.Sprintf("%v", inputValue), field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetFloat(floatVal)
	case reflect.Slice:
		inputValueReflect := reflect.ValueOf(inputValue)
		if inputValueReflect.Kind() != reflect.Slice {
			return fmt.Errorf("invalid type for field %s, expected slice", fieldName)
		}

		sliceType := field.Type().Elem()
		slice := reflect.MakeSlice(reflect.SliceOf(sliceType), inputValueReflect.Len(), inputValueReflect.Len())

		if sliceType.Kind() == reflect.Interface {
			for j := 0; j < inputValueReflect.Len(); j++ {
				elemMap, ok := inputValueReflect.Index(j).Interface().(map[string]any)
				if !ok {
					return fmt.Errorf("expected map for interface slice element")
				}

				typeIdentifier, ok := elemMap["type"].(string)
				if !ok {
					return fmt.Errorf("type identifier %s missing for interface slice element", typeIdentifier)
				}
				if typeRegistry[typeIdentifier] == nil {
					return fmt.Errorf("type identifier %s not found in type registry %v", typeIdentifier, typeRegistry)
				}

				newInstance := typeRegistry[typeIdentifier]()   // Instantiate new type
				err := Fill(newInstance, elemMap, typeRegistry) // Recursive call to fill the new instance
				if err != nil {
					return err
				}

				fmt.Printf("Type of slice: %T\n", slice.Interface())
				fmt.Printf("Type of newInstance: %T\n", newInstance)
				fmt.Printf("Expected type in slice at index %d: %v\n", j, slice.Index(j).Type())
				fmt.Printf("Value of newInstance: %#v\n", newInstance)
				fmt.Printf("Slice content before operation: %#v\n", slice.Interface())
				slice.Index(j).Set(reflect.ValueOf(newInstance).Elem()) // Make sure to set the instantiated type back to the slice
			}
			field.Set(slice)
		} else {
			for j := 0; j < inputValueReflect.Len(); j++ {
				elem := inputValueReflect.Index(j)
				if sliceType.Kind() == reflect.Struct {
					if elem.Kind() == reflect.Map {
						nestedMap, ok := elem.Interface().(map[string]any)
						if !ok {
							return fmt.Errorf("invalid type for slice element in field %s, expected map[string]any for nested struct slice element", fieldName)
						}
						err := Fill(slice.Index(j).Addr().Interface(), nestedMap, typeRegistry)
						if err != nil {
							return err
						}
					} else {
						return fmt.Errorf("invalid type for struct slice element in field %s", fieldName)
					}
				} else {
					// Convert each element to the correct type and set it in the slice
					if !slice.Index(j).CanSet() {
						return fmt.Errorf("cannot set slice element in field %s", fieldName)
					}
					newValue, err := convertType(elem.Interface(), sliceType)
					if err != nil {
						return fmt.Errorf("error converting slice element for field %s: %v", fieldName, err)
					}
					slice.Index(j).Set(reflect.ValueOf(newValue))
				}
			}
			field.Set(slice)
		}
	case reflect.Map:
		inputMapReflectValue := reflect.ValueOf(inputValue)
		if inputMapReflectValue.Kind() != reflect.Map {
			return fmt.Errorf("invalid type for field %s, expected a map", fieldName)
		}

		mapType := field.Type()
		newMap := reflect.MakeMapWithSize(mapType, inputMapReflectValue.Len())

		for _, key := range inputMapReflectValue.MapKeys() {
			val := inputMapReflectValue.MapIndex(key)

			// Convert key to the map's key type
			convertedKey := key.Convert(mapType.Key())

			// Convert value to the map's value type
			convertedVal := val.Convert(mapType.Elem())

			newMap.SetMapIndex(convertedKey, convertedVal)
		}

		field.Set(newMap)
	default:
		return fmt.Errorf("unsupported type: %v", field.Kind())
	}
	return nil
}

func setPrimitiveType(field reflect.Value, value any) bool {
	switch field.Kind() {
	case reflect.String:
		val, ok := value.(string)
		if ok {
			field.SetString(val)
			return true
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, ok := value.(int64) // Assuming the input is int64; adjust based on your input data
		if ok {
			field.SetInt(val)
			return true
		}
	case reflect.Bool:
		val, ok := value.(bool)
		if ok {
			field.SetBool(val)
			return true
		}
	case reflect.Float32, reflect.Float64:
		val, ok := value.(float64) // Assuming the input is float64; adjust based on your input data
		if ok {
			field.SetFloat(val)
			return true
		}
	}
	return false
}

func validateIntField(tag reflect.StructTag, value int64) error {
	validateTag := tag.Get("validate")
	if validateTag == "" {
		return nil // No validation rules
	}

	rules := strings.Split(validateTag, ",")
	for _, rule := range rules {
		ruleParts := strings.SplitN(rule, "=", 2)
		if len(ruleParts) != 2 {
			return errors.New("invalid validate tag format")
		}

		ruleValue, err := strconv.ParseInt(ruleParts[1], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid rule value: %v", err)
		}

		switch ruleParts[0] {
		case "min":
			if value < ruleValue {
				return fmt.Errorf("value %d is less than min %d", value, ruleValue)
			}
		case "max":
			if value > ruleValue {
				return fmt.Errorf("value %d is greater than max %d", value, ruleValue)
			}
		default:
			return fmt.Errorf("unsupported validation rule: %s", ruleParts[0])
		}
	}
	return nil
}

func setDefaultValues(field reflect.Value, tag reflect.StructTag) {
	// Direct default value setting for non-struct fields
	defaultVal := tag.Get("default")
	if defaultVal != "" {
		switch field.Kind() {
		case reflect.String:
			field.SetString(defaultVal)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			intVal, err := strconv.ParseInt(defaultVal, 10, 64)
			if err == nil {
				field.SetInt(intVal)
			}
		case reflect.Bool:
			boolVal, err := strconv.ParseBool(defaultVal)
			if err == nil {
				field.SetBool(boolVal)
			}
		case reflect.Float32, reflect.Float64:
			floatVal, err := strconv.ParseFloat(defaultVal, 64)
			if err == nil {
				field.SetFloat(floatVal)
			}
		}
		return // Return after setting a direct default value
	}

	// Recursively set default values for nested structs
	if field.Kind() == reflect.Struct {
		for i := 0; i < field.NumField(); i++ {
			nestedField := field.Field(i)
			nestedFieldType := field.Type().Field(i)
			if nestedField.CanSet() {
				setDefaultValues(nestedField, nestedFieldType.Tag)
			}
		}
	}
}

func convertType(value any, targetType reflect.Type) (any, error) {
	val := reflect.ValueOf(value)
	if val.Type().ConvertibleTo(targetType) {
		return val.Convert(targetType).Interface(), nil
	}
	return nil, fmt.Errorf("cannot convert %v to %v", val.Type(), targetType)
}
