package transport

import (
	"net/http"

	"example.com/shop/internal/application"
	"example.com/shop/internal/domain"
)

// The handler accepts an explicitly provided client instead of silently
// referring to package globals. The analyzer does not infer runtime behavior.
type Handler struct{ Client *http.Client }

func (Handler) Place(id string) string { return application.Place(domain.Order{ID: id}) }
