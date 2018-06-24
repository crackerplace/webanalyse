//go:generate statik -src=./public
package main

import (
	"net/http"
	"os"
	"os/signal"
	"time"

	log "github.com/Sirupsen/logrus"
	_ "github.com/crackerplace/webanalyse/statik"
	"github.com/gorilla/mux"
	"github.com/rakyll/statik/fs"
)

// Config provides basic server configuration
type Config struct {
	Host         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func newRouter() *mux.Router {
	r := mux.NewRouter()
	statikFS, err := fs.New()
	if err != nil {
		log.Fatal(err)
	}
	fs := http.FileServer(statikFS)
	r.Handle("/", fs).Methods("GET")
	r.HandleFunc("/analyse", analyseHandler()).Methods("POST")
	return r
}

func main() {
	serverCfg := Config{
		Host:        "localhost:8080",
		ReadTimeout: 20 * time.Second,
	}
	server := Start(newRouter(), serverCfg)
	defer server.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan

	log.Info("shutting down")
}
