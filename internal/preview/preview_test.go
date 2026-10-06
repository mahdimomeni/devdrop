package preview

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPreviewOpenGraph(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `
		<!DOCTYPE html>
		<html>
		<head>
			<meta property="og:site_name" content="GitHub" />
			<meta property="og:title" content="DevDrop: Instant LAN File Sharing" />
			<meta property="og:description" content="Peer-to-peer file sharing and chat for local networks." />
			<meta property="og:image" content="/assets/og-banner.png" />
			<link rel="icon" href="/favicon.png" />
			<title>Fallback Title</title>
		</head>
		<body>
			<h1>Hello World</h1>
		</body>
		</html>
		`
		fmt.Fprint(w, html)
	}))
	defer server.Close()

	svc := NewService()
	p, err := svc.Fetch(context.Background(), server.URL+"/repo/devdrop")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.SiteName != "GitHub" {
		t.Errorf("expected SiteName GitHub, got %q", p.SiteName)
	}
	if p.Title != "DevDrop: Instant LAN File Sharing" {
		t.Errorf("expected Title 'DevDrop: Instant LAN File Sharing', got %q", p.Title)
	}
	if p.Description != "Peer-to-peer file sharing and chat for local networks." {
		t.Errorf("expected Description 'Peer-to-peer file sharing and chat for local networks.', got %q", p.Description)
	}
	expectedImg := server.URL + "/assets/og-banner.png"
	if p.Image != expectedImg {
		t.Errorf("expected Image %q, got %q", expectedImg, p.Image)
	}
	expectedFavicon := server.URL + "/favicon.png"
	if p.Favicon != expectedFavicon {
		t.Errorf("expected Favicon %q, got %q", expectedFavicon, p.Favicon)
	}
}

func TestPreviewTwitterFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `
		<!DOCTYPE html>
		<html>
		<head>
			<meta name="twitter:title" content="Twitter Card Title" />
			<meta name="twitter:description" content="Twitter description text." />
			<meta name="twitter:image" content="https://example.com/tw.jpg" />
			<title>Fallback Title</title>
		</head>
		<body></body>
		</html>
		`
		fmt.Fprint(w, html)
	}))
	defer server.Close()

	svc := NewService()
	p, err := svc.Fetch(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Title != "Twitter Card Title" {
		t.Errorf("expected Title 'Twitter Card Title', got %q", p.Title)
	}
	if p.Description != "Twitter description text." {
		t.Errorf("expected Description 'Twitter description text.', got %q", p.Description)
	}
	if p.Image != "https://example.com/tw.jpg" {
		t.Errorf("expected Image 'https://example.com/tw.jpg', got %q", p.Image)
	}
}

func TestPreviewStandardHTMLFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Standard Page Title</title>
			<meta name="description" content="Standard page description." />
		</head>
		<body></body>
		</html>
		`
		fmt.Fprint(w, html)
	}))
	defer server.Close()

	svc := NewService()
	p, err := svc.Fetch(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Title != "Standard Page Title" {
		t.Errorf("expected Title 'Standard Page Title', got %q", p.Title)
	}
	if p.Description != "Standard page description." {
		t.Errorf("expected Description 'Standard page description.', got %q", p.Description)
	}
}

func TestPreviewDirectImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		fmt.Fprint(w, "fakepngdata")
	}))
	defer server.Close()

	svc := NewService()
	p, err := svc.Fetch(context.Background(), server.URL+"/logo.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Image != server.URL+"/logo.png" {
		t.Errorf("expected Image %q, got %q", server.URL+"/logo.png", p.Image)
	}
	if p.Title != "logo.png" {
		t.Errorf("expected Title 'logo.png', got %q", p.Title)
	}
}

func TestPreviewCache(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<html><head><title>Count %d</title></head></html>", callCount)
	}))
	defer server.Close()

	svc := NewService()
	p1, err := svc.Fetch(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p1.Title != "Count 1" {
		t.Errorf("expected Title 'Count 1', got %q", p1.Title)
	}

	p2, err := svc.Fetch(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p2.Title != "Count 1" {
		t.Errorf("expected Title 'Count 1' from cache, got %q", p2.Title)
	}
	if callCount != 1 {
		t.Errorf("expected exactly 1 call to server, got %d", callCount)
	}
}

func TestPreviewInvalidScheme(t *testing.T) {
	svc := NewService()
	_, err := svc.Fetch(context.Background(), "javascript:alert(1)")
	if err == nil {
		t.Errorf("expected error for javascript scheme, got nil")
	}

	_, err = svc.Fetch(context.Background(), "file:///etc/passwd")
	if err == nil {
		t.Errorf("expected error for file scheme, got nil")
	}
}
