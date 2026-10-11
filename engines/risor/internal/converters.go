package internal

import (
	"fmt"
	"math"
	"reflect"

	risor "github.com/deepnoodle-ai/risor/v2"
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
// (maps, slices, arrays, pointers, interfaces and the exported fields of
// structs, which Risor wraps as GoStruct and converts when a script reads
// them) and rejects any unsigned integer that does not fit in Risor's int64.
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

func (c *uintChecker) check(v reflect.Value, path string) error {
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
	case reflect.Slice, reflect.Array:
		if alwaysFits(v.Type().Elem()) {
			return nil
		}
		for i := range v.Len() {
			if err := c.check(v.Index(i), fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	case reflect.Struct:
		// Risor only exposes exported fields to scripts.
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
	}
	return nil
}

// alwaysFits reports whether every value of type t converts to Risor without
// overflow, so checkUintRange can skip walking a container of them.
func alwaysFits(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Float32, reflect.Float64, reflect.Bool, reflect.String:
		return true
	default:
		return false
	}
}
