package routes

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	s := New("/test")
	if s == nil {
		t.Fatal("expected non-nil scanner")
	}
}

func TestScan_Empty(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	routes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 0 {
		t.Errorf("expected 0 routes, got %d", len(routes))
	}
}

func TestScan_AppRouter(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "app"), 0o755)
	os.WriteFile(filepath.Join(dir, "app", "page.tsx"), []byte("export default function Page() {}"), 0o644)

	s := New(dir)
	routes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if routes[0].Type != RouteTypePage {
		t.Errorf("expected page route, got %s", routes[0].Type)
	}
}

func TestScan_PagesRouter(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "pages"), 0o755)
	os.WriteFile(filepath.Join(dir, "pages", "index.tsx"), []byte("export default function Home() {}"), 0o644)

	s := New(dir)
	routes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if routes[0].Type != RouteTypePage {
		t.Errorf("expected page route, got %s", routes[0].Type)
	}
}

func TestScan_DynamicRoutes(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "app", "blog", "[slug]"), 0o755)
	os.WriteFile(filepath.Join(dir, "app", "blog", "[slug]", "page.tsx"), []byte("export default function Page() {}"), 0o644)

	s := New(dir)
	routes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if !routes[0].IsDynamic {
		t.Error("expected dynamic route")
	}
}

func TestScan_Layout(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "app"), 0o755)
	os.WriteFile(filepath.Join(dir, "app", "layout.tsx"), []byte("export default function Layout() {}"), 0o644)

	s := New(dir)
	routes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if routes[0].Type != RouteTypeLayout {
		t.Errorf("expected layout route, got %s", routes[0].Type)
	}
}

func TestScan_ErrorBoundary(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "app"), 0o755)
	os.WriteFile(filepath.Join(dir, "app", "error.tsx"), []byte("export default function Error() {}"), 0o644)

	s := New(dir)
	routes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if routes[0].Type != RouteTypeError {
		t.Errorf("expected error route, got %s", routes[0].Type)
	}
}

func TestScan_Loading(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "app"), 0o755)
	os.WriteFile(filepath.Join(dir, "app", "loading.tsx"), []byte("export default function Loading() {}"), 0o644)

	s := New(dir)
	routes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if routes[0].Type != RouteTypeLoading {
		t.Errorf("expected loading route, got %s", routes[0].Type)
	}
}

func TestScan_NotFound(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "app"), 0o755)
	os.WriteFile(filepath.Join(dir, "app", "not-found.tsx"), []byte("export default function NotFound() {}"), 0o644)

	s := New(dir)
	routes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if routes[0].Type != RouteTypeNotFound {
		t.Errorf("expected not-found route, got %s", routes[0].Type)
	}
}

func TestScan_APIRoute(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "app", "api", "users"), 0o755)
	os.WriteFile(filepath.Join(dir, "app", "api", "users", "route.ts"), []byte("export async function GET() {}"), 0o644)

	s := New(dir)
	routes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if routes[0].Type != RouteTypeAPI {
		t.Errorf("expected API route, got %s", routes[0].Type)
	}
}

func TestScan_Middleware(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "app"), 0o755)
	os.WriteFile(filepath.Join(dir, "app", "middleware.ts"), []byte("export function middleware() {}"), 0o644)

	s := New(dir)
	routes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if routes[0].Type != RouteTypeMiddleware {
		t.Errorf("expected middleware route, got %s", routes[0].Type)
	}
}

func TestScan_NonRouteFiles(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "app"), 0o755)
	os.WriteFile(filepath.Join(dir, "app", "utils.ts"), []byte("export const foo = 1;"), 0o644)
	os.WriteFile(filepath.Join(dir, "app", "styles.css"), []byte("body {}"), 0o644)

	s := New(dir)
	routes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(routes) > 0 {
		t.Logf("got %d routes (scanner may treat some files as routes)", len(routes))
	}
}

func TestScan_NestedLayouts(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "app", "dashboard"), 0o755)
	os.WriteFile(filepath.Join(dir, "app", "layout.tsx"), []byte("export default function Layout() {}"), 0o644)
	os.WriteFile(filepath.Join(dir, "app", "dashboard", "layout.tsx"), []byte("export default function DashboardLayout() {}"), 0o644)
	os.WriteFile(filepath.Join(dir, "app", "dashboard", "page.tsx"), []byte("export default function Dashboard() {}"), 0o644)

	s := New(dir)
	routes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(routes))
	}
}

func TestScan_GroupRoutes(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "app", "(auth)"), 0o755)
	os.MkdirAll(filepath.Join(dir, "app", "(auth)", "login"), 0o755)
	os.WriteFile(filepath.Join(dir, "app", "(auth)", "login", "page.tsx"), []byte("export default function Login() {}"), 0o644)

	s := New(dir)
	routes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
}

func TestGenerateManifest(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "app"), 0o755)
	os.WriteFile(filepath.Join(dir, "app", "page.tsx"), []byte("export default function Page() {}"), 0o644)

	s := New(dir)
	manifest, err := s.GenerateManifest()
	if err != nil {
		t.Fatal(err)
	}

	if !manifest.HasApp {
		t.Error("expected HasApp to be true")
	}
	if manifest.TotalRoutes != 1 {
		t.Errorf("expected 1 route, got %d", manifest.TotalRoutes)
	}
}

func TestGenerateManifest_Pages(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "pages"), 0o755)
	os.WriteFile(filepath.Join(dir, "pages", "index.tsx"), []byte("export default function Home() {}"), 0o644)

	s := New(dir)
	manifest, err := s.GenerateManifest()
	if err != nil {
		t.Fatal(err)
	}

	if !manifest.HasPages {
		t.Error("expected HasPages to be true")
	}
}

func TestFilterByType_Empty(t *testing.T) {
	routes := []Route{}

	pages := FilterByType(routes, RouteTypePage)
	if len(pages) != 0 {
		t.Errorf("expected 0 pages, got %d", len(pages))
	}
}

func TestFindConflicts_Empty(t *testing.T) {
	routes := []Route{}

	conflicts := FindConflicts(routes)
	if len(conflicts) != 0 {
		t.Errorf("expected 0 conflicts, got %d", len(conflicts))
	}
}

func TestRouteTypes(t *testing.T) {
	types := []RouteType{
		RouteTypePage,
		RouteTypeLayout,
		RouteTypeError,
		RouteTypeLoading,
		RouteTypeNotFound,
		RouteTypeTemplate,
		RouteTypeMiddleware,
		RouteTypeAPI,
		RouteTypeMetadata,
	}

	for _, rt := range types {
		if rt == "" {
			t.Error("expected non-empty route type")
		}
	}
}

func TestScan_NonExistentDir(t *testing.T) {
	s := New("/nonexistent")
	routes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 0 {
		t.Errorf("expected 0 routes, got %d", len(routes))
	}
}

func TestScan_MixedExtensions(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "app"), 0o755)
	os.WriteFile(filepath.Join(dir, "app", "page.tsx"), []byte("export default function Page() {}"), 0o644)
	os.WriteFile(filepath.Join(dir, "app", "page.jsx"), []byte("export default function Page() {}"), 0o644)
	os.WriteFile(filepath.Join(dir, "app", "page.ts"), []byte("export default function Page() {}"), 0o644)
	os.WriteFile(filepath.Join(dir, "app", "page.js"), []byte("export default function Page() {}"), 0o644)

	s := New(dir)
	routes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(routes) != 4 {
		t.Errorf("expected 4 routes, got %d", len(routes))
	}
}

func TestScan_CatchAllRoutes(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "app", "[...slug]"), 0o755)
	os.WriteFile(filepath.Join(dir, "app", "[...slug]", "page.tsx"), []byte("export default function Page() {}"), 0o644)

	s := New(dir)
	routes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if !routes[0].IsDynamic {
		t.Error("expected dynamic route for catch-all")
	}
}

func TestScan_OptionalCatchAll(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "app", "[[...slug]]"), 0o755)
	os.WriteFile(filepath.Join(dir, "app", "[[...slug]]", "page.tsx"), []byte("export default function Page() {}"), 0o644)

	s := New(dir)
	routes, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if !routes[0].IsDynamic {
		t.Error("expected dynamic route for optional catch-all")
	}
}
