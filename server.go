package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

// Server represents the daemon that serves end user requests
type Server struct {
	httpServer *http.Server
	wg         sync.WaitGroup
}

// Start launches the Server
func Start(r *mux.Router, cfg Config) *Server {
	// Setup Context
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := Server{
		httpServer: &http.Server{
			Addr:           cfg.Host,
			Handler:        r,
			ReadTimeout:    cfg.ReadTimeout,
			MaxHeaderBytes: 1 << 20,
		},
	}

	// Add to the WaitGroup for the listener goroutine
	server.wg.Add(1)

	// Start the listener
	go func() {
		log.Info("Server : Service started : Host=", cfg.Host)
		server.httpServer.ListenAndServe()
		server.wg.Done()
	}()

	return &server
}

// Stop turns off the Server
func (server *Server) Stop() error {
	// Create a context to attempt a graceful 5 second shutdown
	const timeout = 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.Info("\nServer : Service stopping\n")

	// Attempt the graceful shutdown by closing the listener
	// and completing all inflight requests
	if err := server.httpServer.Shutdown(ctx); err != nil {
		// Force close on timeout
		if err := server.httpServer.Close(); err != nil {
			fmt.Printf("\nServer : Service stopping : Error=%v\n", err)
			return err
		}
	}

	// Wait for the listener to report that it is closed
	server.wg.Wait()
	log.Info("\nServer : Stopped\n")
	return nil
}
