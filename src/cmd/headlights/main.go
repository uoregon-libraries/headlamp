package main

import (
	"context"
	"db"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/uoregon-libraries/gopkg/interrupts"
	"github.com/uoregon-libraries/gopkg/logger"
)

// dbh is our global database handle for DA searches
var dbh = db.New()

func main() {
	var baseURL, bind = getCLI()

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

	var basePath = u.Path
	mux.HandleFunc(basePath+"/", homeHandler)

	var staticPath = filepath.Join(filepath.Dir(os.Args[2]), "static")
	var fileServer = http.FileServer(http.Dir(staticPath))
	var staticPrefix = basePath + "/static/"
	mux.Handle(staticPrefix, http.StripPrefix(staticPrefix, fileServer))

	initTemplates(u)
	var server = &http.Server{Addr: bind, Handler: mux}

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
