package transport

import (
	"net/http"

	"example.com/shop/internal/application"
)

// Capturing the function is enough to violate a symbol reference policy.
var get = http.Get

func Place(id string) []string { return application.Place(id) }
