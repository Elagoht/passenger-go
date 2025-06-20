package cache

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func StaticETagMiddleware(staticDir string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if !strings.HasPrefix(request.URL.Path, "/static/") {
				next.ServeHTTP(writer, request)
				return
			}

			filePath := strings.TrimPrefix(request.URL.Path, "/static/")
			fullPath := filepath.Join(staticDir, filePath)

			fileInfo, err := os.Stat(fullPath)
			if err != nil {
				next.ServeHTTP(writer, request)
				return
			}

			etag := generateETag(fileInfo)

			if match := request.Header.Get("If-None-Match"); match != "" {
				if strings.Contains(match, etag) {
					writer.WriteHeader(http.StatusNotModified)
					return
				}
			}

			writer.Header().Set("ETag", fmt.Sprintf(`"%s"`, etag))

			setCacheHeaders(writer, filePath)

			next.ServeHTTP(writer, request)
		})
	}
}

func generateETag(fileInfo os.FileInfo) string {
	modTime := fileInfo.ModTime().Unix()
	size := fileInfo.Size()

	hash := md5.Sum(fmt.Appendf([]byte{}, "%d-%d", modTime, size))
	return hex.EncodeToString(hash[:8])
}

func setCacheHeaders(w http.ResponseWriter, filePath string) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".css", ".js":
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	case ".png", ".webp", ".ico", ".svg":
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	case ".json":
		w.Header().Set("Cache-Control", "public, max-age=3600")
	default:
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}
}
