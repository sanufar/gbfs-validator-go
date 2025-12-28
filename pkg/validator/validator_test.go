package validator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gbfs-validator-go/pkg/fetcher"
)

// mockGBFSServer returns a test server serving a valid feed.
func mockGBFSServer() *httptest.Server {
	var server *httptest.Server
	mux := http.NewServeMux()

	mux.HandleFunc("/gbfs.json", func(w http.ResponseWriter, r *http.Request) {
		baseURL := "http://" + r.Host
		json.NewEncoder(w).Encode(map[string]interface{}{
			"last_updated": time.Now().Format(time.RFC3339),
			"ttl":          0,
			"version":      "3.0",
			"data": map[string]interface{}{
				"feeds": []map[string]string{
					{"name": "system_information", "url": baseURL + "/system_information.json"},
					{"name": "station_information", "url": baseURL + "/station_information.json"},
					{"name": "station_status", "url": baseURL + "/station_status.json"},
					{"name": "vehicle_types", "url": baseURL + "/vehicle_types.json"},
					{"name": "vehicle_status", "url": baseURL + "/vehicle_status.json"},
				},
			},
		})
	})

	mux.HandleFunc("/system_information.json", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"last_updated": time.Now().Format(time.RFC3339),
			"ttl":          0,
			"version":      "3.0",
			"data": map[string]interface{}{
				"system_id":          "test_system",
				"languages":          []string{"en"},
				"name":               []map[string]string{{"text": "Test System", "language": "en"}},
				"timezone":           "America/New_York",
				"opening_hours":      "Mo-Su 00:00-23:59",
				"feed_contact_email": "test@example.com",
			},
		})
	})

	mux.HandleFunc("/station_information.json", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"last_updated": time.Now().Format(time.RFC3339),
			"ttl":          0,
			"version":      "3.0",
			"data": map[string]interface{}{
				"stations": []map[string]interface{}{
					{
						"station_id": "station1",
						"name":       []map[string]string{{"text": "Station 1", "language": "en"}},
						"lat":        40.7128,
						"lon":        -74.0060,
					},
					{
						"station_id": "station2",
						"name":       []map[string]string{{"text": "Station 2", "language": "en"}},
						"lat":        40.7580,
						"lon":        -73.9855,
					},
				},
			},
		})
	})

	mux.HandleFunc("/station_status.json", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"last_updated": time.Now().Format(time.RFC3339),
			"ttl":          0,
			"version":      "3.0",
			"data": map[string]interface{}{
				"stations": []map[string]interface{}{
					{
						"station_id":             "station1",
						"num_vehicles_available": 5,
						"num_docks_available":    10,
						"is_installed":           true,
						"is_renting":             true,
						"is_returning":           true,
						"last_reported":          time.Now().Format(time.RFC3339),
					},
					{
						"station_id":             "station2",
						"num_vehicles_available": 3,
						"num_docks_available":    7,
						"is_installed":           true,
						"is_renting":             true,
						"is_returning":           true,
						"last_reported":          time.Now().Format(time.RFC3339),
					},
				},
			},
		})
	})

	mux.HandleFunc("/vehicle_types.json", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"last_updated": time.Now().Format(time.RFC3339),
			"ttl":          0,
			"version":      "3.0",
			"data": map[string]interface{}{
				"vehicle_types": []map[string]interface{}{
					{
						"vehicle_type_id": "bike1",
						"form_factor":     "bicycle",
						"propulsion_type": "human",
						"name":            []map[string]string{{"text": "Regular Bike", "language": "en"}},
					},
					{
						"vehicle_type_id":  "ebike1",
						"form_factor":      "bicycle",
						"propulsion_type":  "electric_assist",
						"name":             []map[string]string{{"text": "E-Bike", "language": "en"}},
						"max_range_meters": 50000,
					},
				},
			},
		})
	})

	mux.HandleFunc("/vehicle_status.json", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"last_updated": time.Now().Format(time.RFC3339),
			"ttl":          0,
			"version":      "3.0",
			"data": map[string]interface{}{
				"vehicles": []map[string]interface{}{
					{
						"vehicle_id":      "v1",
						"lat":             40.7300,
						"lon":             -73.9950,
						"is_reserved":     false,
						"is_disabled":     false,
						"vehicle_type_id": "bike1",
						"last_reported":   time.Now().Format(time.RFC3339),
					},
					{
						"vehicle_id":           "v2",
						"lat":                  40.7400,
						"lon":                  -73.9850,
						"is_reserved":          false,
						"is_disabled":          false,
						"vehicle_type_id":      "ebike1",
						"current_range_meters": 45000,
						"last_reported":        time.Now().Format(time.RFC3339),
					},
				},
			},
		})
	})

	server = httptest.NewServer(mux)
	return server
}

// TestValidateValidFeed ensures a complete feed validates cleanly.
func TestValidateValidFeed(t *testing.T) {
	server := mockGBFSServer()
	defer server.Close()

	f := fetcher.New()
	v := New(f, Options{})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := v.Validate(ctx, server.URL+"/gbfs.json")
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if result.Summary.Version.Detected != "3.0" {
		t.Errorf("Expected version 3.0, got %s", result.Summary.Version.Detected)
	}

	if result.Summary.HasErrors {
		t.Errorf("Expected no errors, but got %d errors", result.Summary.ErrorsCount)
		for _, file := range result.Files {
			if file.HasErrors {
				t.Logf("File %s has errors:", file.File)
				for _, err := range file.Errors {
					t.Logf("  - %s: %s", err.Severity, err.Message)
				}
			}
		}
	}

	expectedFiles := []string{"gbfs.json", "system_information.json", "station_information.json"}
	for _, expected := range expectedFiles {
		found := false
		for _, file := range result.Files {
			if file.File == expected {
				found = true
				if !file.Exists {
					t.Errorf("Expected file %s to exist", expected)
				}
				break
			}
		}
		if !found {
			t.Errorf("Expected file %s not found in results", expected)
		}
	}
}

// TestValidateMissingRequiredFile checks required file errors.
func TestValidateMissingRequiredFile(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/gbfs.json", func(w http.ResponseWriter, r *http.Request) {
		baseURL := "http://" + r.Host
		json.NewEncoder(w).Encode(map[string]interface{}{
			"last_updated": time.Now().Format(time.RFC3339),
			"ttl":          0,
			"version":      "3.0",
			"data": map[string]interface{}{
				"feeds": []map[string]string{
					{"name": "system_information", "url": baseURL + "/system_information.json"},
				},
			},
		})
	})
	mux.HandleFunc("/system_information.json", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	f := fetcher.New()
	v := New(f, Options{})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := v.Validate(ctx, server.URL+"/gbfs.json")
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if !result.Summary.HasErrors {
		t.Error("Expected validation errors for missing required file")
	}

	for _, file := range result.Files {
		if file.File == "system_information.json" {
			if file.Exists {
				t.Error("Expected system_information.json to not exist")
			}
			if !file.HasErrors {
				t.Error("Expected errors for missing required file")
			}
			break
		}
	}
}

// TestCrossFileValidation checks cross-file reference errors.
func TestCrossFileValidation(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/gbfs.json", func(w http.ResponseWriter, r *http.Request) {
		baseURL := "http://" + r.Host
		json.NewEncoder(w).Encode(map[string]interface{}{
			"last_updated": time.Now().Format(time.RFC3339),
			"ttl":          0,
			"version":      "3.0",
			"data": map[string]interface{}{
				"feeds": []map[string]string{
					{"name": "system_information", "url": baseURL + "/system_information.json"},
					{"name": "vehicle_types", "url": baseURL + "/vehicle_types.json"},
					{"name": "vehicle_status", "url": baseURL + "/vehicle_status.json"},
				},
			},
		})
	})

	mux.HandleFunc("/system_information.json", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"last_updated": time.Now().Format(time.RFC3339),
			"ttl":          0,
			"version":      "3.0",
			"data": map[string]interface{}{
				"system_id":          "test",
				"languages":          []string{"en"},
				"name":               []map[string]string{{"text": "Test", "language": "en"}},
				"timezone":           "UTC",
				"opening_hours":      "Mo-Su 00:00-23:59",
				"feed_contact_email": "test@test.com",
			},
		})
	})

	mux.HandleFunc("/vehicle_types.json", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"last_updated": time.Now().Format(time.RFC3339),
			"ttl":          0,
			"version":      "3.0",
			"data": map[string]interface{}{
				"vehicle_types": []map[string]interface{}{
					{
						"vehicle_type_id": "bike1",
						"form_factor":     "bicycle",
						"propulsion_type": "human",
						"name":            []map[string]string{{"text": "Bike", "language": "en"}},
					},
				},
			},
		})
	})

	mux.HandleFunc("/vehicle_status.json", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"last_updated": time.Now().Format(time.RFC3339),
			"ttl":          0,
			"version":      "3.0",
			"data": map[string]interface{}{
				"vehicles": []map[string]interface{}{
					{
						"vehicle_id":      "v1",
						"lat":             40.0,
						"lon":             -74.0,
						"is_reserved":     false,
						"is_disabled":     false,
						"vehicle_type_id": "nonexistent_type",
						"last_reported":   time.Now().Format(time.RFC3339),
					},
				},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	f := fetcher.New()
	v := New(f, Options{Freefloating: true})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := v.Validate(ctx, server.URL+"/gbfs.json")
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if !result.Summary.HasErrors {
		t.Error("Expected cross-validation error for invalid vehicle_type_id reference")
	}

	foundError := false
	for _, file := range result.Files {
		if file.File == "vehicle_status.json" {
			for _, err := range file.Errors {
				if strings.Contains(err.Message, "vehicle_type_id") && strings.Contains(err.Message, "not found") {
					foundError = true
					break
				}
			}
		}
	}

	if !foundError {
		t.Error("Expected error about invalid vehicle_type_id reference")
	}
}
