package structfill

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type Address struct {
	Street string `default:"Main St"`
	City   string
	Height float64 `default:"1.8" validate:"min=1.5,max=2.0"`
}

type Employee struct {
	Name    string `default:"John Doe"`
	Age     int    `default:"30" validate:"min=18,max=65"`
	Address Address
}

func TestFillStructFromMap_SimpleStruct(t *testing.T) {
	var person Employee
	inputMap := map[string]any{
		"name": "Alice",
		"age":  29,
	}

	err := FillStructFromMap(&person, inputMap)
	assert.NoError(t, err)
	assert.Equal(t, Employee{Name: "Alice", Age: 29, Address: Address{Street: "Main St", Height: 1.8}}, person)
}

func TestFillStructFromMap_WithDefaults(t *testing.T) {
	var person Employee
	inputMap := map[string]any{}

	err := FillStructFromMap(&person, inputMap)
	assert.NoError(t, err)
	assert.Equal(t, Employee{Name: "John Doe", Age: 30, Address: Address{Street: "Main St", Height: 1.8}}, person)
}

func TestFillStructFromMap_WithNestedStruct(t *testing.T) {
	var person Employee
	inputMap := map[string]any{
		"address": map[string]any{
			"city":   "Springfield",
			"height": 2.0,
		},
	}

	err := FillStructFromMap(&person, inputMap)
	assert.NoError(t, err)
	assert.Equal(t, Employee{Name: "John Doe", Age: 30, Address: Address{Street: "Main St", City: "Springfield", Height: 2.0}}, person)
}

func TestFillStructFromMap_ValidationError(t *testing.T) {
	var person Employee
	inputMap := map[string]any{
		"age": 17, // Below the minimum age defined in the `validate` tag
	}

	err := FillStructFromMap(&person, inputMap)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "value 17 is less than min 18")
}

func TestFillStructFromMap_NonPointerInput(t *testing.T) {
	person := Employee{} // Not a pointer
	inputMap := map[string]any{}

	err := FillStructFromMap(person, inputMap)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provided type must be a pointer to a struct")
}
