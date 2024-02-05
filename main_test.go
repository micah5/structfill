package structfill

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// Primitives
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

// Slices
type Classroom struct {
	Building string
	Number   int
}
type School struct {
	Students   []string
	Ages       []int
	Classrooms []Classroom
}

func TestFillStructFromMap_SliceOfPrimitives(t *testing.T) {
	var school School
	inputMap := map[string]any{
		"students": []string{"Alice", "Bob"},
		"ages":     []int{25, 30},
		"classrooms": []map[string]any{
			{"building": "A", "number": 101},
			{"building": "B", "number": 201},
		},
	}

	err := FillStructFromMap(&school, inputMap)
	assert.NoError(t, err)
	assert.Equal(t, School{
		Students: []string{"Alice", "Bob"},
		Ages:     []int{25, 30},
		Classrooms: []Classroom{
			{Building: "A", Number: 101},
			{Building: "B", Number: 201},
		},
	}, school)
}

// Maps
type Simple struct {
	Items  map[string]string
	Items2 map[string]int
}

func TestFillStructFromMap_MapOfPrimitives(t *testing.T) {
	var simple Simple
	inputMap := map[string]any{
		"items": map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		"items2": map[string]int{
			"key1": 1,
			"key2": 2,
		},
	}

	err := FillStructFromMap(&simple, inputMap)
	assert.NoError(t, err)
	assert.Equal(t, Simple{
		Items:  map[string]string{"key1": "value1", "key2": "value2"},
		Items2: map[string]int{"key1": 1, "key2": 2},
	}, simple)
}

type Company struct {
	Team map[string][]Employee
}

func TestFillStructFromMap_NestedMapOfStructs(t *testing.T) {
	var company Company
	inputMap := map[string]any{
		"team": map[string][]Employee{
			"dev": {
				{Name: "Alice", Age: 25},
				{Name: "Bob", Age: 30},
			},
			"qa": {
				{Name: "Charlie", Age: 35},
			},
		},
	}

	err := FillStructFromMap(&company, inputMap)
	assert.NoError(t, err)
	assert.Equal(t, Company{
		Team: map[string][]Employee{
			"dev": {
				{Name: "Alice", Age: 25},
				{Name: "Bob", Age: 30},
			},
			"qa": {
				{Name: "Charlie", Age: 35},
			},
		},
	}, company)
}
