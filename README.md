# copier

A zero-dependency, reflection-based deep copy library for Go.

```bash
go get github.com/charlienet/copier
```

## Features

- **Copy in any direction**: struct ↔ struct, struct ↔ map, map ↔ map, plus slice /
  array / pointer deep copy — all with deep copy isolation (mutating the destination
  never pollutes the source).
- **Case-insensitive field matching by default**, switchable to sensitive with
  `WithCaseSensitive`.
- **Struct tags**: `copier:"-"` (skip), `copier:"must"` (only with `WithMust`),
  `copier:"toname=x"` (rename target), combinable — and the tag name itself is
  configurable via `WithTagName`.
- **Recursion depth limit** with `WithMaxDepth`.
- **TypeConverter triple match** (field name + source type + destination type) for
  custom conversions.
- **Pointer cycle detection**: self- and mutual references terminate safely without
  stack overflow.
- **Reflection plan cache**: default-configuration struct→struct copying is roughly
  2.5x faster than a naive per-field reflection scan (struct→map and map→struct are
  cached too).
- **Method → field mapping** (`WithMethodMapping`): source getters and destination
  setters participate in copying.
- **Zero third-party dependencies** (standard library only), safe for concurrent use.

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
destination:

```
~1268 ns/op · 1024 B/op · 18 allocs/op
```

The default-configuration struct→struct path uses a precomputed reflection plan cache,
which is roughly **2.5x faster** than a naive per-field reflection scan (measured ~2988 →
~1219 ns/op on the same workload). struct→map and map→struct are cached too.
Three-way comparison benchmarks against other copy libraries live in `bench/`.

## Testing & benchmarking

```bash
go test ./...          # unit + audit + fuzz seeds + example tests
cd bench
go test -run '^$' -bench . -benchmem -count=3
```

## License

[MIT](LICENSE) — Copyright (c) 2026 charlienet

Part of the design is inspired by [github.com/jinzhu/copier](https://github.com/jinzhu/copier)
(MIT License, tag parsing model and method → field mapping concept) and
[github.com/tiendc/go-deepcopy](https://github.com/tiendc/go-deepcopy) (MIT License,
build-and-cache plan approach for reflection caching).
