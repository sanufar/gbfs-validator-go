// Package mapdata transforms GBFS data into map-ready formats.
package mapdata

import (
	"encoding/json"

	"github.com/gbfs-validator-go/pkg/gbfs"
)

// GeoJSONFeatureCollection is a GeoJSON feature collection.
type GeoJSONFeatureCollection struct {
	Type     string           `json:"type"`
	Features []GeoJSONFeature `json:"features"`
}

// GeoJSONFeature is a single GeoJSON feature.
type GeoJSONFeature struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Geometry   GeoJSONGeometry        `json:"geometry"`
}

// GeoJSONGeometry describes a GeoJSON geometry.
type GeoJSONGeometry struct {
	Type        string      `json:"type"`
	Coordinates interface{} `json:"coordinates"`
}

// MapData contains all transformed map layers.
type MapData struct {
	Stations        *GeoJSONFeatureCollection `json:"stations,omitempty"`
	Vehicles        *GeoJSONFeatureCollection `json:"vehicles,omitempty"`
	GeofencingZones *GeoJSONFeatureCollection `json:"geofencingZones,omitempty"`
	Summary         MapSummary                `json:"summary"`
}

// MapSummary aggregates counts and bounds.
type MapSummary struct {
	TotalStations           int                    `json:"totalStations"`
	TotalVehicles           int                    `json:"totalVehicles"`
	TotalVehiclesInStations int                    `json:"totalVehiclesInStations"`
	VehiclesByType          map[string]int         `json:"vehiclesByType"`
	VehicleFormFactors      []string               `json:"vehicleFormFactors"`
	HasStationDetails       bool                   `json:"hasStationDetails"`
	BoundingBox             *BoundingBox           `json:"boundingBox,omitempty"`
}

// BoundingBox is a geographic bounding box.
type BoundingBox struct {
	MinLon float64 `json:"minLon"`
	MinLat float64 `json:"minLat"`
	MaxLon float64 `json:"maxLon"`
	MaxLat float64 `json:"maxLat"`
}

// Transformer converts GBFS payloads to GeoJSON layers.
type Transformer struct {
	vehicleTypes  map[string]gbfs.VehicleType
	pricingPlans  map[string]gbfs.PricingPlan
	stationStatus map[string]gbfs.StationStatusEntry
}

// NewTransformer constructs a Transformer.
func NewTransformer() *Transformer {
	return &Transformer{
		vehicleTypes:  make(map[string]gbfs.VehicleType),
		pricingPlans:  make(map[string]gbfs.PricingPlan),
		stationStatus: make(map[string]gbfs.StationStatusEntry),
	}
}

// LoadVehicleTypes loads vehicle types for enrichment.
func (t *Transformer) LoadVehicleTypes(data []byte) error {
	var vt gbfs.VehicleTypes
	if err := json.Unmarshal(data, &vt); err != nil {
		return err
	}
	for _, v := range vt.Data.VehicleTypes {
		t.vehicleTypes[v.VehicleTypeID] = v
	}
	return nil
}

// LoadPricingPlans loads pricing plans for enrichment.
func (t *Transformer) LoadPricingPlans(data []byte) error {
	var pp gbfs.SystemPricingPlans
	if err := json.Unmarshal(data, &pp); err != nil {
		return err
	}
	for _, p := range pp.Data.Plans {
		t.pricingPlans[p.PlanID] = p
	}
	return nil
}

// LoadStationStatus loads station status for enrichment.
func (t *Transformer) LoadStationStatus(data []byte) error {
	var ss gbfs.StationStatus
	if err := json.Unmarshal(data, &ss); err != nil {
		return err
	}
	for _, s := range ss.Data.Stations {
		t.stationStatus[s.StationID] = s
	}
	return nil
}

// TransformStations converts station_information.json to GeoJSON.
func (t *Transformer) TransformStations(data []byte) (*GeoJSONFeatureCollection, error) {
	var si gbfs.StationInformation
	if err := json.Unmarshal(data, &si); err != nil {
		return nil, err
	}

	fc := &GeoJSONFeatureCollection{
		Type:     "FeatureCollection",
		Features: make([]GeoJSONFeature, 0, len(si.Data.Stations)),
	}

	for _, station := range si.Data.Stations {
		props := map[string]interface{}{
			"station_id": station.StationID,
			"name":       extractText(station.Name),
			"capacity":   station.Capacity,
		}

		if status, ok := t.stationStatus[station.StationID]; ok {
			props["num_bikes_available"] = status.NumBikesAvailable
			props["num_vehicles_available"] = status.NumVehiclesAvailable
			props["num_docks_available"] = status.NumDocksAvailable
			props["is_installed"] = status.IsInstalled
			props["is_renting"] = status.IsRenting
			props["is_returning"] = status.IsReturning
			
			available := status.NumBikesAvailable
			if status.NumVehiclesAvailable > 0 {
				available = status.NumVehiclesAvailable
			}
			props["vehicles_available"] = available
		}

		if station.Address != "" {
			props["address"] = station.Address
		}

		feature := GeoJSONFeature{
			Type:       "Feature",
			Properties: props,
			Geometry: GeoJSONGeometry{
				Type:        "Point",
				Coordinates: []float64{station.Lon, station.Lat},
			},
		}
		fc.Features = append(fc.Features, feature)
	}

	return fc, nil
}

// TransformVehicles converts vehicle status feeds to GeoJSON.
func (t *Transformer) TransformVehicles(data []byte) (*GeoJSONFeatureCollection, error) {
	var vs gbfs.VehicleStatus
	if err := json.Unmarshal(data, &vs); err != nil {
		return nil, err
	}

	vehicles := vs.Data.GetVehicles()
	fc := &GeoJSONFeatureCollection{
		Type:     "FeatureCollection",
		Features: make([]GeoJSONFeature, 0, len(vehicles)),
	}

	for _, vehicle := range vehicles {
		if vehicle.Lat == 0 && vehicle.Lon == 0 {
			continue
		}

		props := map[string]interface{}{
			"vehicle_id":  vehicle.GetID(),
			"is_reserved": vehicle.IsReserved,
			"is_disabled": vehicle.IsDisabled,
		}

		if vt, ok := t.vehicleTypes[vehicle.VehicleTypeID]; ok {
			props["vehicle_type_id"] = vehicle.VehicleTypeID
			props["form_factor"] = vt.FormFactor
			props["propulsion_type"] = vt.PropulsionType
			props["vehicle_type_name"] = extractText(vt.Name)
		}

		if vehicle.CurrentRangeMeters > 0 {
			props["current_range_meters"] = vehicle.CurrentRangeMeters
		}
		if vehicle.CurrentFuelPercent > 0 {
			props["current_fuel_percent"] = vehicle.CurrentFuelPercent
		}

		if vehicle.PricingPlanID != "" {
			props["pricing_plan_id"] = vehicle.PricingPlanID
			if pp, ok := t.pricingPlans[vehicle.PricingPlanID]; ok {
				props["pricing_plan_name"] = extractText(pp.Name)
				props["price"] = pp.Price
				props["currency"] = pp.Currency
			}
		}

		feature := GeoJSONFeature{
			Type:       "Feature",
			Properties: props,
			Geometry: GeoJSONGeometry{
				Type:        "Point",
				Coordinates: []float64{vehicle.Lon, vehicle.Lat},
			},
		}
		fc.Features = append(fc.Features, feature)
	}

	return fc, nil
}

// TransformGeofencingZones converts geofencing zones to GeoJSON.
func (t *Transformer) TransformGeofencingZones(data []byte) (*GeoJSONFeatureCollection, error) {
	var gz gbfs.GeofencingZones
	if err := json.Unmarshal(data, &gz); err != nil {
		return nil, err
	}

	fc := &GeoJSONFeatureCollection{
		Type:     "FeatureCollection",
		Features: make([]GeoJSONFeature, 0, len(gz.Data.GeofencingZones.Features)),
	}

	for _, feature := range gz.Data.GeofencingZones.Features {
		props := make(map[string]interface{})
		props["name"] = extractText(feature.Properties.Name)
		if len(feature.Properties.Rules) > 0 {
			props["rules"] = feature.Properties.Rules
		}
		if feature.Properties.Start != "" {
			props["start"] = feature.Properties.Start
		}
		if feature.Properties.End != "" {
			props["end"] = feature.Properties.End
		}

		geoFeature := GeoJSONFeature{
			Type:       "Feature",
			Properties: props,
			Geometry: GeoJSONGeometry{
				Type:        feature.Geometry.Type,
				Coordinates: feature.Geometry.Coordinates,
			},
		}
		fc.Features = append(fc.Features, geoFeature)
	}

	return fc, nil
}

// CalculateSummary computes counts and bounds for map layers.
func (t *Transformer) CalculateSummary(stations, vehicles *GeoJSONFeatureCollection) MapSummary {
	summary := MapSummary{
		VehiclesByType:     make(map[string]int),
		VehicleFormFactors: []string{},
	}

	if stations != nil {
		summary.TotalStations = len(stations.Features)
		summary.HasStationDetails = len(t.stationStatus) > 0

		for _, status := range t.stationStatus {
			available := status.NumBikesAvailable
			if status.NumVehiclesAvailable > 0 {
				available = status.NumVehiclesAvailable
			}
			summary.TotalVehiclesInStations += available
		}
	}

	if vehicles != nil {
		summary.TotalVehicles = len(vehicles.Features)

		formFactors := make(map[string]bool)
		for _, feature := range vehicles.Features {
			if ff, ok := feature.Properties["form_factor"].(string); ok {
				summary.VehiclesByType[ff]++
				formFactors[ff] = true
			} else {
				summary.VehiclesByType["unknown"]++
			}
		}

		for ff := range formFactors {
			summary.VehicleFormFactors = append(summary.VehicleFormFactors, ff)
		}
	}

	summary.BoundingBox = t.calculateBounds(stations, vehicles)

	return summary
}

// calculateBounds computes a bounding box for all features.
func (t *Transformer) calculateBounds(stations, vehicles *GeoJSONFeatureCollection) *BoundingBox {
	if (stations == nil || len(stations.Features) == 0) &&
		(vehicles == nil || len(vehicles.Features) == 0) {
		return nil
	}

	bbox := &BoundingBox{
		MinLon: 180,
		MinLat: 90,
		MaxLon: -180,
		MaxLat: -90,
	}

	updateBounds := func(coords []float64) {
		if len(coords) >= 2 {
			lon, lat := coords[0], coords[1]
			if lon < bbox.MinLon {
				bbox.MinLon = lon
			}
			if lon > bbox.MaxLon {
				bbox.MaxLon = lon
			}
			if lat < bbox.MinLat {
				bbox.MinLat = lat
			}
			if lat > bbox.MaxLat {
				bbox.MaxLat = lat
			}
		}
	}

	if stations != nil {
		for _, f := range stations.Features {
			if coords, ok := f.Geometry.Coordinates.([]float64); ok {
				updateBounds(coords)
			}
		}
	}

	if vehicles != nil {
		for _, f := range vehicles.Features {
			if coords, ok := f.Geometry.Coordinates.([]float64); ok {
				updateBounds(coords)
			}
		}
	}

	return bbox
}

// extractText reads a plain string or localized string array.
func extractText(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case []interface{}:
		for _, item := range t {
			if m, ok := item.(map[string]interface{}); ok {
				if text, ok := m["text"].(string); ok {
					return text
				}
			}
		}
	case []gbfs.LocalizedString:
		if len(t) > 0 {
			return t[0].Text
		}
	}
	return ""
}

// GetVehicleColor returns an RGB color for a form factor.
func GetVehicleColor(formFactor string) []int {
	switch formFactor {
	case "bicycle", "cargo_bicycle":
		return []int{106, 76, 147}
	case "scooter", "scooter_standing", "scooter_seated":
		return []int{25, 130, 196}
	case "moped":
		return []int{138, 201, 38}
	case "car":
		return []int{255, 202, 58}
	default:
		return []int{180, 180, 180}
	}
}

// GetStationColor returns an RGB color based on availability.
func GetStationColor(vehiclesAvailable int) []int {
	if vehiclesAvailable > 5 {
		return []int{6, 156, 86}
	} else if vehiclesAvailable > 0 {
		return []int{255, 152, 14}
	}
	return []int{211, 33, 44}
}
