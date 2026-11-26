package internal

import (
	"fmt"
	"math"
	"reflect"

	risor "github.com/deepnoodle-ai/risor/v2"
	"github.com/deepnoodle-ai/risor/v2/pkg/object"
)

// BuildRisorEnv builds the full Risor environment map with standard builtins and input data.
// The input data is made available under the given ctxKey (typically "ctx").
//
// For example, with ctxKey "ctx" and inputData {"foo": "bar", "baz": 123}, the
// returned map holds every standard Risor builtin plus a "ctx" entry whose value
// is inputData itself, so a script reads ctx["foo"] as "bar" and ctx["baz"] as 123.
//
// Returns an error wrapping [ErrUintOverflow] if inputData holds an unsigned
// integer above math.MaxInt64, which Risor would otherwise wrap to a negative int.
func BuildRisorEnv(ctxKey string, inputData map[string]any) (map[string]any, error) {
	if err := checkUintRange(reflect.ValueOf(inputData), ctxKey); err != nil {
		return nil, err
	}
	// Builtins() returns a fresh map on each call, so mutating env is safe.
	env := risor.Builtins()
	env[ctxKey] = inputData
	return env, nil
}

// checkUintRange walks v the way Risor's TypeRegistry converts Go values
// (maps, slices, arrays, pointers, interfaces and the direct exported fields
// of structs, which Risor wraps as GoStruct and converts when a script reads
// them) and rejects any unsigned integer that does not fit in Risor's int64.
//
// Like TypeRegistry.FromGo, it checks for values that convert themselves
// before looking at the kind: a value implementing object.RisorValuer, or one
// that already is an object.Object, is skipped along with everything it
// holds, since Risor never reads its fields or its integer value.
//
// Values returned by Go methods that a script calls are not checked: Risor
// converts them only when the method runs, after this check.
func checkUintRange(v reflect.Value, path string) error {
	return (&uintChecker{seen: make(map[visit]struct{})}).check(v, path)
}

// visit identifies a pointer or map already walked by uintChecker.
type visit struct {
	typ reflect.Type
	ptr uintptr
}

// uintChecker carries the visited set for checkUintRange, so cyclic data
// (e.g. structs with parent pointers) terminates and shared values are
// checked once.
type uintChecker struct {
	seen map[visit]struct{}
}

// markSeen records v and reports whether it had already been walked.
func (c *uintChecker) markSeen(v reflect.Value) bool {
	key := visit{typ: v.Type(), ptr: v.Pointer()}
	if _, ok := c.seen[key]; ok {
		return true
	}
	c.seen[key] = struct{}{}
	return false
}

// check rejects v if it, or any value Risor would convert from it, holds an
// unsigned integer above math.MaxInt64.
func (c *uintChecker) check(v reflect.Value, path string) error {
	if !v.IsValid() {
		return nil
	}
	if convertsItself(v.Type()) {
		return nil
	}
	switch v.Kind() {
	case reflect.Uint, reflect.Uint64, reflect.Uintptr:
		if v.Uint() > math.MaxInt64 {
			return fmt.Errorf("%w: %s = %d", ErrUintOverflow, path, v.Uint())
		}
	case reflect.Interface:
		if !v.IsNil() {
			return c.check(v.Elem(), path)
		}
	case reflect.Pointer:
		if !v.IsNil() && !c.markSeen(v) {
			return c.check(v.Elem(), path)
		}
	case reflect.Map:
		return c.checkMap(v, path)
	case reflect.Slice, reflect.Array:
		return c.checkList(v, path)
	case reflect.Struct:
		return c.checkStruct(v, path)
	}
	return nil
}

// checkMap checks the values of a string-keyed map.
func (c *uintChecker) checkMap(v reflect.Value, path string) error {
	// Risor rejects non-string keys itself.
	if v.Type().Key().Kind() != reflect.String || alwaysFits(v.Type().Elem()) ||
		v.IsNil() || c.markSeen(v) {
		return nil
	}
	iter := v.MapRange()
	for iter.Next() {
		if err := c.check(iter.Value(), fmt.Sprintf("%s[%q]", path, iter.Key().String())); err != nil {
			return err
		}
	}
	return nil
}

// checkList checks the elements of a slice or array.
func (c *uintChecker) checkList(v reflect.Value, path string) error {
	if alwaysFits(v.Type().Elem()) {
		return nil
	}
	for i := range v.Len() {
		if err := c.check(v.Index(i), fmt.Sprintf("%s[%d]", path, i)); err != nil {
			return err
		}
	}
	return nil
}

// checkStruct checks the direct exported fields of a struct, the only ones
// Risor exposes to scripts. Risor's GoStruct indexes a struct's own fields
// (NumField plus IsExported) and does not promote fields through embedding,
// so the fields of an unexported embedded struct or pointer are not exposed
// as attributes and are skipped (its promoted methods are still callable;
// see checkUintRange), while an exported embedded struct is an ordinary field
// that a script reads by its type name, and is checked like any other.
func (c *uintChecker) checkStruct(v reflect.Value, path string) error {
	t := v.Type()
	for i := range t.NumField() {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		if err := c.check(v.Field(i), path+"."+field.Name); err != nil {
			return err
		}
	}
	return nil
}

var (
	risorValuerType = reflect.TypeFor[object.RisorValuer]()
	risorObjectType = reflect.TypeFor[object.Object]()
)

// convertsItself reports whether Risor converts values of type t without
// looking at their kind: TypeRegistry.FromGo returns RisorValue() for an
// object.RisorValuer and passes an object.Object through unchanged.
func convertsItself(t reflect.Type) bool {
	return t.Implements(risorValuerType) || t.Implements(risorObjectType)
}

// alwaysFits reports whether every value of type t converts to Risor without
// overflow, so checkUintRange can skip walking a container of them.
func alwaysFits(t reflect.Type) bool {
	if convertsItself(t) {
		return true
	}
	switch t.Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Float32, reflect.Float64, reflect.Bool, reflect.String:
		return true
	default:
		return false
	}
}
