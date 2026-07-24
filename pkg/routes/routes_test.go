package routes

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create App Router structure
	appDir := filepath.Join(dir, "app")
	os.MkdirAll(filepath.Join(appDir, "(marketing)"), 0o755)
	os.WriteFile(filepath.Join(appDir, "layout.tsx"), []byte(`export default function RootLayout() {}`), 0o644)
	os.WriteFile(filepath.Join(appDir, "page.tsx"), []byte(`export default function Home() {}`), 0o644)
	os.WriteFile(filepath.Join(appDir, "loading.tsx"), []byte(`export default function Loading() {}`), 0o644)
	os.WriteFile(filepath.Join(appDir, "error.tsx"), []byte(`export default function Error() {}`), 0o644)
	os.WriteFile(filepath.Join(appDir, "not-found.tsx"), []byte(`export default function NotFound() {}`), 0o644)

	os.WriteFile(filepath.Join(appDir, "(marketing)", "layout.tsx"), []byte(`export default function MarketingLayout() {}`), 0o644)
	os.WriteFile(filepath.Join(appDir, "(marketing)", "about", "page.tsx"), []byte(`export default function About() {}`), 0o644)

	os.MkdirAll(filepath.Join(appDir, "blog", "[slug]"), 0o755)
	os.WriteFile(filepath.Join(appDir, "blog", "page.tsx"), []byte(`export default function Blog() {}`), 0o644)
	os.WriteFile(filepath.Join(appDir, "blog", "[slug]", "page.tsx"), []byte(`export default function BlogPost() {}`), 0o644)

	os.MkdirAll(filepath.Join(appDir, "api"), 0o755)
	os.WriteFile(filepath.Join(appDir, "api", "route.ts"), []byte(`export async function GET() {}`), 0o644)

	return dir
}

func TestScanner_Scan(t *testing.T) {
	dir := setupTestProject(t)

	s := New(dir)
	routes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(routes) == 0 {
		t.Fatal("expected at least one route")
	}

	types := make(map[RouteType]int)
	for _, r := range routes {
		types[r.Type]++
	}

	if types[RouteTypeLayout] == 0 {
		t.Error("expected at least one layout")
	}
	if types[RouteTypePage] == 0 {
		t.Error("expected at least one page")
	}
	if types[RouteTypeAPI] == 0 {
		t.Error("expected at least one API route")
	}
}

func TestScanner_DynamicRoutes(t *testing.T) {
	dir := setupTestProject(t)

	s := New(dir)
	routes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	dynamicCount := 0
	for _, r := range routes {
		if r.IsDynamic {
			dynamicCount++
		}
	}

	if dynamicCount == 0 {
		t.Error("expected at least one dynamic route")
	}
}

func TestScanner_GenerateManifest(t *testing.T) {
	dir := setupTestProject(t)

	s := New(dir)
	manifest, err := s.GenerateManifest()
	if err != nil {
		t.Fatal(err)
	}

	if manifest.TotalRoutes == 0 {
		t.Error("expected manifest to have routes")
	}
	if !manifest.HasApp {
		t.Error("expected manifest to detect app directory")
	}
}

func TestFilterByType(t *testing.T) {
	routes := []Route{
		{Type: RouteTypePage},
		{Type: RouteTypeLayout},
		{Type: RouteTypePage},
		{Type: RouteTypeAPI},
	}

	pages := FilterByType(routes, RouteTypePage)
	if len(pages) != 2 {
		t.Errorf("FilterByType(Page) returned %d, want 2", len(pages))
	}
}

func TestFindConflicts(t *testing.T) {
	routes := []Route{
		{Path: "/about", Type: RouteTypePage},
		{Path: "/about", Type: RouteTypeAPI},
	}

	conflicts := FindConflicts(routes)
	if len(conflicts) == 0 {
		t.Error("expected to find conflicts")
	}
}

func TestScanner_EmptyProject(t *testing.T) {
	dir := t.TempDir()

	s := New(dir)
	routes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(routes) != 0 {
		t.Errorf("expected 0 routes in empty project, got %d", len(routes))
	}
}

func TestScanner_PagesRouter(t *testing.T) {
	dir := t.TempDir()
	pagesDir := filepath.Join(dir, "pages")
	os.MkdirAll(pagesDir, 0o755)
	os.WriteFile(filepath.Join(pagesDir, "index.tsx"), []byte(`export default function Home() {}`), 0o644)
	os.WriteFile(filepath.Join(pagesDir, "about.tsx"), []byte(`export default function About() {}`), 0o644)
	os.MkdirAll(filepath.Join(pagesDir, "api"), 0o755)
	os.WriteFile(filepath.Join(pagesDir, "api", "users.ts"), []byte(`export default function handler() {}`), 0o644)

	s := New(dir)
	manifest, err := s.GenerateManifest()
	if err != nil {
		t.Fatal(err)
	}

	if !manifest.HasPages {
		t.Error("expected to detect pages directory")
	}
}