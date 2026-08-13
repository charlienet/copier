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
  `Clone` builder (allocate a new value), executed via `Do()` / `Result()`;
  one-shot bulk config via `With(&Config{...})`.
- **Zero third-party dependencies** (standard library only).
- **Concurrency**: concurrent calls are safe; do not mutate a builder's options
  (chain methods) while another goroutine executes it.

> **⚠️ v0.2 breaking change**: copying is **strict by default** — invalid conversions
> return an error instead of silently leaving zero values. Opt out with `Lenient()`.

## API

Two main entry points cover every use case:

| Entry point | Signature | Purpose |
|---|---|---|
| `Copy` | `Copy[S, D any](src S, dst *D) *Copier[S, D]` | Fill an existing destination — `src` first, matching the `json.Unmarshal` convention; chain options, then `Do() error`. |
| `Clone` | `Clone[T any](src T) *Copier[T, T]` | Allocate a new value of the same type (deep copy), returning a builder — end with `Result() (value, error)`. |

`Copy` and `Clone` return a `*Copier[S, D]` whose chain methods each return the
same builder, so options compose fluently:
`Copy(src, &dst).IgnoreEmpty().MaxDepth(5).Do()`. End the chain with `Do()`
(returns an error) or, on a `Clone` builder, `Result()` (returns the value and an
error, zero value on failure).

**Dynamic types**: `S` and `D` may be `any`, which covers the classic untyped use
case — pass an interface value and copy into an interface destination:

```go
src := map[string]any{"Name": "John", "Age": 30}
var dst any
err := copier.Copy(src, &dst).Do() // dst is map[string]any{"Name": "John", "Age": 30}
```

`Clone[any](src).Result()` also works for dynamic values.

On failure, seven sentinel errors are returned (see [Errors](#errors)); they are
comparable with `errors.Is`.

## Fail-fast / panic

The library does **not** provide `Must()` / `ResultMust()` panic terminals. The
seven sentinel errors are all runtime *data* errors (unparsable strings, missing
fields, incompatible types — the same class as `strconv` / `json` failures), so
failures should flow through the `error` return value. Whether to fail fast is a
caller decision — opt in explicitly with an `if err != nil` guard:

```go
if err := copier.Copy(src, &dst).Do(); err != nil {
	panic(err)
}

got, err := copier.Clone(src).Result()
if err != nil {
	panic(err)
}
```

Contrast with `regexp.MustCompile`: that panics on *programming* errors
(invalid regex literal), where failing fast is always right. Data errors can
occur on valid programs, so panicking by default would be wrong.

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

Fail fast when the copy cannot fail — opt in explicitly (the library itself
never panics):

```go
var dto User
if err := copier.Copy(src, &dto).Do(); err != nil {
	panic(err)
}
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

### slice → slice

Top-level slice→slice conversion is supported (element-wise, with the same field
matching and conversion rules as struct→struct):

```go
type UserDTO struct {
	Name string
	Age  int64
}
src := []User{{Name: "John", Age: 30}}
var dst []UserDTO
err := copier.Copy(src, &dst).Do() // dst[0] == UserDTO{Name: "John", Age: 30}
```

## Choosing an API style

Use `Copy` to fill an existing destination — it chains options fluently and works
cross-type, or with `S` / `D` = `any` for dynamic values. Use `Clone` to deep-copy
into a new instance of the same type (it also returns a builder — end with
`Result()`). Errors always flow through the return value; if you want the
operation to fail fast, wrap the call in an explicit `if err != nil` guard.

There is no longer a separate untyped `Copy(dst, src, ...)` function — the new
`Copy` takes `src` first (like `json.Unmarshal`) and covers dynamic scenarios with
`S` / `D` = `any` (see [API](#api)).

### Migration from v0.2

> **Note**: `Must()` in the target column was removed in v0.4; apply the
> v0.3→v0.4 migration table below as well.

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

### Migration from v0.3 (v0.4 breaking change)

| v0.3 | v0.4 |
|---|---|
| `Clone[T](src)` → `(T, error)` | `Clone[T](src).Result()` → `(T, error)`; the builder also accepts options (`Clone(src).IgnoreEmpty().Result()`) |
| `Copy(src, &dst).Must()` | `if err := Copy(src, &dst).Do(); err != nil { panic(err) }` — panic terminals removed; fail fast is a caller decision |

## Generic API

`Clone` returns a builder that allocates and returns a new value of the same type
(`Result()`); `Copy` fills an existing destination (chainable, cross-type).

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

	// Clone: deep copy to a new instance of the same type (builder + Result terminal)
	clone, err := copier.Clone(src).Result()
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

	// Fail fast explicitly: wrap Do() in an if-err guard
	dto2 := UserDTO{}
	if err := copier.Copy(src, &dto2).Do(); err != nil {
		panic(err)
	}

	_ = dto2
}
```

## One-shot config with `With(&Config{...})`

Instead of chaining individual methods one by one, apply several options at once
with `With` and a `Config` — **zero-value fields keep the current/default
settings; only non-zero fields are applied**:

```go
err := copier.Copy(src, &dst).With(&copier.Config{IgnoreEmpty: true}).Do()
```

On a `Clone` builder the same works, ending with the `Result()` terminal:

```go
got, err := copier.Clone(src).With(&copier.Config{MaxDepth: 3, NilSrcZero: true}).Result()
```

`nil` config is a no-op. `With` composes with chain methods in any order (later
settings win at execution time). Field rules: strings apply when non-empty,
`MaxDepth` when non-zero, bools when `true` (e.g. `Lenient: true` opts out of
strict mode), and slices / maps / funcs when non-nil. `Lenient()` once enabled
cannot be turned off again via `Config` or chain methods.

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
`Result()`).

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
| `With(cfg)` | `*Copier[S, D]` | Apply a one-shot `Config` (non-zero fields only; `nil` is a no-op) — see [One-shot config](#one-shot-config-with-withconfig). |
| `Do()` | `error` | Execute the copy; returns an error on failure. |
| `Result()` | `(D, error)` | Execute and return the allocated value plus an error (zero value on failure); harmless on a `Copy` builder. |

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

> **Note**: A `TypeConverter.Fn` returning an error or nil is treated as
> "not converted" (silently skipped) and does not produce an error, even in
> strict mode.

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
failed" (error). Skipped fields leave the destination field untouched (its previous
value is preserved).

### `Copy` vs `Clone`

- `Copy` — fill an existing destination, with chainable options and error handling
  (`Do()`).
- `Clone` — same-type deep copy that allocates a new instance via a builder; end with
  `Result()` (value, error); use `Copy` for cross-type copies.
- Fail fast explicitly when you are certain the operation cannot fail — see
  [Fail-fast / panic](#fail-fast--panic).

## License

[MIT](LICENSE) — Copyright (c) 2026 charlienet

Part of the design is inspired by [github.com/jinzhu/copier](https://github.com/jinzhu/copier)
(MIT License, tag parsing model and method → field mapping concept) and
[github.com/tiendc/go-deepcopy](https://github.com/tiendc/go-deepcopy) (MIT License,
build-and-cache plan approach for reflection caching).
