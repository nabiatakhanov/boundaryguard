package transitive

import "example.com/shop/internal/shared"

func Place(id string) string { return shared.Save(id) }
