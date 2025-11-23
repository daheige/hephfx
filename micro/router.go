package micro

import (
	"net/http"
)

// Route represents the route for mux
type Route struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
}
