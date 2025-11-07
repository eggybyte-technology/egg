// Package internal provides internal implementation for the configx package.
package internal

import (
	"fmt"
	"reflect"
	"strconv"
	"time"
)

// Validator is an optional interface that configuration structs can implement
// to perform custom validation after binding environment variables.
// If the configuration struct implements this interface, Validate() will be called
// automatically after binding all fields.
type Validator interface {
	Validate() error
}

// BindToStruct binds configuration values to struct fields using env tags.
// After binding all fields, if the target implements the Validator interface,
// its Validate() method will be called to perform additional validation or
// post-processing (e.g., parsing structured data from raw strings).
func BindToStruct(snapshot map[string]string, target any, onUpdate func()) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr || targetValue.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to struct")
	}

	// Bind all fields from environment variables
	if err := bindStructFields(snapshot, targetValue.Elem()); err != nil {
		return err
	}

	// Call Validate() if the target implements the Validator interface
	if validator, ok := target.(Validator); ok {
		if err := validator.Validate(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
	}

	return nil
}

// bindStructFields recursively binds configuration values to struct fields.
func bindStructFields(snapshot map[string]string, structValue reflect.Value) error {
	structType := structValue.Type()

	for i := 0; i < structValue.NumField(); i++ {
		field := structValue.Field(i)
		fieldType := structType.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Handle nested structs (embedded or regular)
		if field.Kind() == reflect.Struct {
			if err := bindStructFields(snapshot, field); err != nil {
				return fmt.Errorf("failed to bind nested struct %s: %w", fieldType.Name, err)
			}
			continue
		}

		// Get env tag
		envTag := fieldType.Tag.Get("env")
		if envTag == "" {
			continue
		}

		// Get default value
		defaultValue := fieldType.Tag.Get("default")

		// Get value from snapshot or use default
		value, exists := snapshot[envTag]
		if !exists {
			value = defaultValue
		}

		// Set field value
		if err := setFieldValue(field, value); err != nil {
			return fmt.Errorf("failed to set field %s: %w", fieldType.Name, err)
		}
	}

	return nil
}

// setFieldValue sets a field value from a string.
func setFieldValue(field reflect.Value, value string) error {
	if value == "" {
		return nil // Keep zero value
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			duration, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			field.SetInt(int64(duration))
		} else {
			intValue, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return err
			}
			field.SetInt(intValue)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintValue, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(uintValue)
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolValue)
	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatValue)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}

	return nil
}
