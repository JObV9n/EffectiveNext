package plugins

import (
	"fmt"
	"sync"
)

// HookType represents a plugin hook.
type HookType string

const (
	HookConfigLoad     HookType = "onConfigLoad"
	HookScanComplete   HookType = "onScanComplete"
	HookGraphBuilt     HookType = "onGraphBuilt"
	HookBeforeParse    HookType = "beforeParse"
	HookAfterParse     HookType = "afterParse"
	HookBeforeOptimize HookType = "beforeOptimize"
	HookAfterBuild     HookType = "afterBuild"
	HookWatchEvent     HookType = "onWatchEvent"
)

// Context provides data to plugin hooks.
type Context struct {
	Hook      HookType
	Data      map[string]interface{}
	ProjectDir string
	Config    interface{}
}

// Plugin is the interface that plugins must implement.
type Plugin interface {
	Name() string
	Version() string
	Hooks() []HookType
	Execute(ctx *Context) error
}

// Manager manages plugin registration and execution.
type Manager struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	hooks   map[HookType][]Plugin
}

// NewManager creates a new plugin manager.
func NewManager() *Manager {
	return &Manager{
		plugins: make(map[string]Plugin),
		hooks:   make(map[HookType][]Plugin),
	}
}

// Register registers a plugin.
func (m *Manager) Register(p Plugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := p.Name()
	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("plugin %q already registered", name)
	}

	m.plugins[name] = p
	for _, hook := range p.Hooks() {
		m.hooks[hook] = append(m.hooks[hook], p)
	}
	return nil
}

// Unregister removes a plugin.
func (m *Manager) Unregister(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.plugins[name]
	if !ok {
		return
	}

	delete(m.plugins, name)
	for _, hook := range p.Hooks() {
		plugins := m.hooks[hook]
		for i, pp := range plugins {
			if pp.Name() == name {
				m.hooks[hook] = append(plugins[:i], plugins[i+1:]...)
				break
			}
		}
	}
}

// Execute runs all plugins registered for the given hook.
func (m *Manager) Execute(hook HookType, ctx *Context) error {
	m.mu.RLock()
	plugins := m.hooks[hook]
	m.mu.RUnlock()

	for _, p := range plugins {
		if err := p.Execute(ctx); err != nil {
			return fmt.Errorf("plugin %q failed on hook %q: %w", p.Name(), hook, err)
		}
	}
	return nil
}

// GetPlugin returns a plugin by name.
func (m *Manager) GetPlugin(name string) (Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.plugins[name]
	return p, ok
}

// ListPlugins returns all registered plugin names.
func (m *Manager) ListPlugins() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var names []string
	for name := range m.plugins {
		names = append(names, name)
	}
	return names
}

// PluginCount returns the number of registered plugins.
func (m *Manager) PluginCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.plugins)
}

// Clear removes all registered plugins.
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.plugins = make(map[string]Plugin)
	m.hooks = make(map[HookType][]Plugin)
}

// BasePlugin provides a default implementation for common plugin methods.
type BasePlugin struct {
	Name_    string
	Version_ string
	Hooks_   []HookType
}

func (p *BasePlugin) Name() string       { return p.Name_ }
func (p *BasePlugin) Version() string    { return p.Version_ }
func (p *BasePlugin) Hooks() []HookType  { return p.Hooks_ }

// NoopPlugin is a plugin that does nothing.
type NoopPlugin struct {
	BasePlugin
}

func NewNoopPlugin(name string) *NoopPlugin {
	return &NoopPlugin{
		BasePlugin: BasePlugin{
			Name_:    name,
			Version_: "0.0.1",
			Hooks_:   []HookType{HookAfterBuild},
		},
	}
}

func (p *NoopPlugin) Execute(ctx *Context) error {
	return nil
}