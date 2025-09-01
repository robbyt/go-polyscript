package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/extism/go-pdk"
)

// Request represents a complex input object
type Request struct {
	ID        string            `json:"id"`
	Timestamp int64             `json:"timestamp"`
	Data      map[string]any    `json:"data"`
	Tags      []string          `json:"tags"`
	Metadata  map[string]string `json:"metadata"`
	Count     int               `json:"count"`
	Active    bool              `json:"active"`
}

// Response represents a complex output object
type Response struct {
	RequestID   string         `json:"request_id"`
	ProcessedAt string         `json:"processed_at"`
	Results     map[string]any `json:"results"`
	TagCount    int            `json:"tag_count"`
	MetaCount   int            `json:"meta_count"`
	IsActive    bool           `json:"is_active"`
	Summary     string         `json:"summary"`
}

// VowelCount represents a vowel counting result
type VowelCount struct {
	Count  int    `json:"count"`
	Vowels string `json:"vowels"`
	Input  string `json:"input"`
}

//go:wasmexport greet
func greet() int32 {
	// Update to use InputJSON
	var input struct {
		Input string `json:"input"`
	}

	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return 1
	}

	if input.Input == "" {
		pdk.SetError(errors.New("input string is empty"))
		return 1
	}

	result := doGreet(input.Input)

	if err := pdk.OutputJSON(result); err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}

//go:wasmexport process_complex
func processComplex() int32 {
	var req Request
	if err := pdk.InputJSON(&req); err != nil {
		pdk.SetError(err)
		return 1
	}

	// Process the complex input
	resp := Response{
		RequestID:   req.ID,
		ProcessedAt: time.Now().UTC().Format(time.RFC3339),
		Results:     make(map[string]any),
		TagCount:    len(req.Tags),
		MetaCount:   len(req.Metadata),
		IsActive:    req.Active,
	}

	// Do some processing on the data
	resp.Results["uppercase_tags"] = upperStrings(req.Tags)
	resp.Results["data_count"] = len(req.Data)
	resp.Results["original_count"] = req.Count

	// Create a summary
	resp.Summary = createSummary(req)

	// Output the response
	if err := pdk.OutputJSON(resp); err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}

//go:wasmexport count_vowels
func countVowels() int32 {
	// Update to use InputJSON
	var input struct {
		Input string `json:"input"`
	}

	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return 1
	}

	if input.Input == "" {
		pdk.SetError(errors.New("input string is empty"))
		return 1
	}

	result := doCountVowels(input.Input)

	if err := pdk.OutputJSON(result); err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}

//go:wasmexport reverse_string
func reverseString() int32 {
	// Update to use InputJSON
	var input struct {
		Input string `json:"input"`
	}

	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return 1
	}

	if input.Input == "" {
		pdk.SetError(errors.New("input string is empty"))
		return 1
	}

	result := doReverseString(input.Input)

	if err := pdk.OutputJSON(result); err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}

// Helper functions
func upperStrings(s []string) []string {
	result := make([]string, len(s))
	for i, str := range s {
		result[i] = strings.ToUpper(str)
	}
	return result
}

func createSummary(req Request) string {
	return fmt.Sprintf(
		"Processed request %s with %d tags and %d metadata entries. Count: %d, Active: %v",
		req.ID,
		len(req.Tags),
		len(req.Metadata),
		req.Count,
		req.Active,
	)
}

// Core business logic helpers for string processing functions
func doGreet(input string) map[string]string {
	greeting := "Hello, " + input + "!"
	return map[string]string{
		"greeting": greeting,
	}
}

func doCountVowels(input string) VowelCount {
	vowels := "aeiouAEIOU"
	count := 0
	for _, c := range input {
		if strings.ContainsRune(vowels, c) {
			count++
		}
	}

	return VowelCount{
		Count:  count,
		Vowels: vowels,
		Input:  input,
	}
}

func doReverseString(input string) map[string]string {
	// Convert to runes to handle UTF-8 correctly
	runes := []rune(input)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	return map[string]string{
		"reversed": string(runes),
	}
}

//go:wasmexport run
func run() int32 {
	return greet()
}

// Namespaced Functions - These functions accept input under a "data" namespace
// This provides compatibility for applications that use structured data organization

//go:wasmexport greet_namespaced
func greetNamespaced() int32 {
	// Accepts namespaced input structure: {"data": {"input": "..."}, "request": {...}}
	var input struct {
		Data struct {
			Input string `json:"input"`
		} `json:"data"`
		Request map[string]any `json:"request,omitempty"`
	}

	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return 1
	}

	if input.Data.Input == "" {
		pdk.SetError(errors.New("input string is empty"))
		return 1
	}

	result := doGreet(input.Data.Input)

	if err := pdk.OutputJSON(result); err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}

//go:wasmexport count_vowels_namespaced
func countVowelsNamespaced() int32 {
	// Accepts namespaced input structure: {"data": {"input": "..."}, "request": {...}}
	var input struct {
		Data struct {
			Input string `json:"input"`
		} `json:"data"`
		Request map[string]any `json:"request,omitempty"`
	}

	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return 1
	}

	if input.Data.Input == "" {
		pdk.SetError(errors.New("input string is empty"))
		return 1
	}

	result := doCountVowels(input.Data.Input)

	if err := pdk.OutputJSON(result); err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}

//go:wasmexport reverse_string_namespaced
func reverseStringNamespaced() int32 {
	// Accepts namespaced input structure: {"data": {"input": "..."}, "request": {...}}
	var input struct {
		Data struct {
			Input string `json:"input"`
		} `json:"data"`
		Request map[string]any `json:"request,omitempty"`
	}

	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return 1
	}

	if input.Data.Input == "" {
		pdk.SetError(errors.New("input string is empty"))
		return 1
	}

	result := doReverseString(input.Data.Input)

	if err := pdk.OutputJSON(result); err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}
