package domainunknown // want package:"reachability\\(v1,module=true;storage=bg/domainunknown -> bg/unknown -> bg/storage\\)"

import _ "bg/unknown" // want "BG003 \\[no-storage\\] forbidden dependency path: bg/domainunknown -> bg/unknown -> bg/storage"
