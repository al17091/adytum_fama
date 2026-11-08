package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/al17091/adytum_fama/pkg/storage"
	"github.com/al17091/adytum_fama/web"
)

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

type Server struct {
	db        *storage.DB
	staticFS  fs.FS
	staticDir string
}

func main() {
	port := getEnv("PORT", "8080")
	pgDSN := getEnv("POSTGRES_DSN", "postgres://adytum:adytum@localhost:5432/adytum_fama?sslmode=disable")
	staticDir := getEnv("STATIC_DIR", "")

	log.Printf("[INFO] Initializing AdytumFama Web & API Server on port %s...", port)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to database
	var db *storage.DB
	var err error
	if pgDSN != "" {
		db, err = storage.NewDB(ctx, pgDSN)
		if err != nil {
			log.Printf("[WARN] PostgreSQL connection failed (%v). Running in read-only degraded mode.", err)
		} else {
			defer db.Close()
		}
	}

	// Prepare embedded static filesystem
	subFS, err := fs.Sub(web.StaticAssets, "out")
	if err != nil {
		log.Fatalf("[FATAL] Failed resolving embedded static assets: %v", err)
	}

	srv := &Server{
		db:        db,
		staticFS:  subFS,
		staticDir: staticDir,
	}

	mux := http.NewServeMux()

	// Health check endpoints
	mux.HandleFunc("/healthz", srv.handleHealthz)
	mux.HandleFunc("/guardinfo/api/healthz", srv.handleHealthz)

	// Pre-aggregated snapshot API (1-3ms response)
	mux.HandleFunc("/guardinfo/api/snapshot", srv.handleSnapshot)

	// Redirect root to /guardinfo/
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/guardinfo/", http.StatusMovedPermanently)
			return
		}
		srv.handleStatic(w, r)
	})

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown handling
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("[INFO] Server listening on http://0.0.0.0:%s/guardinfo", port)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[FATAL] HTTP server error: %v", err)
		}
	}()

	<-stop
	log.Println("[INFO] Shutting down server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("[ERROR] Server shutdown failed: %v", err)
	}
	log.Println("[INFO] Server stopped.")
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	window := r.URL.Query().Get("window")
	if window != "7d" {
		window = "24h"
	}

	if s.db == nil {
		http.Error(w, `{"error":"database unavailable"}`, http.StatusServiceUnavailable)
		return
	}

	payload, err := s.db.GetSnapshot(r.Context(), window)
	if err != nil {
		log.Printf("[WARN] Snapshot query failed for window %s: %v", window, err)
		http.Error(w, `{"error":"snapshot not ready"}`, http.StatusNotFound)
		return
	}

	// CloudFront edge cache policy: 5 min browser, 30 min CloudFront edge, 60 min stale
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300, s-maxage=1800, stale-while-revalidate=3600")

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("[ERROR] Failed encoding snapshot response: %v", err)
	}
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	// Trim prefix "/guardinfo"
	relPath := strings.TrimPrefix(r.URL.Path, "/guardinfo")
	relPath = strings.TrimPrefix(relPath, "/")
	if relPath == "" {
		relPath = "index.html"
	}

	// If asset is inside _next/static, apply immutable long-term caching
	if strings.HasPrefix(relPath, "_next/static/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else if strings.HasSuffix(relPath, ".html") {
		w.Header().Set("Cache-Control", "public, max-age=300, must-revalidate")
	}

	// Serve from custom local directory if provided
	if s.staticDir != "" {
		filePath := path.Join(s.staticDir, relPath)
		if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
			// Fallback to index.html for SPA
			http.ServeFile(w, r, path.Join(s.staticDir, "index.html"))
			return
		}
		http.ServeFile(w, r, filePath)
		return
	}

	// Check file existence in embedded FS
	f, err := s.staticFS.Open(relPath)
	if err != nil {
		// Fallback to index.html for client-side routing
		indexFile, indexErr := s.staticFS.Open("index.html")
		if indexErr != nil {
			http.NotFound(w, r)
			return
		}
		defer indexFile.Close()
		data, _ := io.ReadAll(indexFile)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
		return
	}
	defer f.Close()

	if rs, ok := f.(io.ReadSeeker); ok {
		http.ServeContent(w, r, path.Base(relPath), time.Time{}, rs)
		return
	}

	data, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(data)
}
