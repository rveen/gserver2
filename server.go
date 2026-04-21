package gserver2

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/rveen/golib/fn"
	"github.com/rveen/ogdl"
	rpc "github.com/rveen/ogdl/ogdlrf"
	"github.com/rveen/session2"
)

type Server struct {
	Host         string
	Hosts        []string
	Config       *ogdl.Graph
	HostContexts map[string]*ogdl.Graph
	Context      *ogdl.Graph
	Root         *fn.FNode
	DocRoot      string
	UploadDir    string
	DefaultUser  string
	UserDb       *sql.DB
	MaxSessions  int
	Multi        bool
	Templates    map[string]*ogdl.Graph
	ContextMu    sync.RWMutex
	server       *http.Server
	Secret       []byte // Used as key for encoding persistent user cookies
}

func NewWithConfig(host string, config, context *ogdl.Graph) (*Server, error) {

	srv := Server{}
	srv.DocRoot = "./"
	srv.UploadDir = "files/"
	srv.Host = host
	srv.Config = config
	srv.Context = context

	// Preload templates
	tpls := srv.Config.Get("templates")
	srv.Templates = make(map[string]*ogdl.Graph)
	if tpls.Len() > 0 {
		for _, tpl := range tpls.Out {
			srv.Templates[tpl.ThisString()] = ogdl.NewTemplate(tpl.String())
		}
	}

	// Register remote functions
	rfs := srv.Config.Get("ogdlrf")
	if rfs != nil {
		for _, rf := range rfs.Out {
			name := rf.ThisString()
			host := rf.Get("host").String()
			proto := rf.Get("protocol").Int64(2)
			log.Println("remote function registered:", name, host, proto)
			f := rpc.Client{Host: host, Timeout: 1, Protocol: int(proto)}
			srv.Context.Set(name, f.Call)
		}
	}

	srv.Hosts = append(srv.Hosts, srv.Host)
	srv.InitSessions()

	GlobalContext(&srv)

	return &srv, nil
}

func (srv *Server) InitSessions() {
	session2.Reinit()
	srv.MaxSessions = 10000
}

// New prepares a Server initialized from .conf/config.ogdl and .conf/context.ogdl.
func New(host string) (*Server, error) {
	config := ogdl.FromFile(".conf/config.ogdl")
	if config == nil {
		return nil, errors.New("missing .conf/config.ogdl file")
	}
	context := ogdl.FromFile(".conf/context.ogdl")
	if context == nil {
		return nil, errors.New("missing .conf/context.ogdl file")
	}
	return NewWithConfig(host, config, context)
}

// NewMulti prepares a Server for virtual-host mode. Each subdirectory that
// looks like a hostname gets its own context loaded from <host>/.conf/context.ogdl.
func NewMulti() (*Server, error) {
	config := ogdl.FromFile(".conf/config.ogdl")
	if config == nil {
		config = ogdl.New(nil)
	}

	srv, err := NewWithConfig(":80", config, ogdl.New(nil))
	if err != nil {
		return nil, err
	}
	srv.Multi = true

	entries, _ := os.ReadDir(".")
	srv.HostContexts = make(map[string]*ogdl.Graph)
	for _, f := range entries {
		name := f.Name()
		if name[0] == '.' || name[0] == '_' || !strings.Contains(name, ".") {
			continue
		}
		if !f.IsDir() {
			continue
		}
		srv.HostContexts[name] = ogdl.FromFile(name + "/.conf/context.ogdl")
		log.Println("context loaded for host", name)
		srv.Hosts = append(srv.Hosts, name)
	}

	GlobalContext(srv)

	return srv, nil
}

func (srv *Server) Serve(timeout int, router http.Handler) {
	if timeout == 0 {
		timeout = 30
	}
	if srv.Host == "" {
		srv.Host = ":80"
	}
	server := &http.Server{
		Addr:              srv.Host,
		Handler:           router,
		ReadTimeout:       time.Second * time.Duration(timeout),
		WriteTimeout:      time.Second * time.Duration(timeout),
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: time.Second * time.Duration(timeout),
	}
	log.Println("gserver starting on", srv.Host)
	srv.server = server
	server.ListenAndServe()
}

func (srv *Server) Shutdown() {
	log.Println("Shutting down server")
	if srv.server != nil {
		srv.server.Shutdown(context.Background())
	}
}

// WatchContext watches the given context.ogdl file and reloads srv.Context
// whenever the file is written or replaced. Intended to be run as a goroutine.
func (srv *Server) WatchContext(path string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println("fsnotify: cannot create watcher:", err)
		return
	}
	defer watcher.Close()

	if err := watcher.Add(path); err != nil {
		log.Println("fsnotify: cannot watch", path, ":", err)
		return
	}
	log.Println("watching", path)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				newCtx := ogdl.FromFile(path)
				if newCtx == nil {
					log.Println("context reload failed: invalid file", path)
					continue
				}
				srv.ContextMu.Lock()
				srv.Context = newCtx
				srv.InitSessions()
				GlobalContext(srv)
				srv.ContextMu.Unlock()

				log.Println("context reloaded from", path)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("fsnotify error:", err)
		}
	}
}
