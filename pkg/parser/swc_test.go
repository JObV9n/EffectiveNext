package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSWCParser_Available(t *testing.T) {
	dir := t.TempDir()
	p := NewSWCParser(dir)
	if p.Available() {
		t.Log("SWC parser is available")
	} else {
		t.Log("SWC parser not available (using regex fallback)")
	}
}

func TestParseFileFallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ts")
	content := `import { foo } from './utils';
export default function App() {}
`
	os.WriteFile(path, []byte(content), 0o644)

	result, err := ParseFileFallback(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 1 {
		t.Errorf("expected 1 import, got %d", len(result.Imports))
	}
	if len(result.Exports) != 1 {
		t.Errorf("expected 1 export, got %d", len(result.Exports))
	}
}

func TestParseFileFallback_WithDynamic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "loader.ts")
	content := `const mod = import('./dynamic');
import x from 'x';
`
	os.WriteFile(path, []byte(content), 0o644)

	result, err := ParseFileFallback(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 2 {
		t.Fatalf("expected 2 imports, got %d", len(result.Imports))
	}
	dynamicCount := 0
	for _, imp := range result.Imports {
		if imp.Dynamic {
			dynamicCount++
		}
	}
	if dynamicCount != 1 {
		t.Errorf("expected 1 dynamic import, got %d", dynamicCount)
	}
}

func TestParseFileFallback_TSX(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "page.tsx")
	content := `import React from 'react';
export default function Page() {
	return <div>Hello</div>;
}
`
	os.WriteFile(path, []byte(content), 0o644)

	result, err := ParseFileFallback(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 1 {
		t.Errorf("expected 1 import, got %d", len(result.Imports))
	}
}

func TestParseFileFallback_JSX(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Component.jsx")
	content := `import React from 'react';
export default function Component() {
	return <div>Hello</div>;
}
`
	os.WriteFile(path, []byte(content), 0o644)

	result, err := ParseFileFallback(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 1 {
		t.Errorf("expected 1 import, got %d", len(result.Imports))
	}
}

func TestSWCParser_ParseFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ts")
	content := `import { foo } from './utils';
export default function App() {}
`
	os.WriteFile(path, []byte(content), 0o644)

	p := NewSWCParser(dir)
	if !p.Available() {
		t.Skip("SWC parser not available")
	}

	result, err := p.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.FilePath != path {
		t.Errorf("expected path %q, got %q", path, result.FilePath)
	}
}

func TestSWCResultToResult(t *testing.T) {
	swc := &SWCResult{
		FilePath: "/test.ts",
		Body: []SWCModuleItem{
			{
				Type: "ImportDeclaration",
				ImportDecl: &SWCImportDecl{
					Source: SWCStr{Value: "./utils"},
				},
			},
			{
				Type: "ExportDeclaration",
				ExportDecl: &struct {
					Span SWCSpan `json:"span"`
				}{},
			},
		},
	}

	result := swcResultToResult(swc)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Imports) != 1 {
		t.Errorf("expected 1 import, got %d", len(result.Imports))
	}
	if len(result.Exports) != 1 {
		t.Errorf("expected 1 export, got %d", len(result.Exports))
	}
	if !result.HasMetadata {
		t.Error("expected HasMetadata to be true")
	}
}
