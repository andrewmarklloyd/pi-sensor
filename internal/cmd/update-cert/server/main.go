package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

const pidFile = "/tmp/simple-http-server.pid"

func main() {
	// ---- single-instance guard ----
	if err := ensureSingleInstance(pidFile); err != nil {
		log.Fatal(err)
	}
	defer os.Remove(pidFile)

	log.Println("starting server")

	// Handle Ctrl+C / SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Serve ./tmp relative to binary
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	tmpDir := filepath.Join(filepath.Dir(exe), "tmp")

	fs := http.FileServer(http.Dir(tmpDir))
	srv := &http.Server{
		Addr:    ":80",
		Handler: fs,
	}

	// Watch for /tmp/shutdown
	go func() {
		t := time.NewTicker(1 * time.Second)
		defer t.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if _, err := os.Stat("/tmp/shutdown"); err == nil {
					log.Println("shutting down server (found /tmp/shutdown)")
					stop()
					return
				}
			}
		}
	}()

	// Start HTTP server
	go func() {
		log.Printf("serving %s on %s\n", tmpDir, srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)

	log.Println("server stopped")
}

func ensureSingleInstance(path string) error {
	// If pid file exists, check if process is alive
	if data, err := os.ReadFile(path); err == nil {
		pid, err := strconv.Atoi(string(data))
		if err == nil && processAlive(pid) {
			return fmt.Errorf("server already running with pid %d", pid)
		}
		// stale pid file
		_ = os.Remove(path)
	}

	// Write our PID
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0644)
}

func processAlive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// signal 0 does not send a signal, just checks existence
	return p.Signal(syscall.Signal(0)) == nil
}
