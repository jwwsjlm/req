package examples

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	req "github.com/jwwsjlm/req/v3"
)

func TestUploadAndDownload(t *testing.T) {
	payload := []byte("documentation example")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/upload":
			part, err := firstMultipartFile(r, "file")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			defer part.Close()
			body, _ := io.ReadAll(part)
			if !bytes.Equal(body, payload) {
				http.Error(w, "bad file", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case "/download":
			_, _ = w.Write(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := req.C()
	if _, err := client.R().SetFileBytes("file", "note.txt", payload).Post(server.URL + "/upload"); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if _, err := client.R().SetOutput(&output).Get(server.URL + "/download"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(output.Bytes(), payload) {
		t.Fatalf("unexpected download: %q", output.Bytes())
	}
}

func firstMultipartFile(r *http.Request, name string) (multipart.File, error) {
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		return nil, err
	}
	file, _, err := r.FormFile(name)
	return file, err
}
