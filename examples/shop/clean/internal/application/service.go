package application

import (
	"example.com/shop/internal/domain"
	"example.com/shop/internal/shared"
)

func Place(order domain.Order) string { return shared.Save(order.ID) }
