package examples

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	req "github.com/jwwsjlm/req/v3"
)

type user struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// fetchUser demonstrates the usual Client -> Request -> Response flow.
// fetchUser 展示 Client、Request、Response 的常规调用链。
func fetchUser(ctx context.Context, client *req.Client, baseURL string, id int) (user, error) {
	var result user
	resp, err := client.R().
		SetContext(ctx).
		SetPathParam("id", fmt.Sprint(id)).
		SetQueryParam("expand", "profile").
		SetHeader("Accept", "application/json").
		SetSuccessResult(&result).
		Get(baseURL + "/users/{id}")
	if err != nil {
		return user{}, err
	}
	if !resp.IsSuccessState() {
		return user{}, fmt.Errorf("unexpected HTTP status: %s", resp.GetStatus())
	}
	return result, nil
}

func TestBasic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/42" || r.URL.Query().Get("expand") != "profile" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":42,"name":"Ada"}`))
	}))
	defer server.Close()

	got, err := fetchUser(context.Background(), req.C(), server.URL, 42)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 42 || got.Name != "Ada" {
		t.Fatalf("unexpected user: %#v", got)
	}
}
