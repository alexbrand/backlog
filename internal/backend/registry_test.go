package backend

import (
	"sort"
	"testing"
)

// mockBackend is a minimal Backend implementation for testing.
type mockBackend struct {
	name    string
	version string
}

func (m *mockBackend) Name() string    { return m.name }
func (m *mockBackend) Version() string { return m.version }

func (m *mockBackend) Connect(cfg Config) error                     { return nil }
func (m *mockBackend) Disconnect() error                            { return nil }
func (m *mockBackend) HealthCheck() (HealthStatus, error)           { return HealthStatus{OK: true}, nil }
func (m *mockBackend) List(filters TaskFilters) (*TaskList, error)  { return &TaskList{}, nil }
func (m *mockBackend) Get(id string) (*Task, error)                 { return nil, nil }
func (m *mockBackend) Create(input TaskInput) (*Task, error)        { return nil, nil }
func (m *mockBackend) Update(id string, changes TaskChanges) (*Task, error) {
	return nil, nil
}
func (m *mockBackend) Delete(id string) error                       { return nil }
func (m *mockBackend) Move(id string, status Status) (*Task, error) { return nil, nil }
func (m *mockBackend) Assign(id string, assignee string) (*Task, error) {
	return nil, nil
}
func (m *mockBackend) Unassign(id string) (*Task, error)            { return nil, nil }
func (m *mockBackend) ListComments(id string) ([]Comment, error)    { return nil, nil }
func (m *mockBackend) AddComment(id string, body string) (*Comment, error) {
	return nil, nil
}

func newMockBackend(name, version string) Factory {
	return func() Backend {
		return &mockBackend{name: name, version: version}
	}
}

func TestRegisterAndGet(t *testing.T) {
	UnregisterAll()
	defer UnregisterAll()

	Register("test", newMockBackend("test", "1.0.0"))

	backend, err := Get("test")
	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}
	if backend == nil {
		t.Fatal("Get() returned nil backend")
	}
	if backend.Name() != "test" {
		t.Errorf("backend.Name() = %q, want %q", backend.Name(), "test")
	}
	if backend.Version() != "1.0.0" {
		t.Errorf("backend.Version() = %q, want %q", backend.Version(), "1.0.0")
	}
}

func TestGetUnknownBackend(t *testing.T) {
	UnregisterAll()
	defer UnregisterAll()

	_, err := Get("nonexistent")
	if err == nil {
		t.Fatal("Get() expected error for unknown backend, got nil")
	}
}

func TestRegisterNilFactory(t *testing.T) {
	UnregisterAll()
	defer UnregisterAll()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Register(nil) did not panic")
		}
	}()

	Register("nil-factory", nil)
}

func TestRegisterDuplicate(t *testing.T) {
	UnregisterAll()
	defer UnregisterAll()

	Register("duplicate", newMockBackend("duplicate", "1.0.0"))

	defer func() {
		if r := recover(); r == nil {
			t.Error("Register() with duplicate name did not panic")
		}
	}()

	Register("duplicate", newMockBackend("duplicate", "2.0.0"))
}

func TestList(t *testing.T) {
	UnregisterAll()
	defer UnregisterAll()

	Register("github", newMockBackend("github", "1.0.0"))
	Register("linear", newMockBackend("linear", "1.0.0"))
	Register("local", newMockBackend("local", "1.0.0"))

	names := List()
	sort.Strings(names)

	expected := []string{"github", "linear", "local"}
	if len(names) != len(expected) {
		t.Fatalf("List() returned %d backends, want %d", len(names), len(expected))
	}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("List()[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestIsRegistered(t *testing.T) {
	UnregisterAll()
	defer UnregisterAll()

	Register("exists", newMockBackend("exists", "1.0.0"))

	if !IsRegistered("exists") {
		t.Error("IsRegistered() = false for registered backend")
	}
	if IsRegistered("notexists") {
		t.Error("IsRegistered() = true for unregistered backend")
	}
}

func TestUnregister(t *testing.T) {
	UnregisterAll()
	defer UnregisterAll()

	Register("removeme", newMockBackend("removeme", "1.0.0"))
	if !IsRegistered("removeme") {
		t.Fatal("backend not registered")
	}

	Unregister("removeme")
	if IsRegistered("removeme") {
		t.Error("IsRegistered() = true after Unregister()")
	}
}

func TestGetCreatesNewInstances(t *testing.T) {
	UnregisterAll()
	defer UnregisterAll()

	callCount := 0
	Register("counter", func() Backend {
		callCount++
		return &mockBackend{name: "counter", version: "1.0.0"}
	})

	_, _ = Get("counter")
	_, _ = Get("counter")
	_, _ = Get("counter")

	if callCount != 3 {
		t.Errorf("Factory called %d times, want 3", callCount)
	}
}
