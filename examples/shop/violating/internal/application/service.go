package application

import (
	"example.com/shop/internal/domain/direct"
	"example.com/shop/internal/domain/transitive"
)

func Place(id string) []string { return []string{direct.Place(id), transitive.Place(id)} }
