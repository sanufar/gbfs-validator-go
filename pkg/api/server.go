// Package api provides HTTP handlers for the validator API and viewer.
package api

import (
	"encoding/json"
	"net/http"

	"github.com/gbfs-validator-go/pkg/fetcher"
	"github.com/gbfs-validator-go/pkg/gbfs"
	"github.com/gbfs-validator-go/pkg/validator"
	"github.com/gbfs-validator-go/pkg/version"
)

// Server routes API and optional static assets.
type Server struct {
	mux        *http.ServeMux
	staticFS   http.Handler
}

// NewServer builds a server with API routes only.
func NewServer() *Server {
	s := &Server{
		mux: http.NewServeMux(),
	}
	s.setupRoutes()
	return s
}

// NewServerWithStatic builds a server that also serves static assets.
func NewServerWithStatic(staticDir string) *Server {
	s := &Server{
		mux:      http.NewServeMux(),
		staticFS: http.FileServer(http.Dir(staticDir)),
	}
	s.setupRoutes()
	return s
}

// ServeHTTP adds CORS headers and dispatches to routes.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	s.mux.ServeHTTP(w, r)
}

// setupRoutes registers API and static routes.
func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/api/validator", s.handleValidate)
	s.mux.HandleFunc("/api/feed", s.handleFeed)
	s.mux.HandleFunc("/api/validator-summary", s.handleValidatorSummary)
	
	s.mux.HandleFunc("/api/gbfs", s.handleGBFS)
	s.mux.HandleFunc("/api/proxy", s.handleProxy)

	s.mux.HandleFunc("/health", s.handleHealth)
	
	if s.staticFS != nil {
		s.mux.Handle("/", s.staticFS)
	}
}

// ValidateRequest is the JSON body for validation endpoints.
type ValidateRequest struct {
	URL     string            `json:"url"`
	Options *ValidateOptions  `json:"options,omitempty"`
}

// ValidateOptions configures feed validation.
type ValidateOptions struct {
	Freefloating bool               `json:"freefloating"`
	Docked       bool               `json:"docked"`
	Version      string             `json:"version,omitempty"`
	Auth         *fetcher.AuthConfig `json:"auth,omitempty"`
	
	LenientMode bool `json:"lenientMode"`
	
	CoerceOptions *CoerceOptions `json:"coerceOptions,omitempty"`
}

// CoerceOptions selects coercions when lenient mode is on.
type CoerceOptions struct {
	CoerceBooleans       bool `json:"coerceBooleans"`
	CoerceTimestamps     bool `json:"coerceTimestamps"`
	CoerceNumericStrings bool `json:"coerceNumericStrings"`
	CoerceCoordinates    bool `json:"coerceCoordinates"`
	TreatNullAsAbsent    bool `json:"treatNullAsAbsent"`
}

// handleValidate validates a feed and returns a full result.
func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	var req ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.URL == "" {
		respondError(w, http.StatusBadRequest, "URL is required")
		return
	}

	fetcherOpts := []fetcher.Option{}
	if req.Options != nil && req.Options.Auth != nil {
		fetcherOpts = append(fetcherOpts, fetcher.WithAuth(req.Options.Auth))
	}
	f := fetcher.New(fetcherOpts...)

	validatorOpts := validator.Options{}
	if req.Options != nil {
		validatorOpts.Docked = req.Options.Docked
		validatorOpts.Freefloating = req.Options.Freefloating
		validatorOpts.Version = req.Options.Version
		validatorOpts.LenientMode = req.Options.LenientMode
		
		if req.Options.CoerceOptions != nil {
			validatorOpts.CoerceOptions = &validator.CoerceOptions{
				CoerceBooleans:       req.Options.CoerceOptions.CoerceBooleans,
				CoerceTimestamps:     req.Options.CoerceOptions.CoerceTimestamps,
				CoerceNumericStrings: req.Options.CoerceOptions.CoerceNumericStrings,
				CoerceCoordinates:    req.Options.CoerceOptions.CoerceCoordinates,
				TreatNullAsAbsent:    req.Options.CoerceOptions.TreatNullAsAbsent,
			}
		}
	}
	v := validator.New(f, validatorOpts)

	result, err := v.Validate(r.Context(), req.URL)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// FeedResponse returns feed data for the viewer.
type FeedResponse struct {
	Summary     FeedSummary      `json:"summary"`
	GBFSResult  json.RawMessage  `json:"gbfsResult,omitempty"`
	GBFSVersion string           `json:"gbfsVersion"`
	Files       []FeedFile       `json:"files"`
}

// FeedSummary summarizes viewer fetch state.
type FeedSummary struct {
	VersionUnimplemented bool `json:"versionUnimplemented,omitempty"`
}

// FeedFile describes a fetched file for the viewer.
type FeedFile struct {
	Type     string          `json:"type"`
	File     string          `json:"file"`
	Required bool            `json:"required"`
	Exists   bool            `json:"exists"`
	Body     json.RawMessage `json:"body,omitempty"`
}

// handleFeed returns raw feed payloads for visualization.
func (s *Server) handleFeed(w http.ResponseWriter, r *http.Request) {
	var req ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.URL == "" {
		respondError(w, http.StatusBadRequest, "URL is required")
		return
	}

	fetcherOpts := []fetcher.Option{}
	if req.Options != nil && req.Options.Auth != nil {
		fetcherOpts = append(fetcherOpts, fetcher.WithAuth(req.Options.Auth))
	}
	f := fetcher.New(fetcherOpts...)

	var gbfsFeed gbfs.GBFSFeed
	result := f.FetchJSON(r.Context(), req.URL, &gbfsFeed)
	if result.Error != nil || !result.Exists {
		respondJSON(w, http.StatusOK, FeedResponse{
			Summary: FeedSummary{VersionUnimplemented: true},
		})
		return
	}

	detectedVersion := gbfsFeed.Version
	if detectedVersion == "" {
		detectedVersion = "1.0"
	}

	requirements := version.GetFileRequirements(detectedVersion, version.Options{})

	feedURLs := make(map[string]string)
	for _, feed := range gbfsFeed.Data.Feeds {
		feedURLs[feed.Name] = feed.URL
	}

	files := []FeedFile{}
	for _, req := range requirements {
		file := FeedFile{
			Type:     req.File,
			File:     req.File + ".json",
			Required: req.Required,
		}

		url, exists := feedURLs[req.File]
		if !exists {
			file.Exists = false
			files = append(files, file)
			continue
		}

		fetchResult := f.Fetch(r.Context(), url)
		if fetchResult.Error != nil || !fetchResult.Exists {
			file.Exists = false
			files = append(files, file)
			continue
		}

		file.Exists = true
		file.Body = fetchResult.Body
		files = append(files, file)
	}

	response := FeedResponse{
		Summary:     FeedSummary{},
		GBFSResult:  result.Body,
		GBFSVersion: detectedVersion,
		Files:       files,
	}

	respondJSON(w, http.StatusOK, response)
}

// ValidationSummaryResponse groups validation issues by file.
type ValidationSummaryResponse struct {
	Summary      validator.ValidationSummary `json:"summary"`
	FilesSummary []FileSummary               `json:"filesSummary"`
}

// FileSummary aggregates errors for a single file.
type FileSummary struct {
	Required      bool           `json:"required"`
	Exists        bool           `json:"exists"`
	File          string         `json:"file"`
	HasErrors     bool           `json:"hasErrors"`
	ErrorsCount   int            `json:"errorsCount"`
	GroupedErrors []GroupedError `json:"groupedErrors"`
}

// GroupedError counts identical errors.
type GroupedError struct {
	Keyword    string `json:"keyword"`
	Message    string `json:"message"`
	SchemaPath string `json:"schemaPath"`
	Count      int    `json:"count"`
}

// handleValidatorSummary returns grouped validation errors.
func (s *Server) handleValidatorSummary(w http.ResponseWriter, r *http.Request) {
	var req ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.URL == "" {
		respondError(w, http.StatusBadRequest, "URL is required")
		return
	}

	fetcherOpts := []fetcher.Option{}
	if req.Options != nil && req.Options.Auth != nil {
		fetcherOpts = append(fetcherOpts, fetcher.WithAuth(req.Options.Auth))
	}
	f := fetcher.New(fetcherOpts...)

	validatorOpts := validator.Options{}
	if req.Options != nil {
		validatorOpts.Docked = req.Options.Docked
		validatorOpts.Freefloating = req.Options.Freefloating
		validatorOpts.Version = req.Options.Version
		validatorOpts.LenientMode = req.Options.LenientMode
		
		if req.Options.CoerceOptions != nil {
			validatorOpts.CoerceOptions = &validator.CoerceOptions{
				CoerceBooleans:       req.Options.CoerceOptions.CoerceBooleans,
				CoerceTimestamps:     req.Options.CoerceOptions.CoerceTimestamps,
				CoerceNumericStrings: req.Options.CoerceOptions.CoerceNumericStrings,
				CoerceCoordinates:    req.Options.CoerceOptions.CoerceCoordinates,
				TreatNullAsAbsent:    req.Options.CoerceOptions.TreatNullAsAbsent,
			}
		}
	}
	v := validator.New(f, validatorOpts)

	result, err := v.Validate(r.Context(), req.URL)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := ValidationSummaryResponse{
		Summary:      result.Summary,
		FilesSummary: make([]FileSummary, 0, len(result.Files)),
	}

	for _, file := range result.Files {
		fileSummary := FileSummary{
			Required:    file.Required,
			Exists:      file.Exists,
			File:        file.File,
			HasErrors:   file.HasErrors,
			ErrorsCount: file.ErrorsCount,
		}

		errorGroups := make(map[string]*GroupedError)
		for _, err := range file.Errors {
			key := err.Keyword + "|" + err.Message + "|" + err.SchemaPath
			if group, exists := errorGroups[key]; exists {
				group.Count++
			} else {
				errorGroups[key] = &GroupedError{
					Keyword:    err.Keyword,
					Message:    err.Message,
					SchemaPath: err.SchemaPath,
					Count:      1,
				}
			}
		}

		for _, group := range errorGroups {
			fileSummary.GroupedErrors = append(fileSummary.GroupedErrors, *group)
		}

		response.FilesSummary = append(response.FilesSummary, fileSummary)
	}

	respondJSON(w, http.StatusOK, response)
}

// handleHealth returns a basic liveness response.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

// respondJSON writes a JSON response.
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError writes an error response as JSON.
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
