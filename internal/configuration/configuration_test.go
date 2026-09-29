package configuration

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

const valid = `{
  "version": 1,
  "layers": [
    {"name":"domain","packages":["example.com/shop/internal/domain/..."]},
    {"name":"storage","packages":["example.com/shop/internal/storage/..."]}
  ],
  "dependency_rules": [
    {"id":"domain-storage","from_layer":"domain","to_layer":"storage","kind":"transitive"}
  ],
  "forbidden_symbols": [
    {"id":"no-get","packages":["example.com/shop/..."],"symbol":"net/http.Get"}
  ]
}`

func TestParseAndMatch(t *testing.T) {
	t.Parallel()
	c, err := Parse([]byte(valid))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ path, want string }{
		{"example.com/shop/internal/domain", "domain"},
		{"example.com/shop/internal/domain/model", "domain"},
		{"example.com/shop/internal/domainish", ""},
		{"example.com/shop/internal/storage/sql", "storage"},
		{"example.com/other", ""},
	} {
		if got := c.LayerFor(tc.path); got != tc.want {
			t.Errorf("LayerFor(%q)=%q, want %q", tc.path, got, tc.want)
		}
	}
	if Match("net/http", "net/http/httptest") || !Match("net/http", "net/http") || Match("net/http/...", "net/httpish") {
		t.Fatal("exact and recursive matching disagree with segment boundaries")
	}
	if len(c.Fingerprint()) != 64 {
		t.Errorf("fingerprint = %q", c.Fingerprint())
	}
}

func TestInvalidConfiguration(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, input, contains string }{
		{"syntax", `{`, "JSON"},
		{"unknown", `{"version":1,"layres":[]}`, "unknown field"},
		{"field-case", `{"Version":1}`, "unknown field"},
		{"duplicate-key", `{"version":1,"version":1}`, "duplicate JSON key"},
		{"nested-duplicate-key", strings.Replace(valid, `"name":"domain"`, `"name":"domain","name":"domain"`, 1), "duplicate JSON key"},
		{"trailing", valid + ` {}`, "multiple JSON values"},
		{"null", `null`, "version"},
		{"version", strings.Replace(valid, `"version": 1`, `"version": 2`, 1), "version"},
		{"missing-layer", strings.Replace(valid, `"from_layer":"domain"`, `"from_layer":"ghost"`, 1), "missing layer"},
		{"kind", strings.Replace(valid, `"kind":"transitive"`, `"kind":"recursive"`, 1), "kind"},
		{"duplicate-layer", strings.Replace(valid, `"name":"storage"`, `"name":"domain"`, 1), "duplicate layer"},
		{"duplicate-id", strings.Replace(valid, `"id":"no-get"`, `"id":"domain-storage"`, 1), "duplicate rule id"},
		{"empty-id", strings.Replace(valid, `"id":"no-get"`, `"id":""`, 1), "invalid rule id"},
		{"empty-layer-name", strings.Replace(valid, `"name":"domain"`, `"name":""`, 1), "invalid layer name"},
		{"empty-patterns", strings.Replace(valid, `["example.com/shop/internal/domain/..."]`, `[]`, 1), "packages"},
		{"symbol-no-package", strings.Replace(valid, `net/http.Get`, `Get`, 1), "symbol"},
		{"symbol-no-name", strings.Replace(valid, `net/http.Get`, `net/http.`, 1), "symbol"},
		{"symbol-keyword", strings.Replace(valid, `net/http.Get`, `net/http.func`, 1), "symbol"},
		{"symbol-underscore", strings.Replace(valid, `net/http.Get`, `net/http._`, 1), "symbol"},
		{"symbol-method-syntax", strings.Replace(valid, `net/http.Get`, `net/http.(*Client).Get`, 1), "symbol"},
		{"rule-unknown", strings.Replace(valid, `"symbol":"net/http.Get"`, `"symbol":"net/http.Get","enabled":true`, 1), "unknown field"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.input))
			if err == nil || !strings.Contains(err.Error(), tc.contains) {
				t.Fatalf("error=%v, want containing %q", err, tc.contains)
			}
		})
	}
}

func TestPatternsAndOverlap(t *testing.T) {
	t.Parallel()
	for _, bad := range []string{"", "...", "./foo", "/foo", "foo/", "foo//bar", "foo/../bar", "foo/*", "foo/**", "foo/.../bar", "foo...bar", "foo\\bar", "foo bar", "foo@v1", "foo?bar", "[foo]"} {
		p := policy{Version: 1, Layers: []Layer{{Name: "a", Packages: []string{bad}}}}
		input, _ := json.Marshal(p)
		if _, err := Parse(input); err == nil {
			t.Errorf("accepted unsupported pattern %q", bad)
		}
	}
	for _, tc := range []struct {
		a, b    string
		overlap bool
	}{
		{"example.com/p/...", "example.com/p", true},
		{"example.com/p/...", "example.com/p/sub/...", true},
		{"example.com/p", "example.com/p", true},
		{"example.com/p", "example.com/p/sub/...", false},
		{"example.com/p/...", "example.com/pish/...", false},
		{"example.com/p/a/...", "example.com/p/b/...", false},
	} {
		p := policy{Version: 1, Layers: []Layer{{Name: "a", Packages: []string{tc.a}}, {Name: "b", Packages: []string{tc.b}}}}
		input, _ := json.Marshal(p)
		_, err := Parse(input)
		if (err != nil) != tc.overlap {
			t.Errorf("patterns %s, %s: error %v; overlap=%v", tc.a, tc.b, err, tc.overlap)
		}
	}
	_, err := Parse([]byte(`{"version":1,"layers":[{"name":"a","packages":["net/http","net/http"]}]}`))
	if err == nil {
		t.Fatal("accepted duplicate pattern")
	}
}

func TestFingerprintAndIsolation(t *testing.T) {
	t.Parallel()
	a, err := Parse([]byte(valid))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(valid), &raw); err != nil {
		t.Fatal(err)
	}
	layers := raw["layers"].([]any)
	layers[0], layers[1] = layers[1], layers[0]
	reordered, _ := json.Marshal(raw)
	b, err := Parse(reordered)
	if err != nil {
		t.Fatal(err)
	}
	if a.Fingerprint() != b.Fingerprint() {
		t.Fatal("order changed semantic fingerprint")
	}
	c, err := Parse([]byte(strings.Replace(valid, "net/http.Get", "net/http.DefaultClient", 1)))
	if err != nil {
		t.Fatal(err)
	}
	if a.Fingerprint() == c.Fingerprint() {
		t.Fatal("symbol policy did not change fingerprint")
	}
	d, err := Parse([]byte(`{"version":1}`))
	if err != nil {
		t.Fatal(err)
	}
	e, err := Parse([]byte(`{"version":1,"layers":[],"dependency_rules":[],"forbidden_symbols":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	if d.Fingerprint() != e.Fingerprint() {
		t.Fatal("empty arrays changed semantic fingerprint")
	}
	before := a.Fingerprint()
	ls := a.Layers()
	ls[0].Packages[0] = "changed"
	ls[0].Name = "changed"
	ds := a.DependencyRules()
	ds[0].ID = "changed"
	ss := a.ForbiddenSymbols()
	ss[0].Packages[0] = "changed"
	ss[0].Symbol = "changed"
	if a.Fingerprint() != before || a.LayerFor("example.com/shop/internal/domain") != "domain" || a.ForbiddenSymbols()[0].Symbol != "net/http.Get" || a.DependencyRules()[0].ID != "domain-storage" {
		t.Fatal("getter mutation escaped into immutable config")
	}
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				_ = a.Layers()
				_ = a.LayerFor("example.com/shop/internal/domain")
				_ = a.Fingerprint()
				_ = a.ForbiddenSymbols()
			}
		}()
	}
	wg.Wait()
}

func TestLoad(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "boundaryguard.json")
	if err := os.WriteFile(path, []byte(valid), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(""); err == nil {
		t.Fatal("accepted missing -config")
	}
	if _, err := Load(path + ".missing"); err == nil || !strings.Contains(err.Error(), strconv.Quote(path+".missing")) {
		t.Fatalf("missing path error = %v", err)
	}
	if err := os.WriteFile(path, []byte(`{`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), strconv.Quote(path)) {
		t.Fatalf("invalid file error = %v", err)
	}
}

func TestCheckedInFixtures(t *testing.T) {
	t.Parallel()
	for _, path := range []string{filepath.Join("config", "valid.json"), "durableq.json"} {
		good, err := os.ReadFile(filepath.Join("..", "..", "testdata", path))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Parse(good); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
	}
	bad, err := os.ReadFile(filepath.Join("..", "..", "testdata", "config", "overlapping.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(bad); err == nil {
		t.Fatal("accepted invalid checked-in example")
	}
}

func FuzzParse(f *testing.F) {
	for _, seed := range []string{valid, `{}`, `null`, `{"version":1}`, `{"version":1,"version":2}`, `[[]]`, `{"layers":[{"packages":["foo/..."]}],"version":1}`} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		c, err := Parse(data)
		if err != nil {
			if c != nil {
				t.Fatal("error returned with nonnil config")
			}
			return
		}
		if c == nil || len(c.Fingerprint()) != 64 {
			t.Fatal("success without valid config")
		}
		encoded, err := json.Marshal(c.policy)
		if err != nil {
			t.Fatal(err)
		}
		again, err := Parse(encoded)
		if err != nil || c.Fingerprint() != again.Fingerprint() {
			t.Fatalf("roundtrip failed: %v", err)
		}
		if !bytes.Equal(encoded, mustJSON(t, again.policy)) {
			t.Fatal("normalization is not idempotent")
		}
	})
}

func mustJSON(t *testing.T, p policy) []byte {
	t.Helper()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
