package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestArtworkMiddleware(t *testing.T) {
	dir := t.TempDir()
	png := []byte("\x89PNG\r\n\x1a\n0000")
	os.WriteFile(filepath.Join(dir, "abc.img"), png, 0o644)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(299) })
	h := ArtworkMiddleware(dir)(next)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/artwork/abc.img", nil))
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "image/png" {
		t.Errorf("code=%d type=%s", rec.Code, rec.Header().Get("Content-Type"))
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/artwork/../settings.json", nil))
	if rec.Code != 404 {
		t.Errorf("traversée de chemin doit donner 404, obtenu %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/index.html", nil))
	if rec.Code != 299 {
		t.Errorf("les autres chemins doivent passer au handler suivant")
	}
}
