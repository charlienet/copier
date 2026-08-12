package copier

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnyMap(t *testing.T) {
	m1 := map[string]any{
		"Name": "John",
		"Age":  "30",
		"Sex":  "Male",
	}

	var m2 map[string]any

	err := Copy(&m2, &m1)
	assert.NoError(t, err)
	assert.NotNil(t, m2)
	assert.Equal(t, m1, m2)

	t.Logf("%+v", m2)
}

func TestValueMap(t *testing.T) {
	m1 := map[string]string{
		"Name": "John",
	}

	m2 := map[string]any{}

	Copy(&m2, &m1)

	t.Logf("%+v", m2)
}

func TestKeyIntMap(t *testing.T) {
	m1 := map[int]string{
		1: "John",
	}

	m2 := map[int]any{}

	Copy(&m2, &m1)

	t.Logf("%+v", m2)
}

func TestStructToMap(t *testing.T) {
	src := Person{Name: "John", Age: 30, Address: Address{City: "Beijing"}}

	m := map[string]any{}

	assert.NoError(t, Copy(&m, &src))

	dst := map[string]any{}

	assert.NoError(t, Copy(&dst, &src, WithIgnoreEmpty()))
	assert.Equal(t, dst["Name"], "John")

	t.Logf("%+v", dst)
}

func TestEmbeddedStruct(t *testing.T) {
	type Embedded struct {
		Name string
	}

	type Person struct {
		Embedded
		Age int
	}

	src := Person{
		Embedded: Embedded{Name: "John"},
		Age:      30,
	}

	dst := map[string]any{}

	assert.NoError(t, Copy(&dst, &src))
	assert.Equal(t, dst["Name"], "John")
	assert.Equal(t, dst["Age"], 30)

	t.Logf("%+v", dst)
	t.Logf("%+v", src)
}

func TestMapToStruct(t *testing.T) {
	m := map[string]any{
		"Name": "John",
		"Age":  30,
	}

	var p Person

	assert.NoError(t, Copy(&p, &m))
	t.Logf("%+v", p)
}
