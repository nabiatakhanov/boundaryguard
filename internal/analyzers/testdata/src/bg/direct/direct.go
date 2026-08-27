package direct // want package:"reachability\\(v1,module=true;application=bg/direct -> bg/shared/a;storage=bg/direct -> bg/storage\\)"

import (
	_ "bg/shared/a"
	_ "bg/storage" // want "BG001 \\[no-storage\\] bg/direct imports forbidden bg/storage"
)
