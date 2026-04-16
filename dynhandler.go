package gserver2

import (
	"bytes"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/rveen/golib/fn"
	"github.com/rveen/ogdl"
)

// DynamicHandler returns a handler that processes dynamic (template/data) requests.
//
// If fs is non-nil it is tried first; on miss the handler falls back to srv.Root.
// If fs is nil, srv.Root is used and path-level auth is enforced via checkPath.
func (srv *Server) DynamicHandler(fs *fn.FNode) http.HandlerFunc {

	return func(w http.ResponseWriter, rh *http.Request) {

		log.Println("DynHandler", rh.URL.Path, rh.RemoteAddr)
		t := time.Now().UnixMicro()

		r := ConvertRequest(rh, w, srv)
		if r == nil {
			http.Error(w, "Number of open sessions exceeded", 429)
			return
		}

		if rh.FormValue("UploadFiles") != "" {
			gf, _ := fileUpload(rh, "")
			data := r.Context.Node("R")
			files := data.Add("files")
			files.Add(gf)
		}

		if fs != nil {
			// Custom FNode: try fs first, fall back to srv.Root
			fd := *fs
			r.File = &fd
			if err := r.Get(); err != nil {
				f := *srv.Root
				r.File = &f
				if err = r.Get(); err != nil {
					http.Error(w, http.StatusText(404), 404)
					return
				}
			}
		} else {
			// Standard: enforce path-level auth, serve from srv.Root
			user := r.Context.Node("user").String()
			if (user == "" || user == "nobody") && !checkPath(r.Path, srv.Config) {
				http.Redirect(w, rh, "/login?redirect="+rh.URL.Path, 302)
				return
			}
			if err := r.Get(); err != nil {
				http.Error(w, http.StatusText(404), 404)
				return
			}
		}

		r.Process(srv)

		w.Header().Set("Content-Type", r.Mime)

		if rh.FormValue("filename") != "" {
			ext := filepath.Ext(r.Path)
			if ext != "" {
				fname := strings.TrimSpace(rh.FormValue("filename"))
				w.Header().Set("Content-Disposition", "inline; filename=\""+fname+ext+"\"")
			}
		}

		if len(r.File.Content) == 0 {
			http.Error(w, "Empty content", 500)
		} else {
			http.ServeContent(w, rh, filepath.Base(r.Path), time.Time{}, bytes.NewReader(r.File.Content))
		}
		log.Printf("DynHandler END %d us\n", time.Now().UnixMicro()-t)
	}
}

func checkPath(path string, cfg *ogdl.Graph) bool {

	if cfg == nil {
		return true
	}

	g := cfg.Node("allowed")
	if g != nil {
		for _, gp := range g.Out {
			if strings.HasPrefix(path, gp.ThisString()) {
				return true
			}
		}
	}

	g = cfg.Node("protected")
	if g == nil {
		return true
	}
	for _, gp := range g.Out {
		if strings.HasPrefix(path, gp.ThisString()) {
			return false
		}
	}
	return true
}
