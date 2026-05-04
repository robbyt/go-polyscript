package internal

import (
	"encoding/json"
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFixJSONNumberTypes(t *testing.T) {
	t.Parallel()

	t.Run("handles nil input", func(t *testing.T) {
		result := FixJSONNumberTypes(nil)
		assert.Nil(t, result)
	})

	t.Run("handles primitive types", func(t *testing.T) {
		assert.Equal(t, "test string", FixJSONNumberTypes("test string"))
		assert.Equal(t, true, FixJSONNumberTypes(true))
		assert.Equal(t, 42, FixJSONNumberTypes(42))
	})

	t.Run("converts integer numbers to int regardless of field name", func(t *testing.T) {
		data := map[string]any{
			"item_count": json.Number("42"),
			"count":      json.Number("100"),
			"user_id":    json.Number("123"),
			"productId":  json.Number("456"),
			"quantity":   json.Number("7"),
			"status":     json.Number("0"),
		}

		result := FixJSONNumberTypes(data)
		mapResult, ok := result.(map[string]any)
		assert.True(t, ok)

		for k, v := range mapResult {
			assert.IsType(t, int(0), v, "field %q should be int", k)
		}
		assert.Equal(t, 42, mapResult["item_count"])
		assert.Equal(t, 100, mapResult["count"])
		assert.Equal(t, 123, mapResult["user_id"])
		assert.Equal(t, 456, mapResult["productId"])
		assert.Equal(t, 7, mapResult["quantity"])
		assert.Equal(t, 0, mapResult["status"])
	})

	t.Run("converts decimal numbers to float64", func(t *testing.T) {
		data := map[string]any{
			"price":      json.Number("19.99"),
			"rating":     json.Number("4.5"),
			"percentage": json.Number("75.5"),
		}

		result := FixJSONNumberTypes(data)
		mapResult, ok := result.(map[string]any)
		assert.True(t, ok)

		assert.InDelta(t, 19.99, mapResult["price"], 0.0001)
		assert.InDelta(t, 4.5, mapResult["rating"], 0.0001)
		assert.InDelta(t, 75.5, mapResult["percentage"], 0.0001)
		assert.IsType(t, float64(0), mapResult["price"])
		assert.IsType(t, float64(0), mapResult["rating"])
		assert.IsType(t, float64(0), mapResult["percentage"])
	})

	t.Run("handles nested maps", func(t *testing.T) {
		data := map[string]any{
			"user": map[string]any{
				"user_id": json.Number("123"),
				"stats": map[string]any{
					"login_count": json.Number("42"),
					"score":       json.Number("95.5"),
				},
			},
			"item_count": json.Number("10"),
		}

		result := FixJSONNumberTypes(data)
		mapResult, ok := result.(map[string]any)
		assert.True(t, ok)

		assert.Equal(t, 10, mapResult["item_count"])

		user, ok := mapResult["user"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, 123, user["user_id"])

		stats, ok := user["stats"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, 42, stats["login_count"])
		assert.InDelta(t, 95.5, stats["score"], 0.0001)
	})

	t.Run("converts numbers in slices", func(t *testing.T) {
		data := []any{
			json.Number("1"),
			json.Number("2.5"),
			"test",
			map[string]any{
				"item_id": json.Number("123"),
				"price":   json.Number("9.99"),
			},
		}

		result := FixJSONNumberTypes(data)
		sliceResult, ok := result.([]any)
		assert.True(t, ok)

		// Numbers in slices are now converted
		assert.Equal(t, 1, sliceResult[0])
		assert.IsType(t, int(0), sliceResult[0])
		assert.InDelta(t, 2.5, sliceResult[1], 0.0001)
		assert.IsType(t, float64(0), sliceResult[1])
		assert.Equal(t, "test", sliceResult[2])

		// Nested map in slice
		itemMap, ok := sliceResult[3].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, 123, itemMap["item_id"])
		assert.InDelta(t, 9.99, itemMap["price"], 0.0001)
	})

	t.Run("handles nested slices in maps", func(t *testing.T) {
		data := map[string]any{
			"products": []any{
				map[string]any{
					"product_id": json.Number("1"),
					"price":      json.Number("19.99"),
				},
				map[string]any{
					"product_id": json.Number("2"),
					"price":      json.Number("29.99"),
				},
			},
			"total_count": json.Number("2"),
		}

		result := FixJSONNumberTypes(data)
		mapResult, ok := result.(map[string]any)
		assert.True(t, ok)

		assert.Equal(t, 2, mapResult["total_count"])

		products, ok := mapResult["products"].([]any)
		assert.True(t, ok)
		assert.Len(t, products, 2)

		product1, ok := products[0].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, 1, product1["product_id"])
		assert.InDelta(t, 19.99, product1["price"], 0.0001)

		product2, ok := products[1].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, 2, product2["product_id"])
		assert.InDelta(t, 29.99, product2["price"], 0.0001)
	})

	t.Run("handles invalid numbers gracefully", func(t *testing.T) {
		data := map[string]any{
			"bad_int":   json.Number("not-a-number"),
			"bad_float": json.Number("also-not-a-number"),
		}

		result := FixJSONNumberTypes(data)
		mapResult, ok := result.(map[string]any)
		assert.True(t, ok)

		// Invalid numbers remain as json.Number
		assert.IsType(t, json.Number(""), mapResult["bad_int"])
		assert.IsType(t, json.Number(""), mapResult["bad_float"])
	})

	t.Run("handles maximum int64 value", func(t *testing.T) {
		data := map[string]any{
			"big": json.Number(strconv.FormatInt(math.MaxInt64, 10)),
		}

		result := FixJSONNumberTypes(data)
		mapResult, ok := result.(map[string]any)
		assert.True(t, ok)

		// On 64-bit platforms int spans the full int64 range, so we keep int.
		// On 32-bit, int64 max overflows int and we fall back to int64.
		if strconv.IntSize == 64 {
			assert.IsType(t, int(0), mapResult["big"])
			assert.Equal(t, math.MaxInt, mapResult["big"])
		} else {
			assert.IsType(t, int64(0), mapResult["big"])
			assert.Equal(t, int64(math.MaxInt64), mapResult["big"])
		}
	})

	t.Run("number too large for int64 falls back to float64", func(t *testing.T) {
		data := map[string]any{
			"huge": json.Number("99999999999999999999"), // exceeds int64
		}

		result := FixJSONNumberTypes(data)
		mapResult, ok := result.(map[string]any)
		assert.True(t, ok)
		assert.IsType(t, float64(0), mapResult["huge"])
	})

	t.Run("handles mixed types in map", func(t *testing.T) {
		data := map[string]any{
			"name":    "test",
			"active":  true,
			"count":   json.Number("5"),
			"rate":    json.Number("3.14"),
			"tags":    []any{"a", "b"},
			"nested":  map[string]any{"x": json.Number("1")},
			"nothing": nil,
		}

		result := FixJSONNumberTypes(data)
		mapResult, ok := result.(map[string]any)
		assert.True(t, ok)

		assert.Equal(t, "test", mapResult["name"])
		assert.Equal(t, true, mapResult["active"])
		assert.Equal(t, 5, mapResult["count"])
		assert.InDelta(t, 3.14, mapResult["rate"], 0.0001)
		assert.Equal(t, []any{"a", "b"}, mapResult["tags"])
		nested, ok := mapResult["nested"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, 1, nested["x"])
		assert.Nil(t, mapResult["nothing"])
	})
}

func TestConvertJSONNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    json.Number
		expected any
	}{
		{"integer", json.Number("42"), int(42)},
		{"zero", json.Number("0"), int(0)},
		{"negative integer", json.Number("-10"), int(-10)},
		{"float", json.Number("3.14"), float64(3.14)},
		{"negative float", json.Number("-2.5"), float64(-2.5)},
		{"invalid", json.Number("xyz"), json.Number("xyz")},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result := convertJSONNumber(tc.input)
			if f, ok := tc.expected.(float64); ok {
				assert.InDelta(t, f, result, 0.0001)
			} else {
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}
