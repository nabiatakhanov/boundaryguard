package dot

import . "net/http"

var captured = Get         // want "BG002 \\[no-get\\] bgsymbols/dot references forbidden symbol net/http.Get$"
var client = DefaultClient // want "BG002 \\[no-default\\] bgsymbols/dot references forbidden symbol net/http.DefaultClient$"

func references() {
	_, _ = Get("https://example.invalid") // want "BG002 \\[no-get\\] bgsymbols/dot references forbidden symbol net/http.Get$"
}
