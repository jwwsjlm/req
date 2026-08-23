package examples

import (
	"net/http"
	"net/http/httptest"
	"testing"

	req "github.com/jwwsjlm/req/v3"
)

func TestAuthAndCookie(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			if got := r.Header.Get("Authorization"); got != "Bearer demo-token" {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc", Path: "/"})
			w.WriteHeader(http.StatusNoContent)
		case "/me":
			cookie, err := r.Cookie("session")
			if err != nil || cookie.Value != "abc" {
				http.Error(w, "missing session", http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte("ok"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := req.C()
	if _, err := client.R().SetBearerAuthToken("demo-token").Post(server.URL + "/login"); err != nil {
		t.Fatal(err)
	}
	resp, err := client.R().Get(server.URL + "/me")
	if err != nil {
		t.Fatal(err)
	}
	if resp.String() != "ok" {
		t.Fatalf("unexpected response: %q", resp.String())
	}
}
