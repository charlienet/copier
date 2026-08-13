package copier

import "fmt"

type exampleUser struct {
	Name string
	Age  int
}

func ExampleCopy() {
	src := exampleUser{Name: "John", Age: 30}
	var dst exampleUser
	if err := Copy(src, &dst).Do(); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%s is %d years old", dst.Name, dst.Age)
	// Output: John is 30 years old
}

func ExampleCopy_map() {
	src := map[string]any{"Name": "John", "Age": 30}
	type user struct {
		Name string
		Age  int
	}
	var dst user
	if err := Copy(src, &dst).Do(); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%s is %d years old", dst.Name, dst.Age)
	// Output: John is 30 years old
}

func ExampleCopy_options() {
	src := map[string]any{"Name": "John", "Age": 30, "Secret": "x"}
	type user struct {
		Name   string
		Age    int
		Secret string
	}
	var dst user
	if err := Copy(src, &dst).SkipFields("Secret").Do(); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("secret=%q", dst.Secret)
	// Output: secret=""
}

func ExampleCopy_structToMap() {
	src := struct {
		Name string
		Age  int
	}{Name: "John", Age: 30}

	// struct → map（跨类型走 Copy）
	var m map[string]any
	if err := Copy(src, &m).Do(); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("age=%d", m["Age"].(int))
	// Output: age=30
}

func ExampleClone() {
	src := exampleUser{Name: "John", Age: 30}

	// 同型深拷贝：Result() 返回新值 + error
	got, err := Clone(src).Result()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%s is %d years old", got.Name, got.Age)
	// Output: John is 30 years old
}

func ExampleClone_options() {
	type user struct {
		Name string
		Meta map[string]string // 零值（nil map）：IgnoreEmpty 跳过，保持 nil
	}
	src := user{Name: "John"}

	// Clone 返回 builder 可链式带选项，终端用 Result()
	got, err := Clone(src).IgnoreEmpty().Result()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("meta_is_nil=%v", got.Meta == nil)
	// Output: meta_is_nil=true
}

func ExampleCopy_withConfig() {
	src := map[string]any{"Name": "John", "Age": 30, "Secret": "x"}
	type user struct {
		Name   string
		Age    int
		Secret string
	}
	var dst user

	// With(&Config{...}) 一次性应用非零字段配置
	if err := Copy(src, &dst).With(&Config{SkipFields: []string{"Secret"}}).Do(); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("secret=%q", dst.Secret)
	// Output: secret=""
}
