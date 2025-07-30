package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	address   = getEnv("ADDRESS", "0.0.0.0")
	port      = getEnv("PORT", "3000")
	redisAddr = getEnv("REDIS_ADDR", "localhost:6379")
	rdb       *redis.Client
)

type Paste struct {
	ID         string    `json:"id"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at,omitempty"`
	Expiration int64     `json:"-"`
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func createPaste(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	content := r.PostForm.Get("content")
	if content == "" {
		http.Error(w, "Content cannot be empty", http.StatusBadRequest)
		return
	}

	// Parse expiration
	expirationStr := r.PostForm.Get("expiration")
	var expiration int64 = 86400 // Default: 1 day (86400 seconds)

	if expirationStr != "" {
		parsed, err := strconv.ParseInt(expirationStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid expiration", http.StatusBadRequest)
			return
		}
		expiration = parsed
	}

	slog.Info("Setting paste expiration", "seconds", expiration)

	// Generate a UUID
	pasteUUID := uuid.New()
	pasteID := pasteUUID.String()

	slog.Info("Generated new paste ID", "id", pasteID)

	// Create the paste object
	paste := Paste{
		ID:         pasteID,
		Content:    content,
		CreatedAt:  time.Now(),
		Expiration: expiration,
	}

	// Set expiration if applicable
	if expiration > 0 {
		paste.ExpiresAt = paste.CreatedAt.Add(time.Duration(expiration) * time.Second)
	}

	// Serialize and store in Redis
	pasteJSON, err := json.Marshal(paste)
	if err != nil {
		slog.Error("Failed to marshal paste", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	ctx := context.Background()
	key := fmt.Sprintf("paste:%s", pasteID)

	if expiration > 0 {
		err = rdb.Set(ctx, key, pasteJSON, time.Duration(expiration)*time.Second).Err()
	} else {
		err = rdb.Set(ctx, key, pasteJSON, 0).Err()
	}

	if err != nil {
		slog.Error("Failed to save paste to Redis", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Verify the paste was stored correctly
	savedData, err := rdb.Get(ctx, key).Result()
	if err != nil {
		slog.Error("Failed to verify paste was saved", "id", pasteID, "error", err)
	} else {
		slog.Info("Paste saved and verified in Redis", "id", pasteID, "keyLength", len(key), "dataLength", len(savedData))
	}

	// Return the paste URL
	w.Header().Set("Content-Type", "text/html")
	tmpl := `
	<div class="alert alert-success" role="alert">
		Paste created successfully!
	</div>
	<div class="card shadow-sm">
		<div class="card-header">Paste URL</div>
		<div class="card-body">
			<p class="card-text">Share this link with others to view your paste:</p>
			<div class="input-group mb-3">
				<input type="text" class="form-control" value="{{.BaseURL}}/view/{{.ID}}" id="paste-url" readonly>
				<button class="btn btn-outline-primary" type="button" onclick="copyToClipboard()">Copy</button>
			</div>
			<div class="d-grid gap-2 d-md-flex justify-content-md-end">
				<a href="/view/{{.ID}}" class="btn btn-primary">View Paste</a>
			</div>
		</div>
	</div>
	<script>
		function copyToClipboard() {
			var copyText = document.getElementById("paste-url");
			copyText.select();
			document.execCommand("copy");
			alert("Copied the URL: " + copyText.value);
		}
	</script>
	`

	t := template.Must(template.New("result").Parse(tmpl))
	baseURL := fmt.Sprintf("http://%s:%s", address, port)
	data := struct {
		ID      string
		BaseURL string
	}{
		ID:      pasteID,
		BaseURL: baseURL,
	}
	t.Execute(w, data)
}

func deletePasteHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("deletePasteHandler called", "path", r.URL.Path, "method", r.Method)

	// Extract paste ID from URL
	pasteID := ""

	// First try to get the ID from path parameters (Go 1.22+ style)
	if pathParams := r.PathValue("id"); pathParams != "" {
		pasteID = pathParams
	} else {
		// Fallback to manual extraction
		parts := strings.Split(r.URL.Path, "/")

		if len(parts) < 3 {
			slog.Error("Invalid paste ID (not enough URL parts)", "parts", parts)
			http.Error(w, "Invalid paste ID", http.StatusBadRequest)
			return
		}

		pasteID = parts[2]
	}

	slog.Info("Deleting paste", "id", pasteID)

	ctx := context.Background()

	// Get paste key
	key := fmt.Sprintf("paste:%s", pasteID)

	// Check if paste exists
	exists, err := rdb.Exists(ctx, key).Result()
	if err != nil {
		slog.Error("Error checking if paste exists", "id", pasteID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if exists == 0 {
		slog.Error("Paste not found for deletion", "id", pasteID)
		http.Error(w, "Paste not found", http.StatusNotFound)
		return
	}

	// Delete the paste
	_, err = rdb.Del(ctx, key).Result()
	if err != nil {
		slog.Error("Error deleting paste", "id", pasteID, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Also remove from recent pastes if it exists there
	rdb.ZRem(ctx, "recent_pastes", pasteID)

	slog.Info("Paste deleted successfully", "id", pasteID)

	// Return success response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Paste deleted successfully"))
}

func getPasteHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("getPasteHandler called", "path", r.URL.Path, "method", r.Method)

	// Check if it's a DELETE request
	if r.Method == http.MethodDelete {
		deletePasteHandler(w, r)
		return
	} else if r.Method != http.MethodGet {
		slog.Error("Method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set headers for better browser handling
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")

	// Extract paste ID from URL - in the new pattern router, we should get it from path parameters
	// However, we need to handle both the pattern style and direct extraction
	pasteID := ""

	// First try to get the ID from path parameters (Go 1.22+ style)
	if pathParams := r.PathValue("id"); pathParams != "" {
		pasteID = pathParams
		slog.Info("Found paste ID in path parameter", "id", pasteID)
	} else {
		// Fallback to manual extraction
		parts := strings.Split(r.URL.Path, "/")
		slog.Info("URL parts", "parts", parts)

		if len(parts) < 3 {
			slog.Error("Invalid paste ID (not enough URL parts)", "parts", parts)
			http.Error(w, "Invalid paste ID", http.StatusBadRequest)
			return
		}

		pasteID = parts[2]
		slog.Info("Extracted paste ID from path manually", "id", pasteID)
	}
	slog.Info("Looking for paste", "id", pasteID)

	ctx := context.Background()

	// Get paste from Redis
	key := fmt.Sprintf("paste:%s", pasteID)
	result, err := rdb.Get(ctx, key).Result()

	if err == redis.Nil {
		// Paste not found or expired
		slog.Error("Paste not found in Redis", "id", pasteID, "key", key)
		http.Error(w, "Paste not found or has expired", http.StatusNotFound)
		return
	} else if err != nil {
		slog.Error("Failed to retrieve paste from Redis", "error", err, "id", pasteID)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Parse the paste
	var paste Paste
	if err := json.Unmarshal([]byte(result), &paste); err != nil {
		slog.Error("Failed to unmarshal paste", "error", err, "id", pasteID, "raw", result)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	slog.Info("Paste retrieved successfully", "id", pasteID, "created", paste.CreatedAt)

	// Return paste data
	tmpl := `
	<div class="card shadow-sm">
		<div class="card-header d-flex justify-content-between align-items-center">
			<div>
				<span class="fw-bold">Paste ID:</span> {{.ID}}
			</div>
			<div>
				<small class="text-muted">Created: {{.CreatedAt.Format "Jan 02, 2006 15:04:05"}}</small>
				{{if not .ExpiresAt.IsZero}}
					<small class="text-muted ms-2">Expires: {{.ExpiresAt.Format "Jan 02, 2006 15:04:05"}}</small>
				{{else}}
					<small class="text-muted ms-2">No expiration</small>
				{{end}}
			</div>
		</div>
		<div class="card-body">
			<pre class="mb-0"><code>{{.Content}}</code></pre>
		</div>
		<div class="card-footer d-flex justify-content-between align-items-center">
			<div>
				<a href="/" class="btn btn-sm btn-outline-secondary">Create New Paste</a>
				<button class="btn btn-sm btn-danger ms-2" onclick="deletePaste()">Delete Paste</button>
			</div>
			<button class="btn btn-sm btn-primary" onclick="copyPasteContent()">Copy Content</button>
		</div>
	</div>
	<script>
		function copyPasteContent() {
			const content = document.querySelector('pre code').innerText;
			navigator.clipboard.writeText(content)
				.then(() => alert('Paste content copied to clipboard!'))
				.catch(err => console.error('Failed to copy:', err));
		}
	</script>
	`

	t := template.Must(template.New("view").Parse(tmpl))
	if err := t.Execute(w, paste); err != nil {
		slog.Error("Failed to render paste template", "error", err)
	}
}

func viewPasteHandler(w http.ResponseWriter, r *http.Request) {
	// This handler serves the view.html file for direct access to pastes

	// Log the request to help with debugging
	slog.Info("viewPasteHandler called",
		"path", r.URL.Path,
		"id", r.PathValue("id"))

	// Just serve the HTML file directly and let client-side JS handle the API call
	http.ServeFile(w, r, "static/view.html")
}

// cleanupExpiredPastes scans all pastes and removes those that are past their expiration time
func cleanupExpiredPastes(ctx context.Context) {
	slog.Info("Starting cleanup of expired pastes")

	// Get all paste keys
	iter := rdb.Scan(ctx, 0, "paste:*", 100).Iterator()

	var expiredCount int
	var totalCount int
	now := time.Now()

	// Iterate through all keys
	for iter.Next(ctx) {
		totalCount++
		key := iter.Val()

		// Get the paste data
		result, err := rdb.Get(ctx, key).Result()
		if err != nil {
			if err != redis.Nil {
				slog.Error("Error retrieving paste during cleanup", "key", key, "error", err)
			}
			continue
		}

		// Parse the paste
		var paste Paste
		if err := json.Unmarshal([]byte(result), &paste); err != nil {
			slog.Error("Error unmarshaling paste during cleanup", "key", key, "error", err)
			continue
		}

		// Check if paste has expired
		if !paste.ExpiresAt.IsZero() && paste.ExpiresAt.Before(now) {
			// Paste has expired, delete it
			_, err := rdb.Del(ctx, key).Result()
			if err != nil {
				slog.Error("Error deleting expired paste", "id", paste.ID, "error", err)
			} else {
				expiredCount++
				slog.Info("Deleted expired paste", "id", paste.ID, "expired", paste.ExpiresAt)

				// Also remove from recent pastes if it exists there
				rdb.ZRem(ctx, "recent_pastes", paste.ID)
			}
		}
	}

	if err := iter.Err(); err != nil {
		slog.Error("Error during paste cleanup iteration", "error", err)
	}

	slog.Info("Completed cleanup of expired pastes",
		"total", totalCount,
		"expired", expiredCount)
}

// startCleanupTask starts a goroutine that periodically cleans up expired pastes
func startCleanupTask(ctx context.Context) {
	go func() {
		// Run immediately on startup
		cleanupExpiredPastes(ctx)

		// Then run every hour
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				cleanupExpiredPastes(ctx)
			case <-ctx.Done():
				slog.Info("Cleanup task shutting down")
				return
			}
		}
	}()
}

func main() {
	// Setup Redis client
	rdb = redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// Test Redis connection
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}

	// Start the cleanup task
	startCleanupTask(ctx)

	// Setup HTTP routes with modern Go pattern
	mux := http.NewServeMux()

	// Static file server for root path
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/", fs)

	// API endpoints
	mux.HandleFunc("POST /create", createPaste)
	mux.HandleFunc("GET /api/paste/{id}", getPasteHandler)
	mux.HandleFunc("DELETE /api/paste/{id}", deletePasteHandler)
	mux.HandleFunc("GET /view/{id}", viewPasteHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", address, port),
		Handler: mux,
	}

	slog.Info("Paste server listening", "address", address, "port", port)
	err = server.ListenAndServe()
	if err != nil {
		slog.Error("Error starting server", "error", err.Error())
	}
}
