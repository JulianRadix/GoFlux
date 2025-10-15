package goflux

import (
	"net/http"
)

// HandlerFunc defines the handler used by GoFlux
// It receives the response writer, request, and any URL parameters
type HandlerFunc func(http.ResponseWriter, *http.Request, Params)

// Params is a slice of parameter key-value pairs extracted from the URL
type Params []Param

// Param represents a single URL parameter (like id in /users/:id)
type Param struct {
	Key   string
	Value string
}

// ByName returns the value of the first parameter with the given name
func (ps Params) ByName(name string) string {
	for _, p := range ps {
		if p.Key == name {
			return p.Value
		}
	}
	return ""
}

// nodeType represents what kind of node this is in the tree
type nodeType uint8

const (
	static   nodeType = iota // static path like /users
	root                     // root of the tree
	param                    // parameter like :id
	catchAll                 // catch-all like *filepath
)

// node represents a single node in the radix tree
type node struct {
	path     string                 // the path segment this node represents
	handlers map[string]HandlerFunc // maps HTTP method -> handler function
	nType    nodeType               // what type of node is this
	children []*node                // child nodes
}

// addRoute adds a new route to the tree
// For now, this handles only static routes (no :params or *catchall)
func (n *node) addRoute(path string, method string, handler HandlerFunc) {
	// If this is an empty tree, just set this node as the route
	if len(n.path) == 0 && len(n.children) == 0 {
		n.path = path
		n.nType = root
		n.handlers = make(map[string]HandlerFunc)
		n.handlers[method] = handler
		return
	}

	// Find the longest common prefix between the new path and current node's path
	commonPrefix := longestCommonPrefix(path, n.path)

	// Case 1: Paths match exactly, just add the handler
	if commonPrefix == len(n.path) && commonPrefix == len(path) {
		if n.handlers == nil {
			n.handlers = make(map[string]HandlerFunc)
		}
		// Check if handler already exists for this method
		if _, exists := n.handlers[method]; exists {
			panic("handler already registered for path '" + path + "' and method '" + method + "'")
		}
		n.handlers[method] = handler
		return
	}

	// Case 2: New path is a prefix of current node (like /users when node is /users/profile)
	if commonPrefix == len(path) && commonPrefix < len(n.path) {
		// Split the node
		child := &node{
			path:     n.path[commonPrefix:],
			handlers: n.handlers,
			children: n.children,
			nType:    static,
		}

		n.path = path
		n.children = []*node{child}
		n.handlers = make(map[string]HandlerFunc)
		n.handlers[method] = handler
		return
	}

	// Case 3: We need to split the current node (uncommon prefix)
	if commonPrefix < len(n.path) {
		// Create a child with the remaining part of current path
		child := &node{
			path:     n.path[commonPrefix:],
			handlers: n.handlers,
			children: n.children,
			nType:    static,
		}

		// Update current node to only have the common prefix
		n.path = n.path[:commonPrefix]
		n.children = []*node{child}
		n.handlers = nil
	}

	// If new path is longer than common prefix, we need to add a child
	if commonPrefix < len(path) {
		remainingPath := path[commonPrefix:]

		// Check if a child already exists that matches
		for _, child := range n.children {
			if child.path[0] == remainingPath[0] {
				// Recursively add to this child
				child.addRoute(remainingPath, method, handler)
				return
			}
		}

		// No matching child, create a new one
		newChild := &node{
			path:     remainingPath,
			handlers: make(map[string]HandlerFunc),
			nType:    static,
		}
		newChild.handlers[method] = handler
		n.children = append(n.children, newChild)
	}
}

// longestCommonPrefix finds the length of the common prefix between two strings
func longestCommonPrefix(a, b string) int {
	i := 0
	max := min(len(a), len(b))
	for i < max && a[i] == b[i] {
		i++
	}
	return i
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
