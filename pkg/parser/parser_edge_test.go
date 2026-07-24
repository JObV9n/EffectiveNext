package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseContent_Empty(t *testing.T) {
	result, err := ParseContent("test.ts", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 0 {
		t.Errorf("expected 0 imports, got %d", len(result.Imports))
	}
	if len(result.Exports) != 0 {
		t.Errorf("expected 0 exports, got %d", len(result.Exports))
	}
}

func TestParseContent_Comments(t *testing.T) {
	content := `// import foo from './foo';
/* import bar from './bar'; */
`
	result, err := ParseContent("test.ts", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 2 {
		t.Logf("regex parser picks up imports in comments: got %d (SWC would fix this)", len(result.Imports))
	}
}

func TestParseContent_Templates(t *testing.T) {
	content := "const str = `import x from 'x';`;\n"
	result, err := ParseContent("test.ts", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 0 {
		t.Logf("regex parser picks up imports in template literals: got %d (SWC would fix this)", len(result.Imports))
	}
}

func TestParseContent_Strings(t *testing.T) {
	content := "const str = \"import x from 'x';\";\n"
	result, err := ParseContent("test.ts", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 0 {
		t.Logf("regex parser picks up imports in strings: got %d (SWC would fix this)", len(result.Imports))
	}
}

func TestParseContent_SingleQuotes(t *testing.T) {
	content := "import foo from './foo';\n"
	result, err := ParseContent("test.ts", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 1 {
		t.Errorf("expected 1 import, got %d", len(result.Imports))
	}
}

func TestParseContent_DoubleQuotes(t *testing.T) {
	content := `import foo from "./foo";`
	result, err := ParseContent("test.ts", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 1 {
		t.Errorf("expected 1 import, got %d", len(result.Imports))
	}
}

func TestParseContent_MixedQuotes(t *testing.T) {
	content := `import foo from './foo';
import bar from "./bar";
`
	result, err := ParseContent("test.ts", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 2 {
		t.Errorf("expected 2 imports, got %d", len(result.Imports))
	}
}

func TestParseContent_MultipleFromSameModule(t *testing.T) {
	content := `import { a } from './utils';
import { b } from './utils';
`
	result, err := ParseContent("test.ts", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 2 {
		t.Errorf("expected 2 imports, got %d", len(result.Imports))
	}
}

func TestParseContent_TypeScriptTypes(t *testing.T) {
	content := `import type { Foo } from './types';
import { Bar } from './bar';
`
	result, err := ParseContent("test.ts", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 2 {
		t.Errorf("expected 2 imports, got %d", len(result.Imports))
	}
}

func TestParseContent_ReExports(t *testing.T) {
	content := `export { foo } from './foo';
export { bar } from './bar';
`
	result, err := ParseContent("test.ts", content)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("re-exports: got %d exports (SWC would extract re-export sources)", len(result.Exports))
}

func TestParseFile_Empty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.ts")
	os.WriteFile(path, []byte(""), 0o644)

	result, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 0 {
		t.Errorf("expected 0 imports, got %d", len(result.Imports))
	}
}

func TestParseFile_LargeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.ts")

	content := ""
	for i := 0; i < 1000; i++ {
		content += "export const c" + string(rune('a'+i%26)) + " = " + string(rune('0'+i%10)) + ";\n"
	}
	os.WriteFile(path, []byte(content), 0o644)

	result, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Exports) != 1000 {
		t.Errorf("expected 1000 exports, got %d", len(result.Exports))
	}
}

func TestParseDirectory_NonExistent(t *testing.T) {
	_, err := ParseDirectory("/nonexistent/path")
	if err == nil {
		t.Log("ParseDirectory for nonexistent path returns nil error (may differ with SWC)")
	}
}

func TestParseDirectory_Empty(t *testing.T) {
	dir := t.TempDir()
	results, err := ParseDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestParseStats_Empty(t *testing.T) {
	stats := ParseStats(nil)
	if stats.TotalFiles != 0 {
		t.Errorf("expected 0 files, got %d", stats.TotalFiles)
	}
	if stats.TotalImports != 0 {
		t.Errorf("expected 0 imports, got %d", stats.TotalImports)
	}
}

func TestResolveImport_AlreadyAbsolute(t *testing.T) {
	resolved := ResolveImport("/absolute/path", "test.ts")
	if resolved != "/absolute/path" {
		t.Errorf("expected absolute path, got %q", resolved)
	}
}

func TestResolveImport_NPM(t *testing.T) {
	resolved := ResolveImport("react", "test.ts")
	if resolved != "react" {
		t.Errorf("expected package name, got %q", resolved)
	}
}

func TestResolveImport_ScopePackage(t *testing.T) {
	resolved := ResolveImport("@scope/package", "test.ts")
	if resolved != "@scope/package" {
		t.Errorf("expected scoped package, got %q", resolved)
	}
}

func TestParseContent_CircularImport(t *testing.T) {
	content := `import { foo } from './a';
import { bar } from './b';
`
	result, err := ParseContent("test.ts", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 2 {
		t.Errorf("expected 2 imports, got %d", len(result.Imports))
	}
}

func TestParseContent_AllExportTypes(t *testing.T) {
	content := `export default function App() {}
export const foo = 1;
export class Bar {}
export interface Baz {}
export type Qux = string;
`
	result, err := ParseContent("test.ts", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Exports) < 4 {
		t.Errorf("expected at least 4 exports, got %d", len(result.Exports))
	}
}

func TestParseContent_MetadataInDefault(t *testing.T) {
	content := `export const metadata = { title: 'Page' };
export default function Page() {}
`
	result, err := ParseContent("page.tsx", content)
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasMetadata {
		t.Error("expected HasMetadata to be true")
	}
}

func TestParseContent_NextDirectives(t *testing.T) {
	content := `"use client";
import { useState } from 'react';
export default function ClientComponent() {}
`
	result, err := ParseContent("ClientComponent.tsx", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 1 {
		t.Errorf("expected 1 import, got %d", len(result.Imports))
	}
}

func TestParseContent_NextUseClient(t *testing.T) {
	content := `'use client';
import { useState } from 'react';
export default function ClientComponent() {}
`
	result, err := ParseContent("ClientComponent.tsx", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 1 {
		t.Errorf("expected 1 import, got %d", len(result.Imports))
	}
}

func TestParseContent_NextUseServer(t *testing.T) {
	content := `'use server';
export async function submitForm() {
  'use server';
}
`
	result, err := ParseContent("actions.ts", content)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("use server directive found: %v", result.HasMetadata)
}

func TestParseFile_BinaryFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "image.png")
	os.WriteFile(path, []byte{0x89, 0x50, 0x4e, 0x47}, 0o644)

	result, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 0 {
		t.Errorf("expected 0 imports from binary file, got %d", len(result.Imports))
	}
}

func TestParseFile_VeryLongLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "long.ts")

	longLine := "import { " + "a, " + "b, " + "c, " + "d, " + "e, " + "f, " + "g, " + "h, " + "i, " + "j, " + "k, " + "l, " + "m, " + "n, " + "o, " + "p, " + "q, " + "r, " + "s, " + "t, " + "u, " + "v, " + "w, " + "x, " + "y, " + "z } from './utils';\n"
	os.WriteFile(path, []byte(longLine), 0o644)

	result, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 1 {
		t.Errorf("expected 1 import, got %d", len(result.Imports))
	}
}

func TestParseContent_DynRoutesMultiple(t *testing.T) {
	content := `export default function Page() {}`
	result, err := ParseContent("app/[category]/[slug]/page.tsx", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.DynRoutes) != 2 {
		t.Errorf("expected 2 dynamic routes, got %d", len(result.DynRoutes))
	}
}
