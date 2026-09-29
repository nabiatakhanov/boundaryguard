package direct

import "example.com/shop/internal/storage"

func Place(id string) string { return storage.Save(id) }
