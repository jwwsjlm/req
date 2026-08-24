package examples

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	req "github.com/jwwsjlm/req/v3"
)

type beginnerUser struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type beginnerAPIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type beginnerAPIClient struct {
	http *req.Client
}

// newBeginnerAPIClient creates the small business client used by the beginner guide.
// newBeginnerAPIClient 创建新手文档使用的简易业务客户端。
func newBeginnerAPIClient(baseURL string) *beginnerAPIClient {
	return &beginnerAPIClient{
		http: req.C().
			SetBaseURL(baseURL).
			SetTimeout(10*time.Second).
			SetCommonHeader("Accept", "application/json"),
	}
}

// getUser demonstrates context propagation, path parameters, typed success
// results, typed error results, and separate HTTP status handling.
// getUser 演示 Context 传递、路径参数、成功/失败结果解析和 HTTP 状态处理。
func (api *beginnerAPIClient) getUser(ctx context.Context, id int) (beginnerUser, error) {
	var user beginnerUser
	var apiErr beginnerAPIError

	resp, err := api.http.R().
		SetContext(ctx).
		SetPathParam("id", strconv.Itoa(id)).
		SetSuccessResult(&user).
		SetErrorResult(&apiErr).
		Get("/users/{id}")
	if err != nil {
		return beginnerUser{}, fmt.Errorf("get user: %w", err)
	}
	if !resp.IsSuccessState() {
		if apiErr.Message != "" {
			return beginnerUser{}, fmt.Errorf("get user: %s: %s", resp.GetStatus(), apiErr.Message)
		}
		return beginnerUser{}, fmt.Errorf("get user: %s", resp.GetStatus())
	}

	return user, nil
}

// createUser demonstrates a JSON request body and a typed JSON response.
// createUser 演示 JSON 请求体和带类型的 JSON 响应。
func (api *beginnerAPIClient) createUser(ctx context.Context, name string) (beginnerUser, error) {
	var user beginnerUser
	resp, err := api.http.R().
		SetContext(ctx).
		SetBodyJsonMarshal(beginnerUser{Name: name}).
		SetSuccessResult(&user).
		Post("/users")
	if err != nil {
		return beginnerUser{}, fmt.Errorf("create user: %w", err)
	}
	if !resp.IsSuccessState() {
		return beginnerUser{}, fmt.Errorf("create user: %s", resp.GetStatus())
	}
	return user, nil
}

func TestBeginnerAPIClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/users" && r.Method == http.MethodPost {
			var user beginnerUser
			if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(beginnerUser{ID: 43, Name: user.Name})
			return
		}
		if r.URL.Path == "/users/99" {
			select {
			case <-time.After(200 * time.Millisecond):
				_ = json.NewEncoder(w).Encode(beginnerUser{ID: 99, Name: "Slow"})
			case <-r.Context().Done():
			}
			return
		}
		if r.URL.Path != "/users/42" || r.Method != http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":"not_found","message":"user does not exist"}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":42,"name":"Ada"}`))
	}))
	defer server.Close()

	api := newBeginnerAPIClient(server.URL)

	t.Run("success", func(t *testing.T) {
		user, err := api.getUser(context.Background(), 42)
		if err != nil {
			t.Fatal(err)
		}
		if user.ID != 42 || user.Name != "Ada" {
			t.Fatalf("unexpected user: %#v", user)
		}
	})

	t.Run("HTTP error", func(t *testing.T) {
		_, err := api.getUser(context.Background(), 7)
		if err == nil {
			t.Fatal("expected an HTTP status error")
		}
	})

	t.Run("POST JSON", func(t *testing.T) {
		user, err := api.createUser(context.Background(), "Grace")
		if err != nil {
			t.Fatal(err)
		}
		if user.ID != 43 || user.Name != "Grace" {
			t.Fatalf("unexpected created user: %#v", user)
		}
	})

	t.Run("context timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		_, err := api.getUser(ctx, 99)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected context deadline exceeded, got %v", err)
		}
	})
}
