package req

import (
	"math/rand"
	"net/url"
	"reflect"
	"testing"
)

func TestEncodeQueryParams(t *testing.T) {
	tests := []struct {
		name    string
		client  url.Values
		request url.Values
		want    string
	}{
		{
			name:   "client only",
			client: url.Values{"a": {"1"}, "tag": {"go", "http"}},
			want:   "a=1&tag=go&tag=http",
		},
		{
			name:    "request only",
			request: url.Values{"b": {"2"}},
			want:    "b=2",
		},
		{
			name:    "request overrides client",
			client:  url.Values{"a": {"client"}, "keep": {"1"}},
			request: url.Values{"a": {"request", "second"}},
			want:    "a=request&a=second&keep=1",
		},
		{
			name:    "empty request value removes client value",
			client:  url.Values{"a": {"client"}, "keep": {"1"}},
			request: url.Values{"a": nil},
			want:    "keep=1",
		},
		{
			name:    "keys remain case sensitive",
			client:  url.Values{"Key": {"client"}},
			request: url.Values{"key": {"request"}},
			want:    "Key=client&key=request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientBefore := cloneURLValues(tt.client)
			requestBefore := cloneURLValues(tt.request)
			if got := encodeQueryParams(tt.client, tt.request); got != tt.want {
				t.Fatalf("encodeQueryParams() = %q, want %q", got, tt.want)
			}
			if !reflect.DeepEqual(tt.client, clientBefore) {
				t.Fatalf("client params mutated: got %#v, want %#v", tt.client, clientBefore)
			}
			if !reflect.DeepEqual(tt.request, requestBefore) {
				t.Fatalf("request params mutated: got %#v, want %#v", tt.request, requestBefore)
			}
		})
	}
}

func TestEncodeQueryParamsMatchesLegacyRandomized(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	keys := []string{"a", "A", "space key", "emoji-😀", "reserved&key", "空"}
	values := []string{"", "plain", "with space", "a&b=c", "中文", "+plus"}

	for iteration := 0; iteration < 500; iteration++ {
		clientParams := randomURLValues(rng, keys, values)
		requestParams := randomURLValues(rng, keys, values)
		clientBefore := cloneURLValues(clientParams)
		requestBefore := cloneURLValues(requestParams)

		got := encodeQueryParams(clientParams, requestParams)
		want := legacyEncodeQueryParams(clientParams, requestParams)
		if got != want {
			t.Fatalf("iteration %d: encodeQueryParams() = %q, want legacy encoding %q", iteration, got, want)
		}
		if !reflect.DeepEqual(clientParams, clientBefore) {
			t.Fatalf("iteration %d: client params mutated: got %#v, want %#v", iteration, clientParams, clientBefore)
		}
		if !reflect.DeepEqual(requestParams, requestBefore) {
			t.Fatalf("iteration %d: request params mutated: got %#v, want %#v", iteration, requestParams, requestBefore)
		}
	}
}

func legacyEncodeQueryParams(clientParams, requestParams url.Values) string {
	query := make(url.Values)
	for key, entries := range clientParams {
		for _, entry := range entries {
			query.Add(key, entry)
		}
	}
	for key, entries := range requestParams {
		query.Del(key)
		for _, entry := range entries {
			query.Add(key, entry)
		}
	}
	return query.Encode()
}

func randomURLValues(rng *rand.Rand, keys, values []string) url.Values {
	if rng.Intn(8) == 0 {
		return nil
	}
	result := make(url.Values)
	for _, key := range keys {
		if rng.Intn(2) == 0 {
			continue
		}
		count := rng.Intn(4)
		entries := make([]string, count)
		for i := range entries {
			entries[i] = values[rng.Intn(len(values))]
		}
		result[key] = entries
	}
	return result
}

func cloneURLValues(values url.Values) url.Values {
	if values == nil {
		return nil
	}
	clone := make(url.Values, len(values))
	for key, value := range values {
		if value == nil {
			clone[key] = nil
		} else {
			clone[key] = append([]string{}, value...)
		}
	}
	return clone
}
