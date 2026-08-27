package bgroot_test // want package:"reachability\\(v1,module=true;storage=bgroot_test -> bgroot/unknown -> bgroot/storage\\)"

import _ "bgroot/unknown" // want "BG003 \\[root-test-storage\\] forbidden dependency path: bgroot_test -> bgroot/unknown -> bgroot/storage"
