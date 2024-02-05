package structfill

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func FillStructFromMap(structType any, inputMap map[string]any) error {
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
				err := FillStructFromMap(field.Addr().Interface(), nestedMap)
				if err != nil {
					return err
				}
			} else {
				// Initialize nested structs with their default values recursively
				err := FillStructFromMap(field.Addr().Interface(), make(map[string]any))
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
