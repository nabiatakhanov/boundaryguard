package domain

import "bgstorage/normal" // want "BG001 \\[no-storage\\] bgdirect/domain imports forbidden bgstorage/normal$"

var _ = normal.NormalValue
