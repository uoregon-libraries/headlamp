package main

import (
	"config"
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
var basePath string
var conf *config.Config
var sessionManager *scs.Manager

func main() {
	conf = getCLI()

	var s = startServer()
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

func startServer() *http.Server {
	var mux = http.NewServeMux()
	var u, _ = url.Parse(conf.WebPath)

	basePath = strings.TrimRight(u.Path, "/")
	logger.Debugf("Serving root from %q", basePath)
	mux.HandleFunc(basePath+"/", homeHandler)
	mux.HandleFunc(basePath+"/browse/", browseHandler)
	mux.HandleFunc(basePath+"/search/", searchHandler)
	mux.HandleFunc(basePath+"/view/", viewFileHandler)
	mux.HandleFunc(basePath+"/download/", downloadFileHandler)
	mux.HandleFunc(basePath+"/bulk/", bulkQueueHandler)
	mux.HandleFunc(basePath+"/bulk/create", bulkCreateArchiveHandler)
	mux.HandleFunc(basePath+"/bulk-download/", bulkDownloadHandler)

	var staticPath = filepath.Join(conf.Approot, "static")
	var fileServer = http.FileServer(http.Dir(staticPath))
	var staticPrefix = basePath + "/static/"
	mux.Handle(staticPrefix, http.StripPrefix(staticPrefix, fileServer))

	var archiveServer = http.FileServer(http.Dir(conf.ArchiveOutputLocation))
	var archiveServerPrefix = basePath + "/archives/"
	mux.Handle(archiveServerPrefix, http.StripPrefix(archiveServerPrefix, archiveServer))

	if basePath == "" {
		basePath = "/"
	}
	initTemplates(basePath)

	// Set up the in-memory session store
	var store = memstore.New(time.Hour * 24)
	sessionManager = scs.NewManager(store)
	sessionManager.Lifetime(time.Hour * 24)
	sessionManager.HttpOnly(false)

	var server = &http.Server{Addr: conf.BindAddress, Handler: sessionManager.Use(mux)}

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
