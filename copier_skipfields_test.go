package copier

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSkipFields(t *testing.T) {
	type Person struct {
		Name string
		Age  int
		Sex  string
	}

	src := Person{Name: "John", Age: 30, Sex: "Male"}
	var dst Person

	err := Copy(src, &dst).SkipFields("Sex").Do()
	assert.NoError(t, err)
	assert.Equal(t, "John", dst.Name)
	assert.Equal(t, 30, dst.Age)
	assert.Equal(t, "", dst.Sex) // Sex 应该被跳过
}

func TestSkipFieldsMultiple(t *testing.T) {
	type Person struct {
		Name    string
		Age     int
		Sex     string
		Address string
	}

	src := Person{Name: "John", Age: 30, Sex: "Male", Address: "Beijing"}
	var dst Person

	err := Copy(src, &dst).SkipFields("Sex", "Address").Do()
	assert.NoError(t, err)
	assert.Equal(t, "John", dst.Name)
	assert.Equal(t, 30, dst.Age)
	assert.Equal(t, "", dst.Sex)
	assert.Equal(t, "", dst.Address)
}

func TestValueConverter(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	src := Person{Name: "john", Age: 30}
	var dst Person

	// 将名字转换为大写
	err := Copy(src, &dst).ValueConverter(func(fieldName string, value any) any {
		if fieldName == "Name" {
			if s, ok := value.(string); ok {
				return "Mr. " + s
			}
		}
		return value
	}).Do()

	assert.NoError(t, err)
	assert.Equal(t, "Mr. john", dst.Name)
	assert.Equal(t, 30, dst.Age)
}

func TestValueConverter_StructToMap(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	src := Person{Name: "John", Age: 30}
	dst := make(map[string]any)

	err := Copy(src, &dst).ValueConverter(func(fieldName string, value any) any {
		if fieldName == "Age" {
			if age, ok := value.(int); ok {
				return age + 1 // 年龄加1
			}
		}
		return value
	}).Do()

	assert.NoError(t, err)
	assert.Equal(t, "John", dst["Name"])
	assert.Equal(t, 31, dst["Age"]) // 应该是31
}

func TestSkipFields_StructToMap(t *testing.T) {
	type Person struct {
		Name    string
		Age     int
		Sex     string
		Address string
	}

	src := Person{Name: "John", Age: 30, Sex: "Male", Address: "Beijing"}
	dst := make(map[string]any)

	err := Copy(src, &dst).SkipFields("Sex", "Address").Do()
	assert.NoError(t, err)
	assert.Equal(t, "John", dst["Name"])
	assert.Equal(t, 30, dst["Age"])
	_, hasSex := dst["Sex"]
	_, hasAddress := dst["Address"]
	assert.False(t, hasSex)
	assert.False(t, hasAddress)
}

func TestCombinedOptions(t *testing.T) {
	type Person struct {
		Name    string
		Age     int
		Sex     string
		Address string
	}

	src := Person{Name: "john", Age: 30, Sex: "Male", Address: "Beijing"}
	var dst Person

	err := Copy(src, &dst).SkipFields("Address").ValueConverter(func(fieldName string, value any) any {
		if fieldName == "Name" {
			if s, ok := value.(string); ok {
				return "Mr. " + s
			}
		}
		return value
	}).Do()

	assert.NoError(t, err)
	assert.Equal(t, "Mr. john", dst.Name)
	assert.Equal(t, 30, dst.Age)
	assert.Equal(t, "Male", dst.Sex)
	assert.Equal(t, "", dst.Address) // 被跳过
}
