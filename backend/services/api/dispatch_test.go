package main

import (
	"net/http"
	"net/http/httptest"
)

// dispatch sends a request through the Echo router so handlers run with the
// same middleware and path-parameter extraction as production.
func dispatch(a *api, rec *httptest.ResponseRecorder, req *http.Request) {
	routes(a).ServeHTTP(rec, req)
}
