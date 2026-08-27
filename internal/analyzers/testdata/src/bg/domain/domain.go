package domain // want package:"reachability\\(v1,module=true;application=bg/domain -> bg/shared/a;storage=bg/domain -> bg/shared/a -> bg/storage\\)"

import (
	_ "bg/shared/a" // want "BG003 \\[no-storage\\] forbidden dependency path: bg/domain -> bg/shared/a -> bg/storage"
	_ "bg/shared/b"
)
