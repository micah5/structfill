package structfill

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func StructFill(structType any, inputMap map[string]any) error {
	structVal := reflect.ValueOf(structType)
	if structVal.Kind() != reflect.Ptr || structVal.Elem().Kind() != reflect.Struct {
		return errors.New("provided type must be a pointer to a struct")
	}
	structVal = structVal.Elem()

	for i := 0; i < structVal.NumField(); i++ {
		field := structVal.Field(i)
		if !field.CanSet() {
			continue
		}

		fieldType := structVal.Type().Field(i)
		fieldName := fieldType.Name
		tag := fieldType.Tag
		inputValue, ok := inputMap[strings.ToLower(fieldName)]

		// Handle nested structs recursively
		if field.Kind() == reflect.Struct {
			if ok && inputValue != nil {
				nestedMap, ok := inputValue.(map[string]any)
				if !ok {
					return fmt.Errorf("invalid type for field %s, expected map[string]any for nested struct", fieldName)
				}
				err := StructFill(field.Addr().Interface(), nestedMap)
				if err != nil {
					return err
				}
			} else {
				// Initialize nested structs with their default values recursively
				err := StructFill(field.Addr().Interface(), make(map[string]any))
				if err != nil {
					return err
				}
			}
			continue
		}

		if ok {
			inputString := fmt.Sprintf("%v", inputValue)
			switch field.Kind() {
			case reflect.String:
				field.SetString(inputString)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				intVal, err := strconv.ParseInt(inputString, 10, 64)
				if err != nil {
					return err
				}
				if err := validateIntField(tag, intVal); err != nil {
					return err
				}
				field.SetInt(intVal)
			case reflect.Bool:
				boolVal, err := strconv.ParseBool(inputString)
				if err != nil {
					return err
				}
				field.SetBool(boolVal)
			case reflect.Float32, reflect.Float64:
				floatVal, err := strconv.ParseFloat(inputString, 64)
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

				for j := 0; j < inputValueReflect.Len(); j++ {
					elem := inputValueReflect.Index(j)
					if sliceType.Kind() == reflect.Struct {
						if elem.Kind() == reflect.Map {
							nestedMap, ok := elem.Interface().(map[string]any)
							if !ok {
								return fmt.Errorf("invalid type for slice element in field %s, expected map[string]any for nested struct slice element", fieldName)
							}
							err := StructFill(slice.Index(j).Addr().Interface(), nestedMap)
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
		} else {
			// Handle default values
			setDefaultValues(field, tag)
		}
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
	defaultVal := tag.Get("default")
	if defaultVal == "" {
		return // No default value
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(defaultVal)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(defaultVal, 10, 64)
		if err != nil {
			return // Skip setting default value on error
		}
		field.SetInt(intVal)
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(defaultVal)
		if err != nil {
			return // Skip setting default value on error
		}
		field.SetBool(boolVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(defaultVal, 64)
		if err != nil {
			return // Skip setting default value on error
		}
		field.SetFloat(floatVal)
	default:
		return // Skip setting default value for unsupported types
	}
}

func convertType(value any, targetType reflect.Type) (any, error) {
	val := reflect.ValueOf(value)
	if val.Type().ConvertibleTo(targetType) {
		return val.Convert(targetType).Interface(), nil
	}
	return nil, fmt.Errorf("cannot convert %v to %v", val.Type(), targetType)
}
