package routes

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RouteType represents the type of Next.js route.
type RouteType string

const (
	RouteTypePage     RouteType = "page"
	RouteTypeLayout   RouteType = "layout"
	RouteTypeError    RouteType = "error"
	RouteTypeLoading  RouteType = "loading"
	RouteTypeNotFound RouteType = "not-found"
	RouteTypeTemplate RouteType = "template"
	RouteTypeMiddleware RouteType = "middleware"
	RouteTypeAPI      RouteType = "api-route"
	RouteTypeMetadata RouteType = "metadata"
)

// Route represents a discovered Next.js route.
type Route struct {
	Path      string    `json:"path"`
	FilePath  string    `json:"file_path"`
	Type      RouteType `json:"type"`
	Layout    string    `json:"layout,omitempty"`
	Parent    string    `json:"parent,omitempty"`
	IsDynamic bool      `json:"is_dynamic"`
	IsGroup   bool      `json:"is_group"`
	Children  []string  `json:"children,omitempty"`
}

// Scanner detects Next.js routes in a project directory.
type Scanner struct {
	root string
}

// New creates a new route scanner.
func New(root string) *Scanner {
	return &Scanner{root: root}
}

// Scan detects all routes in the project.
func (s *Scanner) Scan() ([]Route, error) {
	var routes []Route

	appDir := filepath.Join(s.root, "app")
	pagesDir := filepath.Join(s.root, "pages")

	if info, err := os.Stat(appDir); err == nil && info.IsDir() {
		r, err := s.scanDir(appDir, "", "", "app")
		if err != nil {
			return nil, err
		}
		routes = append(routes, r...)
	}

	if info, err := os.Stat(pagesDir); err == nil && info.IsDir() {
		r, err := s.scanDir(pagesDir, "", "", "pages")
		if err != nil {
			return nil, err
		}
		routes = append(routes, r...)
	}

	s.buildParentChildRelationships(routes)

	return routes, nil
}

func (s *Scanner) scanDir(dir, parentLayout, parentPath, routerType string) ([]Route, error) {
	var routes []Route

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		name := entry.Name()

		if entry.IsDir() {
			subRoutes, err := s.scanDir(filepath.Join(dir, name), parentLayout, parentPath, routerType)
			if err != nil {
				return nil, err
			}
			routes = append(routes, subRoutes...)
			continue
		}

		route := s.parseFile(filepath.Join(dir, name), name, parentLayout, parentPath, routerType)
		if route != nil {
			if route.Type == RouteTypeLayout {
				parentLayout = route.FilePath
			}
			routes = append(routes, *route)
		}
	}

	return routes, nil
}

func (s *Scanner) parseFile(path, name, parentLayout, parentPath, routerType string) *Route {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)

	if ext != ".tsx" && ext != ".ts" && ext != ".jsx" && ext != ".js" {
		return nil
	}

	relPath, _ := filepath.Rel(s.root, path)

	route := &Route{
		FilePath: relPath,
		Layout:   parentLayout,
		Parent:   parentPath,
	}

	switch base {
	case "layout":
		route.Type = RouteTypeLayout
		route.Path = parentPath
	case "error":
		route.Type = RouteTypeError
		route.Path = parentPath
	case "loading":
		route.Type = RouteTypeLoading
		route.Path = parentPath
	case "not-found":
		route.Type = RouteTypeNotFound
		route.Path = parentPath
	case "template":
		route.Type = RouteTypeTemplate
		route.Path = parentPath
	case "page":
		route.Type = RouteTypePage
		route.Path = s.buildRoutePath(parentPath, filepath.Dir(relPath), routerType)
	case "route":
		route.Type = RouteTypeAPI
		route.Path = s.buildRoutePath(parentPath, filepath.Dir(relPath), routerType)
	case "middleware":
		route.Type = RouteTypeMiddleware
		route.Path = "/"
	case "metadata", "opengraph-image", "twitter-image", "icon", "apple-touch-icon", "favicon":
		route.Type = RouteTypeMetadata
		route.Path = parentPath
	default:
		route.Type = RouteTypePage
		route.Path = s.buildRoutePath(parentPath, filepath.Dir(relPath), routerType)
	}

	route.IsDynamic = strings.Contains(route.Path, "[") || strings.Contains(route.Path, "...")
	route.IsGroup = strings.HasPrefix(base, "(") && strings.HasSuffix(base, ")")

	return route
}

func (s *Scanner) buildRoutePath(parentPath, dirPath, routerType string) string {
	if routerType == "pages" {
		rel, _ := filepath.Rel(filepath.Join(s.root, "pages"), filepath.Join(s.root, dirPath))
		if rel == "." {
			return "/"
		}
		return "/" + filepath.ToSlash(rel)
	}

	rel, _ := filepath.Rel(filepath.Join(s.root, "app"), filepath.Join(s.root, dirPath))
	if rel == "." {
		return "/"
	}
	return "/" + filepath.ToSlash(rel)
}

func (s *Scanner) buildParentChildRelationships(routes []Route) {
	layouts := make(map[string]*Route)
	for i := range routes {
		if routes[i].Type == RouteTypeLayout {
			layouts[routes[i].FilePath] = &routes[i]
		}
	}

	for i := range routes {
		if routes[i].Layout != "" {
			if layout, ok := layouts[routes[i].Layout]; ok {
				layout.Children = append(layout.Children, routes[i].FilePath)
			}
		}
	}
}

// RouteManifest holds the complete route manifest.
type RouteManifest struct {
	Routes    []Route `json:"routes"`
	AppDir    string  `json:"app_dir,omitempty"`
	PagesDir  string  `json:"pages_dir,omitempty"`
	HasApp    bool    `json:"has_app"`
	HasPages  bool    `json:"has_pages"`
	TotalRoutes int   `json:"total_routes"`
}

// GenerateManifest creates a route manifest from discovered routes.
func (s *Scanner) GenerateManifest() (*RouteManifest, error) {
	routes, err := s.Scan()
	if err != nil {
		return nil, err
	}

	manifest := &RouteManifest{
		Routes:      routes,
		TotalRoutes: len(routes),
	}

	appDir := filepath.Join(s.root, "app")
	pagesDir := filepath.Join(s.root, "pages")

	if info, err := os.Stat(appDir); err == nil && info.IsDir() {
		manifest.AppDir = appDir
		manifest.HasApp = true
	}
	if info, err := os.Stat(pagesDir); err == nil && info.IsDir() {
		manifest.PagesDir = pagesDir
		manifest.HasPages = true
	}

	sort.Slice(manifest.Routes, func(i, j int) bool {
		return manifest.Routes[i].Path < manifest.Routes[j].Path
	})

	return manifest, nil
}

// FilterByType returns routes matching the given type.
func FilterByType(routes []Route, routeType RouteType) []Route {
	var result []Route
	for _, r := range routes {
		if r.Type == routeType {
			result = append(result, r)
		}
	}
	return result
}

// FindConflicts detects routes with the same path but different types.
func FindConflicts(routes []Route) []Route {
	pathTypes := make(map[string][]Route)
	for _, r := range routes {
		if r.Type == RouteTypePage || r.Type == RouteTypeAPI {
			pathTypes[r.Path] = append(pathTypes[r.Path], r)
		}
	}
	var conflicts []Route
	for _, rs := range pathTypes {
		if len(rs) > 1 {
			conflicts = append(conflicts, rs...)
		}
	}
	return conflicts
}