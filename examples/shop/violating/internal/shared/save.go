package shared

import "example.com/shop/internal/storage"

func Save(id string) string { return storage.Save(id) }
