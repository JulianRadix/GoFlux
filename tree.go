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
	path      string                 // the path segment this node represents
	handlers  map[string]HandlerFunc // maps HTTP method -> handler function
	nType     nodeType               // what type of node is this
	children  []*node                // child nodes
	wildChild bool                   // true if any child is param or catchAll
}

// addRoute adds a new route to the tree
func (n *node) addRoute(path string, method string, handler HandlerFunc) {
	// If this is an empty tree
	if len(n.path) == 0 && len(n.children) == 0 {
		n.nType = root

		// Check if path has wildcards
		wildcard, wildcardIndex, valid := findWildcard(path)

		if wildcardIndex >= 0 {
			if !valid {
				panic("invalid wildcard in path: " + path)
			}

			// Split into static part and wildcard part
			if wildcardIndex > 0 {
				n.path = path[:wildcardIndex]

				// Create wildcard child
				child := &node{
					path:     wildcard,
					nType:    param,
					handlers: make(map[string]HandlerFunc),
				}

				if wildcard[0] == '*' {
					child.nType = catchAll
				}

				child.handlers[method] = handler
				n.children = append(n.children, child)
				n.wildChild = true
			} else {
				// Wildcard at the start
				n.path = wildcard
				n.nType = param
				if wildcard[0] == '*' {
					n.nType = catchAll
				}
				n.handlers = make(map[string]HandlerFunc)
				n.handlers[method] = handler
			}
		} else {
			// No wildcard, simple static route
			n.path = path
			n.handlers = make(map[string]HandlerFunc)
			n.handlers[method] = handler
		}
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

		// No matching child, check if new path has wildcards
		wildcard, wildcardIndex, valid := findWildcard(remainingPath)

		if wildcardIndex >= 0 {
			// Path has a wildcard, need special handling
			if !valid {
				panic("invalid wildcard in path: " + remainingPath)
			}

			// If wildcard doesn't start at beginning, we need to add a static node first
			if wildcardIndex > 0 {
				// Create a static child for the part before wildcard
				staticChild := &node{
					path:  remainingPath[:wildcardIndex],
					nType: static,
				}
				n.children = append(n.children, staticChild)

				// Now continue from the static child
				remainingPath = remainingPath[wildcardIndex:]
				wildcard, _, _ = findWildcard(remainingPath)

				// Create wildcard as child of the static node
				wildcardChild := &node{
					path:  wildcard,
					nType: param,
				}

				if wildcard[0] == '*' {
					wildcardChild.nType = catchAll
				}

				wildcardChild.handlers = make(map[string]HandlerFunc)
				wildcardChild.handlers[method] = handler
				staticChild.children = append(staticChild.children, wildcardChild)
				staticChild.wildChild = true
				return
			}
			// Create wildcard child
			child := &node{
				path:  wildcard,
				nType: param, // Will be set to catchAll if it starts with *
			}

			// Check if it's a catch-all (*filepath)
			if wildcard[0] == '*' {
				child.nType = catchAll
			}

			child.handlers = make(map[string]HandlerFunc)
			child.handlers[method] = handler
			n.children = append(n.children, child)
			n.wildChild = true
		} else {
			// No wildcard, regular static child
			newChild := &node{
				path:     remainingPath,
				handlers: make(map[string]HandlerFunc),
				nType:    static,
			}
			newChild.handlers[method] = handler
			n.children = append(n.children, newChild)
		}
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

// getValue searches the tree for a matching route
func (n *node) getValue(path string, method string) (HandlerFunc, Params) {
	var params Params

	// Walk through the tree
walk:
	for {
		// If the path is longer than this node's path
		if len(path) > len(n.path) {
			// Check if the node's path is a prefix of the search path
			if path[:len(n.path)] == n.path {
				path = path[len(n.path):] // Remove the matched prefix

				// If this node has wildcard children, check them
				if n.wildChild {
					for _, child := range n.children {
						// Handle parameter nodes (:id)
						if child.nType == param {
							// Find the end of the parameter value
							end := 0
							for end < len(path) && path[end] != '/' {
								end++
							}

							// Extract parameter name (remove the :)
							paramName := child.path[1:]
							paramValue := path[:end]

							// Add to params
							params = append(params, Param{
								Key:   paramName,
								Value: paramValue,
							})

							// Continue with remaining path
							if end < len(path) {
								path = path[end:]
								n = child
								continue walk
							}

							// End of path, check if handler exists
							if handler, ok := child.handlers[method]; ok {
								return handler, params
							}
							return nil, nil
						}

						// Handle catch-all nodes (*filepath)
						if child.nType == catchAll {
							// Extract parameter name (remove the *)
							paramName := child.path[1:]

							// Rest of path is the value
							params = append(params, Param{
								Key:   paramName,
								Value: path,
							})

							if handler, ok := child.handlers[method]; ok {
								return handler, params
							}
							return nil, nil
						}
					}
				}

				// Try to find a matching static child
				for _, child := range n.children {
					if child.nType == static && len(child.path) > 0 && child.path[0] == path[0] {
						n = child
						continue walk
					}
				}

				// No matching child found
				return nil, nil
			}
		}

		// Check if we found an exact match
		if path == n.path {
			if handler, ok := n.handlers[method]; ok {
				return handler, params
			}
			// Path matches but method doesn't
			return nil, nil
		}

		// No match found
		return nil, nil
	}
}

// findWildcard finds the first wildcard segment (:param or *catchall) in the path
// Returns: wildcard string, index where it starts, and whether it's valid
func findWildcard(path string) (wildcard string, i int, valid bool) {
	// Find the first : or *
	for start, c := range []byte(path) {
		if c != ':' && c != '*' {
			continue
		}

		// Found a wildcard
		valid = true

		// Find where the wildcard ends (at / or end of string)
		for end, c := range []byte(path[start+1:]) {
			if c == '/' {
				return path[start : start+1+end], start, valid
			}
		}

		// Wildcard goes to end of path
		return path[start:], start, valid
	}

	return "", -1, false
}
