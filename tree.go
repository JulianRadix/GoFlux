package goflux

import (
	"net/http"
)

// HandlerFunc defines the handler used by GoFlux
// It receives the response writer, request and any URL parameters
type HandlerFunc func(http.ResponseWriter, *http.Request, Params)

// Params is a slice of parameters key-value pairs extracted from the URL
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
	nType    nodeType               // what type of node this is
	children []*node                // child nodes
}
