package api

import (
	"io/fs"
	"net/http"
)

// SetupRoutes configures all API routes
func (h *Handler) SetupRoutes(mux *http.ServeMux, staticFS fs.FS) {
	// API routes
	mux.HandleFunc("GET /api/health", h.Health)
	mux.HandleFunc("GET /api/status", h.Status)
	mux.HandleFunc("POST /api/register", h.Register)
	mux.HandleFunc("GET /api/dashboard", h.Dashboard)
	mux.HandleFunc("GET /api/events", h.Events)

	// Admin API routes (protected by basic auth)
	mux.HandleFunc("GET /api/admin/endpoints", h.requireAdminAuth(h.AdminListEndpoints))
	mux.HandleFunc("POST /api/admin/endpoints", h.requireAdminAuth(h.AdminAddEndpoint))
	mux.HandleFunc("DELETE /api/admin/endpoints/{id}", h.requireAdminAuth(h.AdminDeleteEndpoint))
	mux.HandleFunc("GET /api/admin/metrics", h.requireAdminAuth(h.AdminMetrics))

	// Static files (if provided)
	if staticFS != nil {
		mux.Handle("/", spaHandler(staticFS))
	}
}

// requireAdminAuth wraps a handler with basic auth and rate limiting
func (h *Handler) requireAdminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIP := GetClientIP(r)

		// Apply auth-specific rate limiting (prevent brute force)
		if h.authRateLimiter != nil && !h.authRateLimiter.Allow(clientIP) {
			w.Header().Set("Retry-After", "10")
			writeError(w, http.StatusTooManyRequests, "Too many authentication attempts")
			return
		}

		// Check if password is configured
		hasPassword, err := h.db.HasAdminPassword()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Database error")
			return
		}
		if !hasPassword {
			writeError(w, http.StatusForbidden, "Admin access is disabled (no password configured)")
			return
		}

		// Check basic auth
		_, password, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="CCC Admin"`)
			writeError(w, http.StatusUnauthorized, "Authentication required")
			return
		}

		valid, err := h.db.CheckAdminPassword(password)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Database error")
			return
		}
		if !valid {
			w.Header().Set("WWW-Authenticate", `Basic realm="CCC Admin"`)
			writeError(w, http.StatusUnauthorized, "Invalid password")
			return
		}

		next(w, r)
	}
}

// spaHandler serves static files with SPA fallback to index.html
func spaHandler(staticFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(staticFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "index.html"
		} else if path[0] == '/' {
			path = path[1:]
		}

		// Try to open the file
		f, err := staticFS.Open(path)
		if err != nil {
			// File not found, serve index.html for SPA routing
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}
		f.Close()

		// File exists, serve it
		fileServer.ServeHTTP(w, r)
	})
}
