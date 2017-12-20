package main

import (
	"context"
	"db"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexedwards/scs"
	"github.com/alexedwards/scs/stores/memstore"
	"github.com/uoregon-libraries/gopkg/interrupts"
	"github.com/uoregon-libraries/gopkg/logger"
)

// dbh is our global database handle for DA searches
var dbh = db.New()
var basePath, bind, daRoot string
var sessionManager *scs.Manager

func main() {
	var baseURL string
	baseURL, bind, daRoot = getCLI()

	var s = startServer(baseURL, bind)
	interrupts.TrapIntTerm(func() {
		var ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(time.Minute))
		defer cancel()
		s.Shutdown(ctx)
		os.Exit(0)
	})
	for {
		time.Sleep(time.Second)
	}
}

func startServer(baseURL, bind string) *http.Server {
	var mux = http.NewServeMux()
	var u, err = url.Parse(baseURL)
	if err != nil {
		logger.Fatalf("Unable to parse base URL %q: %s", baseURL, err)
	}

	basePath = strings.TrimRight(u.Path, "/")
	logger.Debugf("Serving root from %q", basePath)
	mux.HandleFunc(basePath+"/", homeHandler)
	mux.HandleFunc(basePath+"/browse/", browseHandler)
	mux.HandleFunc(basePath+"/search/", searchHandler)
	mux.HandleFunc(basePath+"/view/", viewFileHandler)
	mux.HandleFunc(basePath+"/download/", downloadFileHandler)

	var staticPath = filepath.Join(filepath.Dir(os.Args[2]), "static")
	var fileServer = http.FileServer(http.Dir(staticPath))
	var staticPrefix = basePath + "/static/"
	mux.Handle(staticPrefix, http.StripPrefix(staticPrefix, fileServer))

	if basePath == "" {
		basePath = "/"
	}
	initTemplates(basePath)

	// Set up the in-memory session store
	var store = memstore.New(time.Hour * 24)
	sessionManager = scs.NewManager(store)
	sessionManager.Lifetime(time.Hour * 24)
	sessionManager.HttpOnly(false)

	var server = &http.Server{Addr: bind, Handler: sessionManager.Use(mux)}

	go func() {
		logger.Infof("Listening for HTTP connections")
		var err = server.ListenAndServe()
		if err == http.ErrServerClosed {
			logger.Infof("Server terminated")
			return
		}
		if err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Unable to start HTTP server: %s", err)
		}
	}()

	return server
}
