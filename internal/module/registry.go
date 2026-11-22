package module

import (
	"fmt"
	"sort"
	"sync"
)

// VisibleModule represents a module currently activated and displayed in the tab bar.
type VisibleModule struct {
	Index    int    // 1-based index in the current visible tabs bar (e.g. 1, 2, 3...)
	ID       ID     // Canonical module identifier
	Title    string // Clean title (e.g. "Containers" or custom title)
	TabLabel string // Formatted label (e.g. "1: Containers")
	Module   Module // Underlying module implementation
	Order    int    // Evaluated sorting weight
}

// Registry manages available modules, their user configuration overrides, and visible ordering.
type Registry struct {
	mu      sync.RWMutex
	modules map[ID]Module
	order   []ID
	configs map[ID]Config
}

// NewRegistry constructs an empty module registry.
func NewRegistry() *Registry {
	return &Registry{
		modules: make(map[ID]Module),
		order:   make([]ID, 0),
		configs: make(map[ID]Config),
	}
}

// DefaultRegistry creates and populates a Registry with standard Dockpulse built-in modules.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register(ContainersModule)
	r.Register(ImagesModule)
	r.Register(ComposeModule)
	r.Register(NetworksModule)
	r.Register(VolumesModule)
	r.Register(SwarmModule)
	return r
}

// Register adds a module to the registry. If a module with the same ID already exists, it is updated.
func (r *Registry) Register(m Module) {
	if m == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	id := m.ID()
	if _, exists := r.modules[id]; !exists {
		r.order = append(r.order, id)
	}
	r.modules[id] = m
}

// SetModuleConfig applies user configuration overrides for a specific module.
func (r *Registry) SetModuleConfig(id ID, cfg Config) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.configs[id] = cfg
}

// ApplyConfigs applies a map of string-keyed module configurations.
func (r *Registry) ApplyConfigs(cfgs map[string]Config) {
	if len(cfgs) == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	for k, cfg := range cfgs {
		id, err := ParseID(k)
		if err == nil {
			r.configs[id] = cfg
		} else {
			r.configs[ID(k)] = cfg
		}
	}
}

// GetModule retrieves a registered module by ID.
func (r *Registry) GetModule(id ID) (Module, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.modules[id]
	return m, ok
}

// GetModuleConfig returns the current configuration applied to a module.
func (r *Registry) GetModuleConfig(id ID) Config {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.configs[id]
}

// VisibleModules returns the ordered list of currently enabled and visible modules.
func (r *Registry) VisibleModules(ctx Context) []VisibleModule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	type sortItem struct {
		id    ID
		m     Module
		cfg   Config
		order int
		title string
	}

	var active []sortItem
	for _, id := range r.order {
		m := r.modules[id]
		cfg := r.configs[id]

		if !m.IsEnabled(ctx, cfg) {
			continue
		}

		effectiveOrder := m.DefaultOrder()
		if cfg.Order > 0 {
			effectiveOrder = cfg.Order
		}

		effectiveTitle := m.DefaultTitle()
		if cfg.Title != "" {
			effectiveTitle = cfg.Title
		}

		active = append(active, sortItem{
			id:    id,
			m:     m,
			cfg:   cfg,
			order: effectiveOrder,
			title: effectiveTitle,
		})
	}

	// Sort active modules by effective order ascending
	sort.SliceStable(active, func(i, j int) bool {
		return active[i].order < active[j].order
	})

	result := make([]VisibleModule, len(active))
	for i, item := range active {
		idx := i + 1
		result[i] = VisibleModule{
			Index:    idx,
			ID:       item.id,
			Title:    item.title,
			TabLabel: fmt.Sprintf("%d: %s", idx, item.title),
			Module:   item.m,
			Order:    item.order,
		}
	}

	return result
}

// FindVisibleByIndex looks up a visible module by its 1-based tab index.
func (r *Registry) FindVisibleByIndex(index int, ctx Context) (VisibleModule, bool) {
	visible := r.VisibleModules(ctx)
	if index < 1 || index > len(visible) {
		return VisibleModule{}, false
	}
	return visible[index-1], true
}

// FindVisibleByID looks up a visible module by its ID.
func (r *Registry) FindVisibleByID(id ID, ctx Context) (VisibleModule, bool) {
	for _, v := range r.VisibleModules(ctx) {
		if v.ID == id {
			return v, true
		}
	}
	return VisibleModule{}, false
}

// IsModuleVisible checks if a module with the given ID is currently visible.
func (r *Registry) IsModuleVisible(id ID, ctx Context) bool {
	_, visible := r.FindVisibleByID(id, ctx)
	return visible
}

// VisibleCount returns the count of currently visible modules.
func (r *Registry) VisibleCount(ctx Context) int {
	return len(r.VisibleModules(ctx))
}

// TotalCount returns the count of all registered modules.
func (r *Registry) TotalCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.modules)
}
