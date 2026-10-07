package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"mime"
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
	passwordFlag := flag.String("password", "", "Access password for DevDrop LAN")
	tlsFlag := flag.Bool("tls", false, "Enable auto-generated self-signed HTTPS/TLS for secure PWA installation")
	certFlag := flag.String("cert", "", "Path to custom TLS certificate file (e.g. from mkcert)")
	keyFlag := flag.String("key", "", "Path to custom TLS private key file")
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

	// Check command-line flag or environment variable for password
	password := *passwordFlag
	if envPass := os.Getenv("DEVDROP_PASSWORD"); envPass != "" {
		password = envPass
	} else if envPass := os.Getenv("PASSWORD"); envPass != "" {
		password = envPass
	}

	if err := db.SyncPassword(password); err != nil {
		log.Fatalf("Failed to configure access password: %v", err)
	}
	if password != "" {
		log.Println("Access password configured from CLI/environment")
	} else {
		log.Println("DevDrop running without access password (set via -password or DEVDROP_PASSWORD)")
	}

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

	// Register PWA MIME types
	_ = mime.AddExtensionType(".webmanifest", "application/manifest+json")
	_ = mime.AddExtensionType(".js", "application/javascript")
	_ = mime.AddExtensionType(".svg", "image/svg+xml")

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
			if path == "sw.js" {
				w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
				w.Header().Set("Service-Worker-Allowed", "/")
				w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			} else if path == "manifest.webmanifest" || path == "manifest.json" {
				w.Header().Set("Content-Type", "application/manifest+json; charset=utf-8")
				w.Header().Set("Cache-Control", "public, max-age=3600")
			}
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

	// Determine TLS configuration
	isTLS := false
	tlsCertFile := *certFlag
	tlsKeyFile := *keyFlag

	if tlsCertFile != "" && tlsKeyFile != "" {
		isTLS = true
	} else if *tlsFlag {
		ips := getLocalIPs()
		certPath, keyPath, err := getOrCreateSelfSignedCert(dataDir, ips)
		if err != nil {
			log.Fatalf("Failed to generate self-signed TLS certificates: %v", err)
		}
		tlsCertFile = certPath
		tlsKeyFile = keyPath
		isTLS = true
	}

	// Print startup information with detected LAN IPs
	printBanner(port, db, isTLS)

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

	if isTLS {
		if err := server.ListenAndServeTLS(tlsCertFile, tlsKeyFile); err != nil && err != http.ErrServerClosed {
			log.Fatalf("TLS Server error: %v", err)
		}
	} else {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}
}

func printBanner(port int, db *database.DB, isTLS bool) {
	scheme := "http"
	if isTLS {
		scheme = "https"
	}

	fmt.Println()
	fmt.Println("  ┌──────────────────────────────────────────────────────────┐")
	fmt.Println("  │                      DevDrop LAN                         │")
	fmt.Println("  │    Zero-Config Developer Collaboration Service           │")
	fmt.Println("  └──────────────────────────────────────────────────────────┘")
	fmt.Printf("   > Local:    %s://localhost:%d\n", scheme, port)

	ips := getLocalIPs()
	for _, ip := range ips {
		fmt.Printf("   > Network:  %s://%s:%d\n", scheme, ip, port)
	}

	if hasPass, _ := db.HasPassword(); hasPass {
		fmt.Println("   > Security: Password Protected (Configured via CLI / ENV)")
	} else {
		fmt.Println("   > Security: Open Access (Set password via -password or DEVDROP_PASSWORD)")
	}

	if isTLS {
		fmt.Println("   > Protocol: HTTPS/TLS Active (Secure Context for PWA Installation)")
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

func getOrCreateSelfSignedCert(dataDir string, ips []string) (string, string, error) {
	certFile := filepath.Join(dataDir, "cert.pem")
	keyFile := filepath.Join(dataDir, "key.pem")

	if _, err := os.Stat(certFile); err == nil {
		if _, err := os.Stat(keyFile); err == nil {
			return certFile, keyFile, nil
		}
	}

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return "", "", err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"DevDrop LAN"},
			CommonName:   "DevDrop LAN Node",
		},
		NotBefore: time.Now().Add(-1 * time.Hour),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost", "devdrop.local"},
	}

	template.IPAddresses = append(template.IPAddresses, net.ParseIP("127.0.0.1"), net.ParseIP("::1"))
	for _, ipStr := range ips {
		if ip := net.ParseIP(ipStr); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		}
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return "", "", err
	}

	certOut, err := os.Create(certFile)
	if err != nil {
		return "", "", err
	}
	_ = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	_ = certOut.Close()

	keyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return "", "", err
	}
	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", "", err
	}
	_ = pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
	_ = keyOut.Close()

	return certFile, keyFile, nil
}
