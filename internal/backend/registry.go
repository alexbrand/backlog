package backend

import (
	"fmt"
	"sync"
)

// Factory is a function that creates a new instance of a backend.
type Factory func() Backend

// registry holds registered backend factories.
var (
	registryMu sync.RWMutex
	backends   = make(map[string]Factory)
)

// Register registers a backend factory under the given name.
// It panics if the name is already registered or if the factory is nil.
func Register(name string, factory Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()

	if factory == nil {
		panic(fmt.Sprintf("backend: Register factory is nil for %q", name))
	}
	if _, exists := backends[name]; exists {
		panic(fmt.Sprintf("backend: Register called twice for %q", name))
	}
	backends[name] = factory
}

// Get returns a new instance of the backend with the given name.
// Returns an error if no backend is registered with that name.
func Get(name string) (Backend, error) {
	registryMu.RLock()
	factory, exists := backends[name]
	registryMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("backend: unknown backend %q", name)
	}
	return factory(), nil
}

// List returns the names of all registered backends.
func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	names := make([]string, 0, len(backends))
	for name := range backends {
		names = append(names, name)
	}
	return names
}

// IsRegistered returns true if a backend with the given name is registered.
func IsRegistered(name string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()

	_, exists := backends[name]
	return exists
}

// Unregister removes a backend from the registry.
// This is primarily useful for testing.
func Unregister(name string) {
	registryMu.Lock()
	defer registryMu.Unlock()

	delete(backends, name)
}

// UnregisterAll removes all backends from the registry.
// This is primarily useful for testing.
func UnregisterAll() {
	registryMu.Lock()
	defer registryMu.Unlock()

	backends = make(map[string]Factory)
}
