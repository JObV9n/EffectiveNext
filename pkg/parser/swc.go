package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SWCParser wraps the SWC WASM parser for high-performance parsing.
type SWCParser struct {
	binary string
}

// NewSWCParser creates a new SWC parser.
// It looks for swc in the system PATH or node_modules.
func NewSWCParser(rootDir string) *SWCParser {
	binary := findSWCBinary(rootDir)
	return &SWCParser{binary: binary}
}

// Available returns true if the SWC parser is available.
func (p *SWCParser) Available() bool {
	return p.binary != ""
}

// SWCResult holds the parsed output from SWC.
type SWCResult struct {
	FilePath   string         `json:"-"`
	Body       []SWCModuleItem `json:"body"`
	Hash       uint64         `json:"-"`
	Shebang    string         `json:"shebang,omitempty"`
}

// SWCModuleItem represents a top-level statement/declaration.
type SWCModuleItem struct {
	Type       string `json:"type"`
	Span       SWCSpan `json:"span"`
	ExportDecl *struct {
		Span SWCSpan `json:"span"`
	} `json:"ExportDeclaration,omitempty"`
	ImportDecl *SWCImportDecl `json:"ImportDeclaration,omitempty"`
}

// SWCImportDecl represents an import declaration.
type SWCImportDecl struct {
	Span     SWCSpan       `json:"span"`
	Specifiers []SWCSpecifier `json:"specifiers"`
	Source   SWCStr         `json:"source"`
}

// SWCSpecifier represents an import specifier.
type SWCSpecifier struct {
	Type string `json:"type"`
}

// SWCStr represents a string literal.
type SWCStr struct {
	Value string `json:"value"`
}

// SWCSpan represents a source span.
type SWCSpan struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// SWCOptions represents parser options.
type SWCOptions struct {
	Jsx    bool `json:"jsx"`
	TSX    bool `json:"tsx"`
	TS     bool `json:"typescript"`
	Target string `json:"target,omitempty"`
}

// ParseFile parses a file using SWC.
func (p *SWCParser) ParseFile(path string) (*SWCResult, error) {
	if p.binary == "" {
		return nil, fmt.Errorf("SWC parser not available")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	opts := SWCOptions{
		TSX: strings.HasSuffix(path, ".tsx"),
		Jsx: strings.HasSuffix(path, ".jsx"),
		TS:  strings.HasSuffix(path, ".ts"),
	}

	optsJSON, _ := json.Marshal(opts)
	cmd := exec.Command(p.binary, "parse", "--json", string(optsJSON))
	cmd.Stdin = strings.NewReader(string(data))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("swc parse error: %w: %s", err, string(out))
	}

	var result SWCResult
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("swc output parse error: %w", err)
	}

	result.FilePath = path
	return &result, nil
}

// ParseFileFallback parses a file, trying SWC first, falling back to regex.
func ParseFileFallback(path string) (*Result, error) {
	parser := NewSWCParser(filepath.Dir(path))
	if parser.Available() {
		swcResult, err := parser.ParseFile(path)
		if err == nil {
			return swcResultToResult(swcResult), nil
		}
	}

	return ParseFile(path)
}

func swcResultToResult(swc *SWCResult) *Result {
	result := &Result{Path: swc.FilePath}

	for _, item := range swc.Body {
		if item.ImportDecl != nil {
			result.Imports = append(result.Imports, Import{
				Path:    item.ImportDecl.Source.Value,
				Dynamic: false,
			})
		}
		if item.ExportDecl != nil {
			result.Exports = append(result.Exports, Export{
				Name: "(default)",
			})
		}
	}

	result.HasMetadata = false
	for _, item := range swc.Body {
		if item.Type == "ExportDeclaration" {
			result.HasMetadata = true
		}
	}

	return result
}

func findSWCBinary(rootDir string) string {
	candidates := []string{
		filepath.Join(rootDir, "node_modules", ".bin", "swc"),
		filepath.Join(rootDir, "node_modules", "@swc", "cli", "bin", "swc.js"),
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			if strings.HasSuffix(c, ".js") {
				return "node " + c
			}
			return c
		}
	}

	path, err := exec.LookPath("swc")
	if err == nil {
		return path
	}

	return ""
}
