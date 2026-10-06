package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"devdrop/internal/database"
	"devdrop/internal/handlers"
	"devdrop/internal/hub"
	"devdrop/internal/transfer"
	"devdrop/web"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	portFlag := flag.Int("port", 8080, "Port to listen on")
	dataDirFlag := flag.String("data", "./data", "Directory to store database and uploaded files")
	flag.Parse()

	port := *portFlag
	if envPort := os.Getenv("PORT"); envPort != "" {
		_, _ = fmt.Sscanf(envPort, "%d", &port)
	}

	dataDir := *dataDirFlag
	if envData := os.Getenv("DATA_DIR"); envData != "" {
		dataDir = envData
	}

	uploadDir := filepath.Join(dataDir, "uploads")

	// 1. Initialize SQLite Database
	db, err := database.InitDB(dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// 2. Initialize WebSocket Hub
	wsHub := hub.NewHub(db)
	go wsHub.Run()

	// 3. Initialize Transfer Manager & Start Ephemeral Cleaner Worker
	tm, err := transfer.NewManager(uploadDir, db, wsHub)
	if err != nil {
		log.Fatalf("Failed to initialize transfer manager: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tm.StartCleanupWorker(ctx)

	// 4. Setup Routing
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// CORS for dev convenience
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link", "Content-Disposition", "X-Has-More"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Register API and WS routes
	h := handlers.NewServerHandler(db, wsHub, tm)
	h.RegisterRoutes(r)

	// Embedded Static File Server with SPA Fallback
	distFS, err := web.GetFileSystem()
	if err != nil {
		log.Fatalf("Failed to get embedded filesystem: %v", err)
	}
	fileServer := http.FileServer(distFS)

	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		// If requesting /api/* or /ws, return 404
		if strings.HasPrefix(req.URL.Path, "/api") || strings.HasPrefix(req.URL.Path, "/ws") {
			http.NotFound(w, req)
			return
		}

		// Try opening the requested file in embedded FS
		path := strings.TrimPrefix(req.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		f, err := distFS.Open(path)
		if err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, req)
			return
		}

		// Fallback to index.html for SPA routing
		indexFile, err := distFS.Open("index.html")
		if err != nil {
			http.Error(w, "SPA index.html not found", http.StatusNotFound)
			return
		}
		defer indexFile.Close()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.Copy(w, indexFile)
	})

	addr := fmt.Sprintf(":%d", port)
	server := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Minute, // Support long uploads/downloads
		WriteTimeout: 30 * time.Minute,
	}

	// Print startup information with detected LAN IPs
	printBanner(port)

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down DevDrop server...")
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

func printBanner(port int) {
	fmt.Println()
	fmt.Println("  ┌──────────────────────────────────────────────────────────┐")
	fmt.Println("  │                      DevDrop LAN                         │")
	fmt.Println("  │    Zero-Config Developer Collaboration Service           │")
	fmt.Println("  └──────────────────────────────────────────────────────────┘")
	fmt.Printf("   > Local:    http://localhost:%d\n", port)

	ips := getLocalIPs()
	for _, ip := range ips {
		fmt.Printf("   > Network:  http://%s:%d\n", ip, port)
	}
	fmt.Println()
}

func getLocalIPs() []string {
	var ips []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	return ips
}
