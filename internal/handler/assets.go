package handler

import (
	"io/fs"
	"net/http"
	"strings"
)

const assetPrefix = "/assets/"

// Assets serves public files from the embedded web filesystem under /assets/.
// Template files and directories are intentionally not exposed.
func Assets(assets fs.FS) http.Handler {
	if assets == nil {
		return http.NotFoundHandler()
	}

	fileServer := http.FileServerFS(assets)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name, ok := strings.CutPrefix(r.URL.Path, assetPrefix)
		if !ok || !publicAssetPath(name) {
			http.NotFound(w, r)
			return
		}

		info, err := fs.Stat(assets, name)
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}

		request := r.Clone(r.Context())
		url := *r.URL
		request.URL = &url
		request.URL.Path = "/" + name

		w.Header().Set("Cache-Control", "public, max-age=3600")
		fileServer.ServeHTTP(w, request)
	})
}

// publicAssetPath reports whether name is a safe public asset path.
func publicAssetPath(name string) bool {
	if name == "" || !fs.ValidPath(name) {
		return false
	}

	return !strings.HasSuffix(name, ".gohtml")
}
