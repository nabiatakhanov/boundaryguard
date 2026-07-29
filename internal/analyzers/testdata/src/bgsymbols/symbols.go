package bgsymbols

import (
	"bgother"
	"log"
	net "net/http"
)

var Get = func() {}
var captured = net.Get         // want "BG002 \\[no-get\\] bgsymbols references forbidden symbol net/http.Get$"
var client = net.DefaultClient // want "BG002 \\[no-default\\] bgsymbols references forbidden symbol net/http.DefaultClient$"

func references() {
	_, _ = net.Get("https://example.invalid") // want "BG002 \\[no-get\\] bgsymbols references forbidden symbol net/http.Get$"
	get := net.Get                            // want "BG002 \\[no-get\\] bgsymbols references forbidden symbol net/http.Get$"
	_ = get
	log.Fatal("failed") // want "BG002 \\[no-fatal\\] bgsymbols references forbidden symbol log.Fatal$"
}

type localClient struct{}

func (localClient) Get() {}

func clean() {
	http := struct{ Get func() }{Get: func() {}}
	http.Get()
	Get := func() {}
	Get()
	bgother.Get()
	_ = bgother.DefaultClient
	localClient{}.Get()
	_ = &net.Client{}
}
