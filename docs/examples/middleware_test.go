package examples

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	req "github.com/jwwsjlm/req/v3"
)

func TestMiddleware(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Request-ID") == "" {
			http.Error(w, "missing request ID", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := req.C().
		OnBeforeRequest(func(_ *req.Client, r *req.Request) error {
			r.SetHeader("X-Request-ID", fmt.Sprint(time.Now().UnixNano()))
			return nil
		}).
		OnAfterResponse(func(_ *req.Client, resp *req.Response) error {
			if resp.Err == nil && resp.GetStatusCode() >= 500 {
				resp.Err = fmt.Errorf("remote service failed: %s", resp.GetStatus())
			}
			return nil
		})

	if _, err := client.R().Get(server.URL); err != nil {
		t.Fatal(err)
	}
}
