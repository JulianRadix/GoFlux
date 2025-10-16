package goflux

import (
	"net/http"
	"testing"
)

// Simple test handler
func testHandler(w http.ResponseWriter, r *http.Request, p Params) {
	w.Write([]byte("test"))
}

func TestAddRoute_EmptyTree(t *testing.T) {
	tree := &node{}
	tree.addRoute("/users", "GET", testHandler)

	if tree.path != "/users" {
		t.Errorf("Expected path '/users', got '%s'", tree.path)
	}

	if tree.handlers["GET"] == nil {
		t.Error("Expected GET handler to be registered")
	}
}

func TestAddRoute_ExactMatch(t *testing.T) {
	tree := &node{}
	tree.addRoute("/users", "GET", testHandler)
	tree.addRoute("/users", "POST", testHandler)

	if len(tree.handlers) != 2 {
		t.Errorf("Expected 2 handlers, got %d", len(tree.handlers))
	}

	if tree.handlers["GET"] == nil {
		t.Error("Expected GET handler")
	}

	if tree.handlers["POST"] == nil {
		t.Error("Expected POST handler")
	}
}

func TestAddRoute_DuplicateMethod(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for duplicate route registration")
		}
	}()

	tree := &node{}
	tree.addRoute("/users", "GET", testHandler)
	tree.addRoute("/users", "GET", testHandler) // Should panic
}

func TestAddRoute_MultipleRoutes(t *testing.T) {
	tree := &node{}
	tree.addRoute("/users", "GET", testHandler)
	tree.addRoute("/about", "GET", testHandler)
	tree.addRoute("/contact", "GET", testHandler)

	// After adding different routes, tree should have split
	if len(tree.children) == 0 {
		t.Error("Expected tree to have children")
	}
}

func TestAddRoute_CommonPrefix(t *testing.T) {
	tree := &node{}
	tree.addRoute("/users", "GET", testHandler)
	tree.addRoute("/user", "GET", testHandler)

	// Should split into /user with child 's'
	if tree.path != "/user" {
		t.Errorf("Expected path '/user', got '%s'", tree.path)
	}

	if len(tree.children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(tree.children))
	}

	if tree.children[0].path != "s" {
		t.Errorf("Expected child path 's', got '%s'", tree.children[0].path)
	}
}

func TestAddRoute_NestedRoutes(t *testing.T) {
	tree := &node{}
	tree.addRoute("/users/profile", "GET", testHandler)
	tree.addRoute("/users/settings", "GET", testHandler)
	tree.addRoute("/users", "GET", testHandler)

	// Root should be /users
	if tree.path != "/users" {
		t.Errorf("Expected path '/users', got '%s'", tree.path)
	}

	// Should have handler for /users itself
	if tree.handlers["GET"] == nil {
		t.Error("Expected GET handler for /users")
	}

	// Should have children for /profile and /settings
	if len(tree.children) == 0 {
		t.Error("Expected children for nested routes")
	}
}

// Tests for getValue (route lookup)

func TestGetValue_ExactMatch(t *testing.T) {
	tree := &node{}
	tree.addRoute("/users", "GET", testHandler)

	handler, _ := tree.getValue("/users", "GET")
	if handler == nil {
		t.Error("Expected to find handler for /users GET")
	}
}

func TestGetValue_NotFound(t *testing.T) {
	tree := &node{}
	tree.addRoute("/users", "GET", testHandler)

	handler, _ := tree.getValue("/posts", "GET")
	if handler != nil {
		t.Error("Expected no handler for /posts")
	}
}

func TestGetValue_WrongMethod(t *testing.T) {
	tree := &node{}
	tree.addRoute("/users", "GET", testHandler)

	handler, _ := tree.getValue("/users", "POST")
	if handler != nil {
		t.Error("Expected no handler for POST /users")
	}
}

func TestGetValue_NestedRoute(t *testing.T) {
	tree := &node{}
	tree.addRoute("/users/profile", "GET", testHandler)
	tree.addRoute("/users/settings", "GET", testHandler)

	handler, _ := tree.getValue("/users/profile", "GET")
	if handler == nil {
		t.Error("Expected to find handler for /users/profile")
	}

	handler, _ = tree.getValue("/users/settings", "GET")
	if handler == nil {
		t.Error("Expected to find handler for /users/settings")
	}
}

func TestGetValue_MultipleRoutes(t *testing.T) {
	tree := &node{}
	tree.addRoute("/users", "GET", testHandler)
	tree.addRoute("/about", "GET", testHandler)
	tree.addRoute("/contact", "GET", testHandler)

	handler, _ := tree.getValue("/users", "GET")
	if handler == nil {
		t.Error("Expected to find /users")
	}

	handler, _ = tree.getValue("/about", "GET")
	if handler == nil {
		t.Error("Expected to find /about")
	}

	handler, _ = tree.getValue("/contact", "GET")
	if handler == nil {
		t.Error("Expected to find /contact")
	}
}
