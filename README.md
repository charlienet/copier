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
  `CaseSensitive()`.
- **Struct tags**: `copier:"-"` (skip), `copier:"must"` (only with `MustFields()`),
  `copier:"toname=x"` (rename target), combinable — and the tag name itself is
  configurable via `TagName()`.
- **Recursion depth limit** with `MaxDepth()`.
- **TypeConverter triple match** (field name + source type + destination type) for
  custom conversions.
- **Pointer cycle detection**: self- and mutual references terminate safely without
  stack overflow.
- **Reflection plan cache**: default-configuration struct→struct copying is roughly
  2.5x faster than a naive per-field reflection scan (struct→map and map→struct are
  cached too).
- **Method → field mapping** (`MethodMapping()`): source getters and destination
  setters participate in copying.
- **Strict mode by default** (v0.2+): conversion failures surface as
  `ErrConversionFailed` errors instead of silently skipping or leaving zero values —
  opt out with `Lenient()`.
- **Chainable generic API**: `Copy` builder (fill an existing destination) and
  `Clone` (allocate a new value), with panic via `Copy(...).Must()`.
- **Zero third-party dependencies** (standard library only), safe for concurrent use.

> **⚠️ v0.2 breaking change**: copying is **strict by default** — invalid conversions
> return an error instead of silently leaving zero values. Opt out with `Lenient()`.

## API

Two main entry points cover every use case (panic via `Copy(...).Must()`):

| Entry point | Signature | Purpose |
|---|---|---|
| `Copy` | `Copy[S, D any](src S, dst *D) *Copier[S, D]` | Fill an existing destination — `src` first, matching the `json.Unmarshal` convention; chain options, then `Do() error` or `Must()`. |
| `Clone` | `Clone[T any](src T) (T, error)` | Allocate and return a new value of the same type (deep copy). |

`Copy` returns a `*Copier[S, D]` whose chain methods each return the same builder,
so options compose fluently: `Copy(src, &dst).IgnoreEmpty().MaxDepth(5).Do()`. End
the chain with `Do()` (returns an error) or `Must()` (panics on error).

**Dynamic types**: `S` and `D` may be `any`, which covers the classic untyped use
case — pass an interface value and copy into an interface destination:

```go
src := map[string]any{"Name": "John", "Age": 30}
var dst any
err := copier.Copy(src, &dst).Do() // dst is map[string]any{"Name": "John", "Age": 30}
```

On failure, seven sentinel errors are returned (see [Errors](#errors)); they are
comparable with `errors.Is`.

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

	if err := copier.Copy(src, &dst).Do(); err != nil {
		panic(err)
	}
	fmt.Printf("%s is %d years old", dst.Name, dst.Age) // John is 30 years old
}
```

Field matching is **case-insensitive by default** (use `CaseSensitive()` to opt in),
and unmatched fields are silently skipped.

Configuration chains fluently on the builder:

```go
err := copier.Copy(src, &dst).IgnoreEmpty().CaseSensitive().Do()
```

Use `Must()` when the copy cannot fail:

```go
var dto User
copier.Copy(src, &dto).Must()
```

### Field tags

```go
type srcUser struct {
	Name string `copier:"-"`              // always skipped
	ID   int    `copier:"must"`           // copied only when MustFields() is enabled
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
var m map[string]any
err := copier.Copy(src, &m).Do() // m == map[string]any{"Name": "John", "Age": 30}

// map → struct
srcMap := map[string]any{"Name": "John", "Age": 30}
var dst User
err = copier.Copy(srcMap, &dst).Do()
```

## Choosing an API style

Use `Copy` to fill an existing destination — it chains options fluently and works
cross-type, or with `S` / `D` = `any` for dynamic values. Use `Clone` to deep-copy
into a new instance of the same type, and `Copy(...).Must()` when the operation
cannot fail.

There is no longer a separate untyped `Copy(dst, src, ...)` function — the new
`Copy` takes `src` first (like `json.Unmarshal`) and covers dynamic scenarios with
`S` / `D` = `any` (see [API](#api)).

### Migration from v0.2

| v0.2 (historical) | v0.3 |
|---|---|
| `Copy(&dst, src)` | `Copy(src, &dst).Do()` — **argument order reversed**: `src` first, like `json.Unmarshal` |
| `Copy(&dst, src, WithIgnoreEmpty(), WithMaxDepth(5))` | `Copy(src, &dst).IgnoreEmpty().MaxDepth(5).Do()` |
| `CopyTo(src, &dst).Do()` | `Copy(src, &dst).Do()` |
| `Convert[Src, Dst](src)` | same-type: `Clone[T](src)`; cross-type: `var dst Dst; Copy(src, &dst).Do()` |
| `MustConvert[Src, Dst](src)` | same-type: `var dst T; Copy(src, &dst).Must()`; cross-type: `var dst Dst; Copy(src, &dst).Must()` |
| `ToMap(src)` | `var m map[string]any; Copy(src, &m).Do()` |
| `Clone[T](src)` (same-type) | `Clone[T](src)` (unchanged) |
| `WithX(...)` option functions | chain methods `.X(...)` — see [Chainable options](#chainable-options) |

## Generic API

`Clone` allocates and returns a new value; `Copy` fills an existing
destination (chainable, cross-type).

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

	// Clone: deep copy to a new instance of the same type
	clone, err := copier.Clone(src)
	if err != nil {
		panic(err)
	}
	fmt.Println(clone.Name) // John

	// Cross-type copy (e.g. DTO → domain model): fill an existing value with Copy,
	// with automatic field conversion such as int → int64
	type UserDTO struct {
		Name string
		Age  int64
	}
	var dto UserDTO
	if err := copier.Copy(src, &dto).Do(); err != nil {
		panic(err)
	}
	fmt.Println(dto.Age) // 30

	// Copy(...).Must(): panics on error — fills the destination
	dto2 := UserDTO{}
	copier.Copy(src, &dto2).Must()

	_ = dto2
}
```

## Copy Semantics

Field-kind table for `Copy` / `Clone`:

| Field kind | Behavior | Mutating dst affects src? |
|---|---|---|
| value fields (int, string, bool, ...) | value copy | No |
| slice / map (top-level) | deep copy (new allocation) | No |
| `*T` pointer fields | deep copy (auto-alloc + recursive) | No |
| pointers **inside** struct value fields | shallow copy (shared reference) | **Yes** |

The last row is the one that surprises people: a pointer nested inside a struct value
field is copied by reference, so dst and src end up sharing the same object:

```go
type Data struct{ V int }

type Leaf struct {
	D *Data // pointer inside a struct value field
}

type Tree struct {
	Leaf Leaf // Leaf is a struct value field of Tree
}

func demo() {
	src := Tree{Leaf: Leaf{D: &Data{V: 1}}}
	var dst Tree
	copier.Copy(src, &dst).Do()

	dst.Leaf.D.V = 42
	fmt.Println(src.Leaf.D.V) // 42 — dst and src share the same *Data
}
```

For a full deep copy of such a field, declare it as a pointer type instead
(`Leaf *Leaf`) — top-level `*T` fields take the auto-alloc + recursive copy path.

## Chainable options

All options are chain methods on the `*Copier[S, D]` builder. Each method returns
the same builder, so they compose in any order and take effect at `Do()` (or
`Must()`).

Semantics are unchanged from v0.2: **strict mode is on by default** (invalid
conversions return `ErrConversionFailed` instead of silently leaving zero values; opt
out with `Lenient()`), and field matching is **case-insensitive by default** (opt in
with `CaseSensitive()`).

| Method | Signature | Description |
|---|---|---|
| `Lenient()` | `*Copier[S, D]` | Opt out of strict mode: invalid conversions are skipped instead of erroring. |
| `IgnoreEmpty()` | `*Copier[S, D]` | Skip fields whose (converted) value is the zero value. |
| `CaseSensitive()` | `*Copier[S, D]` | Match field names case-sensitively. |
| `MustFields()` | `*Copier[S, D]` | Copy only fields tagged `copier:"must"`. |
| `Converters(...)` | `*Copier[S, D]` | Register `TypeConverter`s (FieldName + SrcType + DstType triple match). |
| `NameMapping(map)` | `*Copier[S, D]` | Map src field names to target names. |
| `NameFn(fn)` | `*Copier[S, D]` | Transform field names (applied after `toname`). |
| `TagName(name)` | `*Copier[S, D]` | Use a custom tag name instead of `copier`. |
| `SkipFields(...)` | `*Copier[S, D]` | Skip the given field names. |
| `ValueConverter(fn)` | `*Copier[S, D]` | Transform field values per field name. |
| `MethodMapping()` | `*Copier[S, D]` | Enable method → field mapping (getters/setters). |
| `MaxDepth(depth)` | `*Copier[S, D]` | Limit recursion depth; exceeding it returns `ErrMaxDepthExceeded`. |
| `NilSrcZero()` | `*Copier[S, D]` | Treat a nil source as a zero target instead of erroring. |
| `Do()` | `error` | Execute the copy; returns an error on failure. |
| `Must()` | — | Execute the copy; panics on error. |

## Method → field mapping

Enable with `MethodMapping()`. A **setter** is invoked when the destination has no
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
	copier.Copy(&report{raw: 21}, &d).MethodMapping().Do() // d.Total == 42

	// setter: dst has no "Name" field → call dst.Name(v)
	var st store
	copier.Copy(storeSrc{Name: "x"}, &st).MethodMapping().Do() // st.saved == "x"
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
| `ErrMaxDepthExceeded` | Recursion depth exceeded `MaxDepth()`. |
| `ErrMethodReturnError` | A mapped getter/setter returned a non-nil error. |
| `ErrConversionFailed` | A conversion failed (strict mode) or lost precision. |

```go
var m map[string]any
err := copier.Copy[any, map[string]any](nil, &m).Do()
errors.Is(err, copier.ErrInvalidCopyFrom) // true
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

## Common Pitfalls

### nil dst / nil src

A top-level nil pointer **dst** is auto-allocated (v0.2). A nil **source** errors with
`ErrInvalidCopyFrom`; pass `NilSrcZero()` to treat it as a zero target instead.

### Deep vs. shallow copy boundaries

See the [Copy Semantics](#copy-semantics) table. Pointers nested inside struct value
fields are shared between dst and src — mutating one side affects the other. When in
doubt, check the field kind against the table.

### Case-insensitive matching

Matching is case-insensitive by default, so `userID` may match `UserId`. An accidental
match is more dangerous than a missed one — a missed field simply stays zero, while an
accidental match can copy semantically different data. Use `CaseSensitive()` when
names must match exactly.

### Numeric truncation

The lenient mode silently truncates float → int (3.9 → 3). Under strict mode (default
since v0.2), precision loss returns `ErrConversionFailed` instead.

### Silent skipping

Under strict mode, parse failures and incompatible types return errors — but fields
that simply don't match (no same-name / same-tag field on the target) are still skipped
silently. This is by design: distinguish "missing field" (silent) from "conversion
failed" (error).

### `Copy` vs `Clone`

- `Copy` — fill an existing destination, with chainable options and error handling
  (`Do()` / `Must()`).
- `Clone` — same-type deep copy that returns a new instance; use `Copy` for
  cross-type copies.
- `Copy(...).Must()` — use only when you are certain the operation
  cannot fail (it panics).

## License

[MIT](LICENSE) — Copyright (c) 2026 charlienet

Part of the design is inspired by [github.com/jinzhu/copier](https://github.com/jinzhu/copier)
(MIT License, tag parsing model and method → field mapping concept) and
[github.com/tiendc/go-deepcopy](https://github.com/tiendc/go-deepcopy) (MIT License,
build-and-cache plan approach for reflection caching).
