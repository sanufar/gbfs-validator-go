// Package api provides HTTP handlers for the validator API and viewer.
package api

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gbfs-validator-go/pkg/fetcher"
)

// ViewerRequest is the JSON body for /api/gbfs.
type ViewerRequest struct {
	URL string `json:"url"`
}

// ViewerResponse returns data for the viewer UI.
type ViewerResponse struct {
	Version        string                 `json:"version"`
	SystemInfo     map[string]interface{} `json:"systemInfo,omitempty"`
	Stations       []Station              `json:"stations"`
	Vehicles       []Vehicle              `json:"vehicles"`
	VehicleTypes   []interface{}          `json:"vehicleTypes"`
	GeofencingZones interface{}           `json:"geofencingZones"`
	FeedURLs       map[string]string      `json:"feedUrls"`
}

// Station merges station_information and station_status.
type Station struct {
	StationID           string      `json:"station_id"`
	Name                string      `json:"name,omitempty"`
	Lat                 float64     `json:"lat"`
	Lon                 float64     `json:"lon"`
	Capacity            int         `json:"capacity,omitempty"`
	IsInstalled         interface{} `json:"is_installed,omitempty"`
	IsRenting           interface{} `json:"is_renting,omitempty"`
	IsReturning         interface{} `json:"is_returning,omitempty"`
	NumBikesAvailable   int         `json:"num_bikes_available,omitempty"`
	NumDocksAvailable   int         `json:"num_docks_available,omitempty"`
	NumVehiclesAvailable int        `json:"num_vehicles_available,omitempty"`
}

// Vehicle represents a free-floating vehicle.
type Vehicle struct {
	VehicleID          string  `json:"vehicle_id,omitempty"`
	BikeID             string  `json:"bike_id,omitempty"`
	Lat                float64 `json:"lat"`
	Lon                float64 `json:"lon"`
	IsReserved         interface{} `json:"is_reserved,omitempty"`
	IsDisabled         interface{} `json:"is_disabled,omitempty"`
	VehicleTypeID      string  `json:"vehicle_type_id,omitempty"`
	CurrentRangeMeters float64 `json:"current_range_meters,omitempty"`
}

// handleGBFS fetches feeds and builds a viewer payload.
func (s *Server) handleGBFS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	var req ViewerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.URL == "" {
		respondError(w, http.StatusBadRequest, "URL is required")
		return
	}

	ctx := r.Context()
	f := fetcher.New()
	
	var gbfsData map[string]interface{}
	result := f.FetchJSON(ctx, req.URL, &gbfsData)
	if result.Error != nil {
		respondError(w, http.StatusBadGateway, "Failed to fetch GBFS: "+result.Error.Error())
		return
	}

	version := "1.0"
	if v, ok := gbfsData["version"].(string); ok {
		version = v
	}

	feedURLs := extractFeedURLs(gbfsData)
	if len(feedURLs) == 0 {
		respondError(w, http.StatusBadRequest, "No feeds found in autodiscovery")
		return
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	feeds := make(map[string]map[string]interface{})

	feedsToFetch := []string{
		"system_information",
		"station_information",
		"station_status",
		"free_bike_status",
		"vehicle_status",
		"vehicle_types",
		"geofencing_zones",
	}

	for _, feedName := range feedsToFetch {
		url, ok := feedURLs[feedName]
		if !ok {
			continue
		}

		wg.Add(1)
		go func(name, url string) {
			defer wg.Done()
			var data map[string]interface{}
			result := f.FetchJSON(ctx, url, &data)
			if result.Error != nil {
				return
			}
			mu.Lock()
			feeds[name] = data
			mu.Unlock()
		}(feedName, url)
	}

	wg.Wait()

	resp := ViewerResponse{
		Version:   version,
		Stations:  []Station{},
		Vehicles:  []Vehicle{},
		VehicleTypes: []interface{}{},
		FeedURLs:  feedURLs,
	}

	if sysInfo, ok := feeds["system_information"]; ok {
		if data, ok := sysInfo["data"].(map[string]interface{}); ok {
			resp.SystemInfo = data
		}
	}

	resp.Stations = mergeStations(feeds["station_information"], feeds["station_status"])

	if fbs, ok := feeds["free_bike_status"]; ok {
		resp.Vehicles = extractVehicles(fbs)
	} else if vs, ok := feeds["vehicle_status"]; ok {
		resp.Vehicles = extractVehicles(vs)
	}

	if vt, ok := feeds["vehicle_types"]; ok {
		if data, ok := vt["data"].(map[string]interface{}); ok {
			if types, ok := data["vehicle_types"].([]interface{}); ok {
				resp.VehicleTypes = types
			}
		}
	}

	if gz, ok := feeds["geofencing_zones"]; ok {
		if data, ok := gz["data"].(map[string]interface{}); ok {
			resp.GeofencingZones = data["geofencing_zones"]
		}
	}

	respondJSON(w, http.StatusOK, resp)
}

// handleProxy proxies a URL and returns its body.
func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL.Query().Get("url")
	if targetURL == "" {
		respondError(w, http.StatusBadRequest, "url parameter required")
		return
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(targetURL)
	if err != nil {
		respondError(w, http.StatusBadGateway, "Failed to fetch: "+err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		respondError(w, http.StatusBadGateway, "Failed to read response")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

// extractFeedURLs reads feed URLs from autodiscovery data.
func extractFeedURLs(gbfs map[string]interface{}) map[string]string {
	urls := make(map[string]string)

	data, ok := gbfs["data"].(map[string]interface{})
	if !ok {
		return urls
	}

	for _, value := range data {
		if langData, ok := value.(map[string]interface{}); ok {
			if feeds, ok := langData["feeds"].([]interface{}); ok {
				for _, f := range feeds {
					if feed, ok := f.(map[string]interface{}); ok {
						name, _ := feed["name"].(string)
						url, _ := feed["url"].(string)
						if name != "" && url != "" {
							urls[name] = url
						}
					}
				}
				return urls
			}
		}
	}

	if feeds, ok := data["feeds"].([]interface{}); ok {
		for _, f := range feeds {
			if feed, ok := f.(map[string]interface{}); ok {
				name, _ := feed["name"].(string)
				url, _ := feed["url"].(string)
				if name != "" && url != "" {
					urls[name] = url
				}
			}
		}
	}

	return urls
}

// mergeStations combines station info and status.
func mergeStations(info, status map[string]interface{}) []Station {
	stations := []Station{}

	if info == nil {
		return stations
	}

	data, ok := info["data"].(map[string]interface{})
	if !ok {
		return stations
	}

	infoStations, ok := data["stations"].([]interface{})
	if !ok {
		return stations
	}

	statusMap := make(map[string]map[string]interface{})
	if status != nil {
		if statusData, ok := status["data"].(map[string]interface{}); ok {
			if statusStations, ok := statusData["stations"].([]interface{}); ok {
				for _, s := range statusStations {
					if station, ok := s.(map[string]interface{}); ok {
						if id, ok := station["station_id"].(string); ok {
							statusMap[id] = station
						}
					}
				}
			}
		}
	}

	for _, s := range infoStations {
		info, ok := s.(map[string]interface{})
		if !ok {
			continue
		}

		station := Station{}
		station.StationID, _ = info["station_id"].(string)
		station.Name, _ = info["name"].(string)
		station.Lat, _ = info["lat"].(float64)
		station.Lon, _ = info["lon"].(float64)
		if cap, ok := info["capacity"].(float64); ok {
			station.Capacity = int(cap)
		}

		if ss, ok := statusMap[station.StationID]; ok {
			station.IsInstalled = ss["is_installed"]
			station.IsRenting = ss["is_renting"]
			station.IsReturning = ss["is_returning"]
			if n, ok := ss["num_bikes_available"].(float64); ok {
				station.NumBikesAvailable = int(n)
			}
			if n, ok := ss["num_docks_available"].(float64); ok {
				station.NumDocksAvailable = int(n)
			}
			if n, ok := ss["num_vehicles_available"].(float64); ok {
				station.NumVehiclesAvailable = int(n)
			}
		}

		stations = append(stations, station)
	}

	return stations
}

// extractVehicles reads vehicles from free_bike_status or vehicle_status.
func extractVehicles(feed map[string]interface{}) []Vehicle {
	vehicles := []Vehicle{}

	if feed == nil {
		return vehicles
	}

	data, ok := feed["data"].(map[string]interface{})
	if !ok {
		return vehicles
	}

	var vehicleList []interface{}
	if v, ok := data["vehicles"].([]interface{}); ok {
		vehicleList = v
	} else if v, ok := data["bikes"].([]interface{}); ok {
		vehicleList = v
	}

	for _, v := range vehicleList {
		vMap, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		vehicle := Vehicle{}
		vehicle.VehicleID, _ = vMap["vehicle_id"].(string)
		vehicle.BikeID, _ = vMap["bike_id"].(string)
		vehicle.Lat, _ = vMap["lat"].(float64)
		vehicle.Lon, _ = vMap["lon"].(float64)
		vehicle.IsReserved = vMap["is_reserved"]
		vehicle.IsDisabled = vMap["is_disabled"]
		vehicle.VehicleTypeID, _ = vMap["vehicle_type_id"].(string)
		if r, ok := vMap["current_range_meters"].(float64); ok {
			vehicle.CurrentRangeMeters = r
		}

		vehicles = append(vehicles, vehicle)
	}

	return vehicles
}
