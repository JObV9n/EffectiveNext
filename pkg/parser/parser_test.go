package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseContent_StaticImports(t *testing.T) {
	content := `import { foo } from './utils';
import React from 'react';
import './styles.css';
`
	result, err := ParseContent("test.ts", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 3 {
		t.Fatalf("expected 3 imports, got %d", len(result.Imports))
	}
	for _, imp := range result.Imports {
		if imp.Dynamic {
			t.Error("expected static import")
		}
	}
}

func TestParseContent_DynamicImports(t *testing.T) {
	content := `const mod = import('./lazy');
const other = import("dynamic");
`
	result, err := ParseContent("test.ts", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 2 {
		t.Fatalf("expected 2 dynamic imports, got %d", len(result.Imports))
	}
	for _, imp := range result.Imports {
		if !imp.Dynamic {
			t.Error("expected dynamic import")
		}
	}
}

func TestParseContent_Exports(t *testing.T) {
	content := `export default function App() {}
export const util = {};
export class Foo {}
`
	result, err := ParseContent("test.ts", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Exports) != 3 {
		t.Fatalf("expected 3 exports, got %d", len(result.Exports))
	}
}

func TestParseContent_Metadata(t *testing.T) {
	content := `export const metadata = { title: 'Test' };`
	result, err := ParseContent("page.tsx", content)
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasMetadata {
		t.Error("expected HasMetadata to be true")
	}
}

func TestParseContent_NoMetadata(t *testing.T) {
	content := `export default function Page() {}`
	result, err := ParseContent("page.tsx", content)
	if err != nil {
		t.Fatal(err)
	}
	if result.HasMetadata {
		t.Error("expected HasMetadata to be false")
	}
}

func TestParseContent_Require(t *testing.T) {
	content := `const pkg = require('some-package');`
	result, err := ParseContent("test.js", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(result.Imports))
	}
	if !result.Imports[0].Dynamic {
		t.Error("expected require to be treated as dynamic")
	}
}

func TestParseContent_IgnoresHTTP(t *testing.T) {
	content := `import foo from 'https://example.com/foo';
import bar from 'node:fs';`
	result, err := ParseContent("test.ts", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 0 {
		t.Errorf("expected 0 imports, got %d", len(result.Imports))
	}
}

func TestParseFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ts")
	os.WriteFile(path, []byte(`import x from './x';`), 0o644)

	result, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(result.Imports))
	}
}

func TestParseDirectory(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.ts"), []byte(`import b from './b';`), 0o644)
	os.WriteFile(filepath.Join(dir, "b.ts"), []byte(`export const b = 1;`), 0o644)
	os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755)
	os.WriteFile(filepath.Join(dir, "node_modules", "pkg.ts"), []byte(`import x from 'x';`), 0o644)

	results, err := ParseDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results (excluding node_modules), got %d", len(results))
	}
}

func TestParseStats(t *testing.T) {
	results := []*Result{
		{Imports: []Import{{Path: "a"}, {Path: "b", Dynamic: true}}, Exports: []Export{{Name: "foo"}}},
		{Imports: []Import{{Path: "c"}}, Exports: []Export{{Name: "bar"}, {Name: "baz"}}},
	}
	stats := ParseStats(results)
	if stats.TotalFiles != 2 {
		t.Errorf("TotalFiles = %d, want 2", stats.TotalFiles)
	}
	if stats.TotalImports != 3 {
		t.Errorf("TotalImports = %d, want 3", stats.TotalImports)
	}
	if stats.StaticImports != 2 {
		t.Errorf("StaticImports = %d, want 2", stats.StaticImports)
	}
	if stats.DynamicImports != 1 {
		t.Errorf("DynamicImports = %d, want 1", stats.DynamicImports)
	}
	if stats.TotalExports != 3 {
		t.Errorf("TotalExports = %d, want 3", stats.TotalExports)
	}
}

func TestResolveImport_Relative(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "utils.ts"), []byte(`export {}`), 0o644)

	resolved := ResolveImport("./utils", filepath.Join(dir, "index.ts"))
	if resolved != filepath.Join(dir, "utils.ts") {
		t.Errorf("ResolveImport = %q, want %q", resolved, filepath.Join(dir, "utils.ts"))
	}
}

func TestResolveImport_Package(t *testing.T) {
	resolved := ResolveImport("react", "test.ts")
	if resolved != "react" {
		t.Errorf("ResolveImport should return package name as-is, got %q", resolved)
	}
}

func TestParseContent_DynRoutes(t *testing.T) {
	content := `export default function Page() {}`
	result, err := ParseContent("app/blog/[slug]/page.tsx", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.DynRoutes) != 1 || result.DynRoutes[0] != "slug" {
		t.Errorf("expected dyn route [slug], got %v", result.DynRoutes)
	}
}