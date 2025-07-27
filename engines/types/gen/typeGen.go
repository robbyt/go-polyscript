package main

import (
	"bytes"
	"fmt"
	"go/format"
	"log"
	"os"
	"path/filepath"
	"text/template"
)

type EngineType struct {
	Name        string
	Value       string
	BuildTag    string
	Description string
}

func loadTemplate(filename string) string {
	// Get directory containing the generator
	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	// Load template file
	templatePath := filepath.Join(dir, "gen", "templates", filename)
	content, err := os.ReadFile(templatePath)
	if err != nil {
		log.Fatalf("Failed to read template file %s: %v", templatePath, err)
	}

	return string(content)
}

func main() {
	eTypes := []EngineType{
		{
			Name:        "Risor",
			Value:       "risor",
			BuildTag:    "risor",
			Description: "Risor engine: https://github.com/risor-io/risor",
		},
		{
			Name:        "Starlark",
			Value:       "starlark",
			BuildTag:    "starlark",
			Description: "Starlark engine: https://github.com/google/starlark-go",
		},
		{
			Name:        "Extism",
			Value:       "extism",
			BuildTag:    "extism",
			Description: "Extism WASM engine: https://extism.org/",
		},
	}

	outputTargets := []struct {
		TemplateFile string
		OutputFile   string
	}{
		{
			TemplateFile: "type.go.tmpl",
			OutputFile:   "./type.go",
		},
		{
			TemplateFile: "type_test.go.tmpl",
			OutputFile:   "./type_test.go",
		},
		{
			TemplateFile: "new.go.tmpl",
			OutputFile:   "../new.go",
		},
		{
			TemplateFile: "new_test.go.tmpl",
			OutputFile:   "../new_test.go",
		},
	}

	for _, target := range outputTargets {
		templateContent := loadTemplate(target.TemplateFile)
		t := template.Must(template.New("types").Parse(templateContent))

		var buf bytes.Buffer
		if err := t.Execute(&buf, struct{ Types []EngineType }{eTypes}); err != nil {
			log.Fatal(err)
		}

		formatted, err := format.Source(buf.Bytes())
		if err != nil {
			log.Fatal(err)
		}

		if err := os.WriteFile(target.OutputFile, formatted, 0o644); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Generated: %s\n", target.OutputFile)
	}
}
