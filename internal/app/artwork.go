package app

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ArtworkMiddleware sert les pochettes exportées depuis Musique sous /artwork/<fichier>.
func ArtworkMiddleware(dir string) application.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			name, ok := strings.CutPrefix(r.URL.Path, "/artwork/")
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			if name == "" || name != filepath.Base(name) || strings.Contains(name, "..") {
				http.NotFound(w, r)
				return
			}
			data, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", http.DetectContentType(data))
			w.Header().Set("Cache-Control", "max-age=86400")
			w.Write(data)
		})
	}
}
