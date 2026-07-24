package plugins

import (
	"errors"
	"testing"
)

type ErrorPlugin struct {
	BasePlugin
	err error
}

func (p *ErrorPlugin) Execute(ctx *Context) error {
	return p.err
}

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestRegister(t *testing.T) {
	m := NewManager()
	p := NewNoopPlugin("test-plugin")

	if err := m.Register(p); err != nil {
		t.Fatal(err)
	}

	if m.PluginCount() != 1 {
		t.Errorf("expected 1 plugin, got %d", m.PluginCount())
	}
}

func TestRegister_Duplicate(t *testing.T) {
	m := NewManager()
	p1 := NewNoopPlugin("test-plugin")
	p2 := NewNoopPlugin("test-plugin")

	if err := m.Register(p1); err != nil {
		t.Fatal(err)
	}

	if err := m.Register(p2); err == nil {
		t.Error("expected error for duplicate plugin")
	}
}

func TestUnregister(t *testing.T) {
	m := NewManager()
	p := NewNoopPlugin("test-plugin")
	m.Register(p)

	m.Unregister("test-plugin")

	if m.PluginCount() != 0 {
		t.Errorf("expected 0 plugins, got %d", m.PluginCount())
	}
}

func TestUnregister_NonExistent(t *testing.T) {
	m := NewManager()
	m.Unregister("nonexistent")
}

func TestExecute(t *testing.T) {
	m := NewManager()
	p := NewNoopPlugin("test-plugin")
	m.Register(p)

	ctx := &Context{
		Hook: HookAfterBuild,
		Data: make(map[string]interface{}),
	}

	if err := m.Execute(HookAfterBuild, ctx); err != nil {
		t.Fatal(err)
	}
}

func TestExecute_NoPlugins(t *testing.T) {
	m := NewManager()
	ctx := &Context{Hook: HookAfterBuild}

	if err := m.Execute(HookAfterBuild, ctx); err != nil {
		t.Fatal(err)
	}
}

func TestExecute_PluginError(t *testing.T) {
	m := NewManager()
	p := &ErrorPlugin{
		BasePlugin: BasePlugin{
			Name_:    "error-plugin",
			Version_: "0.0.1",
			Hooks_:   []HookType{HookAfterBuild},
		},
		err: errors.New("plugin error"),
	}
	m.Register(p)

	ctx := &Context{Hook: HookAfterBuild}
	err := m.Execute(HookAfterBuild, ctx)
	if err == nil {
		t.Error("expected error from plugin")
	}
}

func TestGetPlugin(t *testing.T) {
	m := NewManager()
	p := NewNoopPlugin("test-plugin")
	m.Register(p)

	got, ok := m.GetPlugin("test-plugin")
	if !ok {
		t.Fatal("expected to find plugin")
	}
	if got.Name() != "test-plugin" {
		t.Errorf("expected name test-plugin, got %s", got.Name())
	}
}

func TestGetPlugin_NotFound(t *testing.T) {
	m := NewManager()
	_, ok := m.GetPlugin("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestListPlugins(t *testing.T) {
	m := NewManager()
	m.Register(NewNoopPlugin("plugin-a"))
	m.Register(NewNoopPlugin("plugin-b"))

	names := m.ListPlugins()
	if len(names) != 2 {
		t.Errorf("expected 2 plugins, got %d", len(names))
	}
}

func TestPluginCount(t *testing.T) {
	m := NewManager()
	if m.PluginCount() != 0 {
		t.Errorf("expected 0, got %d", m.PluginCount())
	}

	m.Register(NewNoopPlugin("plugin"))
	if m.PluginCount() != 1 {
		t.Errorf("expected 1, got %d", m.PluginCount())
	}
}

func TestClear(t *testing.T) {
	m := NewManager()
	m.Register(NewNoopPlugin("plugin-a"))
	m.Register(NewNoopPlugin("plugin-b"))

	m.Clear()

	if m.PluginCount() != 0 {
		t.Errorf("expected 0 plugins after clear, got %d", m.PluginCount())
	}
}

func TestBasePlugin(t *testing.T) {
	p := &BasePlugin{
		Name_:    "base",
		Version_: "1.0.0",
		Hooks_:   []HookType{HookConfigLoad, HookScanComplete},
	}

	if p.Name() != "base" {
		t.Errorf("expected name base, got %s", p.Name())
	}
	if p.Version() != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", p.Version())
	}
	if len(p.Hooks()) != 2 {
		t.Errorf("expected 2 hooks, got %d", len(p.Hooks()))
	}
}

func TestNoopPlugin(t *testing.T) {
	p := NewNoopPlugin("noop")

	if p.Name() != "noop" {
		t.Errorf("expected name noop, got %s", p.Name())
	}
	if p.Version() != "0.0.1" {
		t.Errorf("expected version 0.0.1, got %s", p.Version())
	}

	ctx := &Context{}
	if err := p.Execute(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestMultipleHooks(t *testing.T) {
	m := NewManager()
	p := NewNoopPlugin("multi-hook")
	m.Register(p)

	for _, hook := range []HookType{HookConfigLoad, HookScanComplete, HookAfterBuild} {
		ctx := &Context{Hook: hook}
		if err := m.Execute(hook, ctx); err != nil {
			t.Fatal(err)
		}
	}
}

func TestContext_Data(t *testing.T) {
	ctx := &Context{
		Hook: HookConfigLoad,
		Data: map[string]interface{}{
			"key": "value",
		},
		ProjectDir: "/test",
	}

	if ctx.Data["key"] != "value" {
		t.Errorf("expected value, got %v", ctx.Data["key"])
	}
	if ctx.ProjectDir != "/test" {
		t.Errorf("expected /test, got %s", ctx.ProjectDir)
	}
}

func TestConcurrentRegister(t *testing.T) {
	m := NewManager()
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			p := NewNoopPlugin("concurrent-plugin")
			m.Register(p)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestConcurrentExecute(t *testing.T) {
	m := NewManager()
	p := NewNoopPlugin("test-plugin")
	m.Register(p)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			ctx := &Context{Hook: HookAfterBuild}
			m.Execute(HookAfterBuild, ctx)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestHookTypes(t *testing.T) {
	hooks := []HookType{
		HookConfigLoad,
		HookScanComplete,
		HookGraphBuilt,
		HookBeforeParse,
		HookAfterParse,
		HookBeforeOptimize,
		HookAfterBuild,
		HookWatchEvent,
	}

	for _, h := range hooks {
		if h == "" {
			t.Error("expected non-empty hook type")
		}
	}
}
