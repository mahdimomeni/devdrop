package preview

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

// LinkPreview contains metadata extracted from a URL
type LinkPreview struct {
	URL         string `json:"url"`
	SiteName    string `json:"site_name,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Image       string `json:"image,omitempty"`
	Favicon     string `json:"favicon,omitempty"`
}

type cacheEntry struct {
	preview   *LinkPreview
	expiresAt time.Time
}

// Service manages fetching and caching link preview metadata
type Service struct {
	client *http.Client
	mu     sync.RWMutex
	cache  map[string]cacheEntry
}

// NewService creates a new preview Service
func NewService() *Service {
	client := &http.Client{
		Timeout: 7 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			return nil
		},
	}

	return &Service{
		client: client,
		cache:  make(map[string]cacheEntry),
	}
}

// Fetch returns the LinkPreview for the given URL
func (s *Service) Fetch(ctx context.Context, rawURL string) (*LinkPreview, error) {
	cleanURL := strings.TrimSpace(rawURL)
	if cleanURL == "" {
		return nil, errors.New("empty url")
	}

	parsedURL, err := url.Parse(cleanURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, errors.New("unsupported url scheme: must be http or https")
	}

	if parsedURL.Host == "" {
		return nil, errors.New("url host is empty")
	}

	// 1. Check in-memory cache
	s.mu.RLock()
	entry, found := s.cache[cleanURL]
	s.mu.RUnlock()
	if found && time.Now().Before(entry.expiresAt) {
		return entry.preview, nil
	}

	// 2. Fetch over HTTP
	preview, err := s.doFetch(ctx, parsedURL)
	if err != nil {
		return nil, err
	}

	// 3. Store in cache (30 minutes TTL)
	s.mu.Lock()
	s.cache[cleanURL] = cacheEntry{
		preview:   preview,
		expiresAt: time.Now().Add(30 * time.Minute),
	}
	// Periodically prune stale entries if cache gets large
	if len(s.cache) > 500 {
		now := time.Now()
		for k, v := range s.cache {
			if now.After(v.expiresAt) {
				delete(s.cache, k)
			}
		}
	}
	s.mu.Unlock()

	return preview, nil
}

func (s *Service) doFetch(ctx context.Context, targetURL *url.URL) (*LinkPreview, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL.String(), nil)
	if err != nil {
		return nil, err
	}

	// Use realistic browser user agent so CDNs and sites return rich OpenGraph tags
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36 DevDrop/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch url: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("remote server returned status: %d", resp.StatusCode)
	}

	finalURL := resp.Request.URL
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))

	// If the URL directly points to an image
	if strings.HasPrefix(contentType, "image/") {
		fileName := path.Base(finalURL.Path)
		if fileName == "" || fileName == "/" || fileName == "." {
			fileName = "Image"
		}
		return &LinkPreview{
			URL:      targetURL.String(),
			SiteName: finalURL.Hostname(),
			Title:    fileName,
			Image:    finalURL.String(),
		}, nil
	}

	// Read up to 1MB of HTML body
	limitReader := io.LimitReader(resp.Body, 1024*1024)
	doc, err := html.Parse(limitReader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse html: %w", err)
	}

	var (
		rawTitle     string
		ogTitle      string
		twitterTitle string

		metaDesc    string
		ogDesc      string
		twitterDesc string

		ogImage      string
		twitterImage string

		ogSiteName string
		appName    string

		favicon string
	)

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			tag := strings.ToLower(n.Data)
			switch tag {
			case "title":
				if rawTitle == "" && n.FirstChild != nil {
					rawTitle = strings.TrimSpace(n.FirstChild.Data)
				}
			case "meta":
				var prop, name, content string
				for _, attr := range n.Attr {
					k := strings.ToLower(attr.Key)
					switch k {
					case "property":
						prop = strings.ToLower(attr.Val)
					case "name":
						name = strings.ToLower(attr.Val)
					case "content":
						content = strings.TrimSpace(attr.Val)
					}
				}
				if content != "" {
					switch prop {
					case "og:title":
						if ogTitle == "" {
							ogTitle = content
						}
					case "og:description":
						if ogDesc == "" {
							ogDesc = content
						}
					case "og:image":
						if ogImage == "" {
							ogImage = content
						}
					case "og:site_name":
						if ogSiteName == "" {
							ogSiteName = content
						}
					}
					switch name {
					case "twitter:title":
						if twitterTitle == "" {
							twitterTitle = content
						}
					case "twitter:description":
						if twitterDesc == "" {
							twitterDesc = content
						}
					case "twitter:image", "twitter:image:src":
						if twitterImage == "" {
							twitterImage = content
						}
					case "description":
						if metaDesc == "" {
							metaDesc = content
						}
					case "application-name":
						if appName == "" {
							appName = content
						}
					}
				}
			case "link":
				var rel, href string
				for _, attr := range n.Attr {
					k := strings.ToLower(attr.Key)
					switch k {
					case "rel":
						rel = strings.ToLower(attr.Val)
					case "href":
						href = strings.TrimSpace(attr.Val)
					}
				}
				if href != "" && (rel == "icon" || rel == "shortcut icon" || rel == "apple-touch-icon") {
					if favicon == "" {
						favicon = href
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	// Consolidate fields by priority
	title := ogTitle
	if title == "" {
		title = twitterTitle
	}
	if title == "" {
		title = rawTitle
	}
	if title == "" {
		title = finalURL.Hostname()
	}
	title = cleanText(title, 250)

	desc := ogDesc
	if desc == "" {
		desc = twitterDesc
	}
	if desc == "" {
		desc = metaDesc
	}
	desc = cleanText(desc, 500)

	siteName := ogSiteName
	if siteName == "" {
		siteName = appName
	}
	if siteName == "" {
		siteName = finalURL.Hostname()
	}
	siteName = cleanText(siteName, 80)

	image := ogImage
	if image == "" {
		image = twitterImage
	}
	if image != "" {
		image = resolveURL(finalURL, image)
	}

	if favicon != "" {
		favicon = resolveURL(finalURL, favicon)
	} else {
		// Default fallback to /favicon.ico
		favicon = fmt.Sprintf("%s://%s/favicon.ico", finalURL.Scheme, finalURL.Host)
	}

	return &LinkPreview{
		URL:         targetURL.String(),
		SiteName:    siteName,
		Title:       title,
		Description: desc,
		Image:       image,
		Favicon:     favicon,
	}, nil
}

func resolveURL(base *url.URL, target string) string {
	if target == "" {
		return ""
	}
	u, err := url.Parse(target)
	if err != nil {
		return target
	}
	return base.ResolveReference(u).String()
}

func cleanText(text string, maxLen int) string {
	trimmed := strings.Join(strings.Fields(text), " ")
	if len(trimmed) > maxLen {
		return trimmed[:maxLen] + "..."
	}
	return trimmed
}
