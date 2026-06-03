package req

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestParallelDownloadClosesWorkersOnBadContentLength(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("unexpected body"))
	}))
	defer server.Close()

	pd := C().
		NewParallelDownload(server.URL).
		SetConcurrency(3)

	err := pd.Do()
	if err == nil || !strings.Contains(err.Error(), "bad content length") {
		t.Fatalf("expected bad content length error, got %v", err)
	}

	select {
	case <-pd.doneCh:
	default:
		t.Fatal("parallel download workers were not stopped")
	}
	if _, statErr := os.Stat(pd.tempDir); !os.IsNotExist(statErr) {
		t.Fatalf("parallel download temp dir was not cleaned: %v", statErr)
	}
}
