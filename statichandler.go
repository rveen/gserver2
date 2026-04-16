package gserver2

import (
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/rveen/golib/fn"
)

// StaticFileHandler returns a handler that serves static files.
//
// If protect is true, an authenticated user is required.
// If fs is non-nil it is used as the file tree; otherwise srv.Root is used.
// In multi-host mode (srv.Multi) the hostname is prepended to the path.
func (srv *Server) StaticFileHandler(protect bool, fs *fn.FNode) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		path := r.URL.Path
		if srv.Multi {
			path = r.Host + "/" + path
		}

		log.Println("StaticHandler", path, r.RemoteAddr)

		if protect {
			u := UserCookieValue(r)
			if (u == "" || u == "nobody") && srv.DefaultUser == "" {
				http.Error(w, "Need to log in to access this content", 401)
				return
			}
		}

		var fd fn.FNode
		if fs != nil {
			fd = *fs
		} else {
			fd = *srv.Root
		}
		file := &fd

		// Phase 1: resolve path without reading content
		if err := file.GetMeta(path); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// Phase 2a: native-FS plain file — stream directly via http.ServeContent
		if file.Type == "file" && file.RootFs == nil {
			f, err := os.Open(file.Path)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			defer f.Close()
			stat, err := f.Stat()
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			ext := filepath.Ext(file.Path)
			w.Header().Set("Content-Type", mime.TypeByExtension(ext))
			w.Header().Set("Cache-Control", "public, max-age=7200")
			http.ServeContent(w, r, filepath.Base(file.Path), stat.ModTime(), f)
			return
		}

		// Phase 2b: embedded FS or document/data — full read
		if err := file.Get(path); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if len(file.Content) == 0 {
			http.Error(w, "Zero length file", 500)
			return
		}
		ext := filepath.Ext(file.Path)
		w.Header().Set("Content-Type", mime.TypeByExtension(ext))
		w.Header().Set("Cache-Control", "public, max-age=7200")
		w.Write(file.Content)
	}
}
