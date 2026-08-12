# copier

A zero-dependency, reflection-based deep copy library for Go. It copies values across
structs, maps, slices, arrays and pointers — in any direction:

- struct → struct
- struct → map / map → struct
- map → map (with deep copy isolation)
- slice / array / pointer deep copy

```go
go get github.com/charlienet/copier
```

## Why this library?

`copier` shares its name and core ideas with [jinzhu/copier](https://github.com/jinzhu/copier)
but diverges in several practical ways:

| Feature | copier | jinzhu/copier |
|---|---|---|
| struct ↔ map bidirectional copy | ✅ native | ❌ not supported |
| Case-insensitive field matching by default | ✅ | ❌ |
| `WithMaxDepth` recursion limit | ✅ | ❌ |
| `TypeConverter` triple match (FieldName + SrcType + DstType) | ✅ | field name only |
| Combined tag syntax `copier:"must,toname=x"` | ✅ | limited |
| Pointer cycle detection (self/mutual references) | ✅ | ❌ |
| Reflection plan cache (default struct→struct ~2.5x faster) | ✅ | ❌ |
| Method → field mapping (`WithMethodMapping`) | ✅ | ❌ |

The library is **zero-dependency** (standard library only) and safe for concurrent use.

## Quick start

### struct → struct

```go
package main

import (
	"fmt"

	"github.com/charlienet/copier"
)

type User struct {
	Name string
	Age  int
}

func main() {
	src := User{Name: "John", Age: 30}
	var dst User

	if err := copier.Copy(&dst, src); err != nil {
		panic(err)
	}
	fmt.Printf("%s is %d years old", dst.Name, dst.Age) // John is 30 years old
}
```

Field matching is **case-insensitive by default** (use `WithCaseSensitive` to opt in),
and unmatched fields are silently skipped.

### Field tags

```go
type srcUser struct {
	Name string `copier:"-"`              // always skipped
	ID   int    `copier:"must"`           // copied only when WithMust is enabled
	City string `copier:"toname=Address"` // copied to the Address field
}

type dstUser struct {
	Name    string
	ID      int
	Address string
}
```

Tags combine: `copier:"must,toname=Address"`.

### struct → map / map → struct

```go
// struct → map
src := User{Name: "John", Age: 30}
m, err := copier.ToMap(src) // map[string]any{"Name": "John", "Age": 30}

// map → struct
srcMap := map[string]any{"Name": "John", "Age": 30}
var dst User
err = copier.Copy(&dst, srcMap)
```

## Options

All options are `With*` functions passed as variadic arguments to `Copy` / `ToMap`.

| Option | Description |
|---|---|
| `WithMaxDepth(depth)` | Limit recursion depth; exceeding it returns `ErrMaxDepthExceeded`. |
| `WithIgnoreEmpty()` | Skip fields whose (converted) value is the zero value. |
| `WithCaseSensitive()` | Match field names case-sensitively. |
| `WithMust()` | Copy only fields tagged `copier:"must"`. |
| `WithConverters(...)` | Register `TypeConverter`s (FieldName + SrcType + DstType triple match). |
| `WithNameMapping(map)` | Map src field names to target names. |
| `WithNameFn(fn)` | Transform field names (applied after `toname`). |
| `WithTagName(name)` | Use a custom tag name instead of `copier`. |
| `WithSkipFields(...)` | Skip the given field names. |
| `WithValueConverter(fn)` | Transform field values per field name. |
| `WithMethodMapping()` | Enable method → field mapping (getters/setters). |

## Method → field mapping

Enable with `WithMethodMapping()`. A **setter** is invoked when the destination has no
matching field; a **getter** populates destination fields when the source has no
matching field. Method names equal the target field name.

```go
// getter: source method supplies the value
type report struct{ raw int }

func (r *report) Total() int { return r.raw * 2 } // dst field "Total"

type reportDst struct{ Total int }

// setter: destination method receives the value
type store struct{ saved string }

func (s *store) Name(v string) { s.saved = v } // src field "Name"

type storeSrc struct{ Name string }

func demo() {
	// getter: dst field "Total" has no matching src field → call src.Total()
	var d reportDst
	copier.Copy(&d, &report{raw: 21}, copier.WithMethodMapping()) // d.Total == 42

	// setter: dst has no "Name" field → call dst.Name(v)
	var st store
	copier.Copy(&st, storeSrc{Name: "x"}, copier.WithMethodMapping()) // st.saved == "x"
}
```

## Errors

All errors are sentinel values, comparable with `errors.Is`:

| Sentinel | Meaning |
|---|---|
| `ErrInvalidCopyDestination` | Destination is nil or not addressable. |
| `ErrInvalidCopyFrom` | Source is nil. |
| `ErrMapKeyNotMatch` | map→struct requires string keys. |
| `ErrNotSupported` | The type combination is not supported. |
| `ErrMaxDepthExceeded` | Recursion depth exceeded `WithMaxDepth`. |
| `ErrMethodReturnError` | A mapped getter/setter returned a non-nil error. |

```go
err := copier.Copy(nil, src)
errors.Is(err, copier.ErrInvalidCopyDestination) // true
```

## Performance

Same-type struct → struct (10 mixed-type fields), deep copy with per-iteration fresh
destination. Measured with the dedicated comparison module `copier/bench`:

```
go test -run '^$' -bench . -benchmem -count=3   # from copier/bench
```

| Library | ns/op | B/op | allocs/op |
|---|---|---|---|
| **copier** (this library) | **~1268** | 1024 | 18 |
| jinzhu/copier (shallow) | ~4609 | 928 | 30 |
| jinzhu/copier `DeepCopy:true` | ~6134 | 1984 | 52 |
| tiendc/go-deepcopy | ~610 | 712 | 9 |

The default-configuration struct→struct path uses a precomputed reflection plan cache,
which is roughly **2.5x faster** than the naive reflection field scan (measured ~2988 →
~1219 ns/op on the same workload). struct→map and map→struct are cached too.

## Testing & benchmarking

```bash
go test ./...          # unit + audit + fuzz seeds + example tests
cd copier/bench
go test -run '^$' -bench . -benchmem -count=3   # comparison vs jinzhu / tiendc
```

## Acknowledgements

Parts of the design and implementation are inspired by:

- [github.com/jinzhu/copier](https://github.com/jinzhu/copier) (MIT License): tag parsing
  model, method → field mapping concept
- [github.com/tiendc/go-deepcopy](https://github.com/tiendc/go-deepcopy) (MIT License):
  build-and-cache plan approach for reflection caching

## License

[MIT](LICENSE) — Copyright (c) 2026 charlienet
