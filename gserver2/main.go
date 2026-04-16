// Copyright 2017-2022, Rolf Veen.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Gserver is a web server.
//
// Summary of features
//
//   - Any path of the form /[user]/file/* is served as static (StaticHandler) and
//     converted to /files/user/*
//   - Any other path is handled by Handler as follows.
//   - Path elements of the form @rev are taken as revisions.
//   - Path elements of the form _t (t != number) are taken as variables
//   - Extensions of files are optional (if the file name is unique)
//   - index.* (if found) is returned for paths that point to directories.
//   - OGDL templates are processed
//   - Markdown is processed
//   - The root directory must be a standard directory. Below there can be versioned
//     repositories
//   - The path can continue into data files and documents (markdown)
//
// # Authentication and sessions
//
// - htpasswd, SVN Auth, ACL
package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"

	"github.com/go-chi/chi/v5"
	"github.com/rveen/golib/fn"
	"github.com/rveen/gserver2"
)

func main() {

	var logging, verbose, hosts bool
	var host, userdb string
	var timeout, sessionTimeout int

	flag.BoolVar(&logging, "log", true, "turn logging ON/off")
	flag.BoolVar(&hosts, "m", false, "enable multiple hosts (path on disk are affected)")
	flag.BoolVar(&verbose, "v", false, "turn periodic status message on/OFF")
	flag.StringVar(&host, "H", ":80", "set host:port")
	flag.IntVar(&timeout, "t", 10, "set http timeout (seconds)")
	flag.IntVar(&sessionTimeout, "ts", 30, "set session timeout (minutes)")
	flag.StringVar(&userdb, "userdb", "htaccess", "user db: sqlite or htaccess (default)")

	flag.Parse()

	var srv *gserver2.Server
	var err error

	if !hosts {
		srv, err = gserver2.New(host)
	} else {
		srv, err = gserver2.NewMulti()
	}
	if err != nil {
		log.Println(err.Error())
		return
	}

	go srv.WatchContext(".conf/context.ogdl")

	staticHandler := srv.StaticFileHandler(false, nil)
	fileHandler := http.FileServer(http.Dir("."))

	r := chi.NewRouter()
	r.Get("/favicon.ico", staticHandler)
	r.Handle("/files/*", fileHandler)
	r.Handle("/static/*", staticHandler)
	r.Handle("/file/*", staticHandler)
	r.With(srv.LoginAdapter(userdb)).Handle("/*", srv.DynamicHandler(nil))
	// To re-enable ACL: r.With(srv.LoginAdapter(userdb), gserver2.AccessAdapter(".conf/acl.conf")).Handle("/*", ...)

	log.Println("gserver starting, ", runtime.NumCPU(), "procs")

	if !logging {
		println("further logging disabled!")
		log.SetOutput(ioutil.Discard)
	}

	// Overwrite the default file handler with the configured doc root
	srv.Root = fn.New(srv.DocRoot)

	srv.Serve(timeout, r)
}
