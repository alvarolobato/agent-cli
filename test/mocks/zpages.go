package mocks

import "net/http"

// ZPagesHandler serves a tiny zpages-like HTML response.
func ZPagesHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body><h1>pipelinez</h1><p>ok</p></body></html>"))
	})
}
