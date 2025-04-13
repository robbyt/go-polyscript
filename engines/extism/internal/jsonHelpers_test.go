package internal

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFixJSONNumberTypes(t *testing.T) {
	t.Run("handles nil input", func(t *testing.T) {
		result := FixJSONNumberTypes(nil)
		assert.Nil(t, result)
	})

	t.Run("handles primitive types", func(t *testing.T) {
		// Test string
		str := "test string"
		result := FixJSONNumberTypes(str)
		assert.Equal(t, str, result)

		// Test bool
		boolVal := true
		result = FixJSONNumberTypes(boolVal)
		assert.Equal(t, boolVal, result)
	})

	t.Run("converts count fields to integers", func(t *testing.T) {
		// Create a map with json.Number values
		data := map[string]any{
			"item_count": json.Number("42"),
			"count":      json.Number("100"),
		}

		result := FixJSONNumberTypes(data)
		mapResult, ok := result.(map[string]any)
		assert.True(t, ok, "Result should be a map")

		// Check that the values were converted to integers
		assert.Equal(t, int(42), mapResult["item_count"])
		assert.Equal(t, int(100), mapResult["count"])
		assert.IsType(t, int(0), mapResult["item_count"])
		assert.IsType(t, int(0), mapResult["count"])
	})

	t.Run("converts ID fields to integers", func(t *testing.T) {
		// Create a map with json.Number values
		data := map[string]any{
			"user_id":     json.Number("123"),
			"productId":   json.Number("456"),
			"category_id": json.Number("789"),
		}

		result := FixJSONNumberTypes(data)
		mapResult, ok := result.(map[string]any)
		assert.True(t, ok, "Result should be a map")

		// Check that the values were converted to integers
		assert.Equal(t, int(123), mapResult["user_id"])
		assert.Equal(t, int(456), mapResult["productId"])
		assert.Equal(t, int(789), mapResult["category_id"])
		assert.IsType(t, int(0), mapResult["user_id"])
		assert.IsType(t, int(0), mapResult["productId"])
		assert.IsType(t, int(0), mapResult["category_id"])
	})

	t.Run("converts other number fields to float64", func(t *testing.T) {
		// Create a map with json.Number values for non-integer fields
		data := map[string]any{
			"price":      json.Number("19.99"),
			"rating":     json.Number("4.5"),
			"percentage": json.Number("75.5"),
		}

		result := FixJSONNumberTypes(data)
		mapResult, ok := result.(map[string]any)
		assert.True(t, ok, "Result should be a map")

		// Check that the values were converted to float64
		assert.Equal(t, float64(19.99), mapResult["price"])
		assert.Equal(t, float64(4.5), mapResult["rating"])
		assert.Equal(t, float64(75.5), mapResult["percentage"])
		assert.IsType(t, float64(0), mapResult["price"])
		assert.IsType(t, float64(0), mapResult["rating"])
		assert.IsType(t, float64(0), mapResult["percentage"])
	})

	t.Run("handles nested maps", func(t *testing.T) {
		// Create a map with nested maps
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
		assert.True(t, ok, "Result should be a map")

		// Check top level integer conversion
		assert.Equal(t, int(10), mapResult["item_count"])

		// Check nested map conversions
		user, ok := mapResult["user"].(map[string]any)
		assert.True(t, ok, "user should be a map")
		assert.Equal(t, int(123), user["user_id"])

		stats, ok := user["stats"].(map[string]any)
		assert.True(t, ok, "stats should be a map")
		assert.Equal(t, int(42), stats["login_count"])
		assert.Equal(t, float64(95.5), stats["score"])
	})

	t.Run("handles slices", func(t *testing.T) {
		// Create a slice with json.Number values
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
		assert.True(t, ok, "Result should be a slice")

		// Values in the slice should remain as json.Number
		// since they don't have field names to determine type
		assert.IsType(t, json.Number(""), sliceResult[0])
		assert.IsType(t, json.Number(""), sliceResult[1])
		assert.Equal(t, "test", sliceResult[2])

		// Check the map in the slice
		itemMap, ok := sliceResult[3].(map[string]any)
		assert.True(t, ok, "Item at index 3 should be a map")
		assert.Equal(t, int(123), itemMap["item_id"])
		assert.Equal(t, float64(9.99), itemMap["price"])
	})

	t.Run("handles nested slices", func(t *testing.T) {
		// Create a map with nested slices
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
		assert.True(t, ok, "Result should be a map")

		// Check top level count conversion
		assert.Equal(t, int(2), mapResult["total_count"])

		// Check nested slice
		products, ok := mapResult["products"].([]any)
		assert.True(t, ok, "products should be a slice")
		assert.Len(t, products, 2)

		// Check first product
		product1, ok := products[0].(map[string]any)
		assert.True(t, ok, "First product should be a map")
		assert.Equal(t, int(1), product1["product_id"])
		assert.Equal(t, float64(19.99), product1["price"])

		// Check second product
		product2, ok := products[1].(map[string]any)
		assert.True(t, ok, "Second product should be a map")
		assert.Equal(t, int(2), product2["product_id"])
		assert.Equal(t, float64(29.99), product2["price"])
	})

	t.Run("handles invalid numbers gracefully", func(t *testing.T) {
		// Create a map with invalid json.Number
		data := map[string]any{
			"item_count": json.Number("not-a-number"),
			"price":      json.Number("also-not-a-number"),
		}

		result := FixJSONNumberTypes(data)
		mapResult, ok := result.(map[string]any)
		assert.True(t, ok, "Result should be a map")

		// Values should remain as json.Number since conversion failed
		assert.IsType(t, json.Number(""), mapResult["item_count"])
		assert.IsType(t, json.Number(""), mapResult["price"])
	})
}
