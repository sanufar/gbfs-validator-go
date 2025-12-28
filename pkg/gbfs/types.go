// Package gbfs defines GBFS feed types.
package gbfs

import (
	"encoding/json"
	"time"
)

// Version identifies a supported GBFS version.
type Version string

const (
	V1_0     Version = "1.0"
	V1_1     Version = "1.1"
	V2_0     Version = "2.0"
	V2_1     Version = "2.1"
	V2_2     Version = "2.2"
	V2_3     Version = "2.3"
	V3_0     Version = "3.0"
	V3_1_RC2 Version = "3.1-RC2"
)

// Timestamp parses POSIX or RFC3339 timestamps.
type Timestamp struct {
	Time   time.Time
	IsUnix bool
}

// UnmarshalJSON accepts RFC3339 strings or POSIX ints.
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		parsed, err := time.Parse(time.RFC3339, str)
		if err == nil {
			t.Time = parsed
			t.IsUnix = false
			return nil
		}
	}

	var unix int64
	if err := json.Unmarshal(data, &unix); err == nil {
		t.Time = time.Unix(unix, 0)
		t.IsUnix = true
		return nil
	}

	return json.Unmarshal(data, &t.Time)
}

// MarshalJSON outputs POSIX ints or RFC3339 strings.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	if t.IsUnix {
		return json.Marshal(t.Time.Unix())
	}
	return json.Marshal(t.Time.Format(time.RFC3339))
}

// LocalizedString pairs text with a language tag.
type LocalizedString struct {
	Text     string `json:"text"`
	Language string `json:"language"`
}

// CommonHeader contains fields shared by GBFS files.
type CommonHeader struct {
	LastUpdated Timestamp `json:"last_updated"`
	TTL         int       `json:"ttl"`
	Version     string    `json:"version"`
}

// GBFSFeed represents the gbfs.json autodiscovery file.
type GBFSFeed struct {
	CommonHeader
	Data GBFSData `json:"data"`
}

// GBFSData handles v2 language maps and v3 feed lists.
type GBFSData struct {
	Feeds []FeedInfo `json:"feeds,omitempty"`

	Languages map[string]LanguageFeeds `json:"-"`
}

// UnmarshalJSON supports v2 and v3 formats.
func (d *GBFSData) UnmarshalJSON(data []byte) error {
	type v3Format struct {
		Feeds []FeedInfo `json:"feeds"`
	}
	var v3 v3Format
	if err := json.Unmarshal(data, &v3); err == nil && len(v3.Feeds) > 0 {
		d.Feeds = v3.Feeds
		return nil
	}

	var v2 map[string]LanguageFeeds
	if err := json.Unmarshal(data, &v2); err == nil {
		d.Languages = v2
		for _, langFeeds := range v2 {
			d.Feeds = langFeeds.Feeds
			break
		}
		return nil
	}

	return nil
}

// LanguageFeeds groups feeds by language.
type LanguageFeeds struct {
	Feeds []FeedInfo `json:"feeds"`
}

// FeedInfo is a name/URL pair from autodiscovery.
type FeedInfo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// SystemInformation represents system_information.json.
type SystemInformation struct {
	CommonHeader
	Data SystemInfoData `json:"data"`
}

// SystemInfoData contains system details.
type SystemInfoData struct {
	SystemID   string `json:"system_id"`
	Language   string `json:"language,omitempty"`
	Languages  []string `json:"languages,omitempty"`
	
	Name             interface{} `json:"name"`
	ShortName        interface{} `json:"short_name,omitempty"`
	Operator         interface{} `json:"operator,omitempty"`
	
	URL              string `json:"url,omitempty"`
	PurchaseURL      string `json:"purchase_url,omitempty"`
	StartDate        string `json:"start_date,omitempty"`
	PhoneNumber      string `json:"phone_number,omitempty"`
	Email            string `json:"email,omitempty"`
	FeedContactEmail string `json:"feed_contact_email,omitempty"`
	Timezone         string `json:"timezone"`
	LicenseID        string `json:"license_id,omitempty"`
	LicenseURL       string `json:"license_url,omitempty"`
	OpeningHours     string `json:"opening_hours,omitempty"`
	
	BrandAssets *BrandAssets `json:"brand_assets,omitempty"`
	RentalApps  *RentalApps  `json:"rental_apps,omitempty"`
	
	ManifestURL string `json:"manifest_url,omitempty"`
}

// BrandAssets describes system branding assets.
type BrandAssets struct {
	BrandLastModified string `json:"brand_last_modified"`
	BrandImageURL     string `json:"brand_image_url"`
	BrandImageURLDark string `json:"brand_image_url_dark,omitempty"`
	Color             string `json:"color,omitempty"`
	BrandTermsURL     string `json:"brand_terms_url,omitempty"`
}

// RentalApps lists platform app info.
type RentalApps struct {
	Android *AppInfo `json:"android,omitempty"`
	IOS     *AppInfo `json:"ios,omitempty"`
}

// AppInfo identifies an app store and discovery URL.
type AppInfo struct {
	StoreURI     string `json:"store_uri,omitempty"`
	DiscoveryURI string `json:"discovery_uri,omitempty"`
}

// StationInformation represents station_information.json.
type StationInformation struct {
	CommonHeader
	Data StationInfoData `json:"data"`
}

// StationInfoData wraps station list data.
type StationInfoData struct {
	Stations []Station `json:"stations"`
}

// Station describes a station entry.
type Station struct {
	StationID     string      `json:"station_id"`
	Name          interface{} `json:"name"`
	ShortName     interface{} `json:"short_name,omitempty"`
	Lat           float64     `json:"lat"`
	Lon           float64     `json:"lon"`
	Address       string      `json:"address,omitempty"`
	CrossStreet   string      `json:"cross_street,omitempty"`
	RegionID      string      `json:"region_id,omitempty"`
	PostCode      string      `json:"post_code,omitempty"`
	RentalMethods []string    `json:"rental_methods,omitempty"`
	Capacity      int         `json:"capacity,omitempty"`
	
	IsVirtualStation   bool        `json:"is_virtual_station,omitempty"`
	StationArea        *GeoJSON    `json:"station_area,omitempty"`
	ParkingType        string      `json:"parking_type,omitempty"`
	ParkingHoop        bool        `json:"parking_hoop,omitempty"`
	ContactPhone       string      `json:"contact_phone,omitempty"`
	IsValetStation     bool        `json:"is_valet_station,omitempty"`
	IsChargingStation  bool        `json:"is_charging_station,omitempty"`
	
	VehicleTypesCapacity []VehicleTypeCapacity `json:"vehicle_types_capacity,omitempty"`
	VehicleDocksCapacity []VehicleTypeCapacity `json:"vehicle_docks_capacity,omitempty"`
	
	RentalURIs *RentalURIs `json:"rental_uris,omitempty"`
	
	StationOpeningHours string `json:"station_opening_hours,omitempty"`
}

// VehicleTypeCapacity assigns capacity per vehicle type.
type VehicleTypeCapacity struct {
	VehicleTypeIDs []string `json:"vehicle_type_ids"`
	Count          int      `json:"count"`
}

// RentalURIs provides platform-specific rental URLs.
type RentalURIs struct {
	Android string `json:"android,omitempty"`
	IOS     string `json:"ios,omitempty"`
	Web     string `json:"web,omitempty"`
}

// GeoJSON holds a generic geometry value.
type GeoJSON struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

// StationStatus represents station_status.json.
type StationStatus struct {
	CommonHeader
	Data StationStatusData `json:"data"`
}

// StationStatusData wraps station status entries.
type StationStatusData struct {
	Stations []StationStatusEntry `json:"stations"`
}

// StationStatusEntry describes a station's status.
type StationStatusEntry struct {
	StationID             string    `json:"station_id"`
	NumBikesAvailable     int       `json:"num_bikes_available,omitempty"`
	NumVehiclesAvailable  int       `json:"num_vehicles_available,omitempty"`
	NumBikesDisabled      int       `json:"num_bikes_disabled,omitempty"`
	NumVehiclesDisabled   int       `json:"num_vehicles_disabled,omitempty"`
	NumDocksAvailable     int       `json:"num_docks_available,omitempty"`
	NumDocksDisabled      int       `json:"num_docks_disabled,omitempty"`
	IsInstalled           bool      `json:"is_installed"`
	IsRenting             bool      `json:"is_renting"`
	IsReturning           bool      `json:"is_returning"`
	LastReported          Timestamp `json:"last_reported"`
	
	VehicleTypesAvailable []VehicleTypeAvailable `json:"vehicle_types_available,omitempty"`
	VehicleDocksAvailable []VehicleDockAvailable `json:"vehicle_docks_available,omitempty"`
}

// VehicleTypeAvailable counts vehicles by type.
type VehicleTypeAvailable struct {
	VehicleTypeID string `json:"vehicle_type_id"`
	Count         int    `json:"count"`
}

// VehicleDockAvailable counts docks by type.
type VehicleDockAvailable struct {
	VehicleTypeIDs []string `json:"vehicle_type_ids"`
	Count          int      `json:"count"`
}

// VehicleTypes represents vehicle_types.json.
type VehicleTypes struct {
	CommonHeader
	Data VehicleTypesData `json:"data"`
}

// VehicleTypesData wraps vehicle type entries.
type VehicleTypesData struct {
	VehicleTypes []VehicleType `json:"vehicle_types"`
}

// VehicleType describes a vehicle type.
type VehicleType struct {
	VehicleTypeID        string      `json:"vehicle_type_id"`
	FormFactor           string      `json:"form_factor"`
	RiderCapacity        int         `json:"rider_capacity,omitempty"`
	CargoVolumeCapacity  int         `json:"cargo_volume_capacity,omitempty"`
	CargoLoadCapacity    int         `json:"cargo_load_capacity,omitempty"`
	PropulsionType       string      `json:"propulsion_type"`
	EcoLabels            []EcoLabel  `json:"eco_labels,omitempty"`
	MaxRangeMeters       float64     `json:"max_range_meters,omitempty"`
	Name                 interface{} `json:"name"` // string (v2.x) or []LocalizedString (v3.0+)
	VehicleAccessories   []string    `json:"vehicle_accessories,omitempty"`
	GCO2Km               int         `json:"g_CO2_km,omitempty"`
	VehicleImage         string      `json:"vehicle_image,omitempty"`
	Make                 interface{} `json:"make,omitempty"`
	Model                interface{} `json:"model,omitempty"`
	Color                string      `json:"color,omitempty"`
	Description          interface{} `json:"description,omitempty"`
	WheelCount           int         `json:"wheel_count,omitempty"`
	MaxPermittedSpeed    int         `json:"max_permitted_speed,omitempty"`
	RatedPower           int         `json:"rated_power,omitempty"`
	DefaultReserveTime   int         `json:"default_reserve_time,omitempty"`
	ReturnConstraint     string      `json:"return_constraint,omitempty"`
	ReturnType           []string    `json:"return_type,omitempty"`
	VehicleAssets        *VehicleAssets `json:"vehicle_assets,omitempty"`
	DefaultPricingPlanID string      `json:"default_pricing_plan_id,omitempty"`
	PricingPlanIDs       []string    `json:"pricing_plan_ids,omitempty"`
}

// EcoLabel contains eco label data.
type EcoLabel struct {
	CountryCode string `json:"country_code"`
	EcoSticker  string `json:"eco_sticker"`
}

// VehicleAssets provides URLs for vehicle assets.
type VehicleAssets struct {
	IconURL          string `json:"icon_url"`
	IconURLDark      string `json:"icon_url_dark,omitempty"`
	IconLastModified string `json:"icon_last_modified"`
}

// VehicleStatus represents vehicle_status.json or free_bike_status.json.
type VehicleStatus struct {
	CommonHeader
	Data VehicleStatusData `json:"data"`
}

// VehicleStatusData contains vehicles or bikes arrays.
type VehicleStatusData struct {
	Vehicles []Vehicle `json:"vehicles,omitempty"`
	Bikes    []Vehicle `json:"bikes,omitempty"`
}

// GetVehicles returns vehicles for any version.
func (d *VehicleStatusData) GetVehicles() []Vehicle {
	if len(d.Vehicles) > 0 {
		return d.Vehicles
	}
	return d.Bikes
}

// Vehicle describes a vehicle status entry.
type Vehicle struct {
	VehicleID          string    `json:"vehicle_id,omitempty"`
	BikeID             string    `json:"bike_id,omitempty"`
	Lat                float64   `json:"lat,omitempty"`
	Lon                float64   `json:"lon,omitempty"`
	IsReserved         bool      `json:"is_reserved"`
	IsDisabled         bool      `json:"is_disabled"`
	RentalURIs         *RentalURIs `json:"rental_uris,omitempty"`
	VehicleTypeID      string    `json:"vehicle_type_id,omitempty"`
	LastReported       Timestamp `json:"last_reported,omitempty"`
	CurrentRangeMeters float64   `json:"current_range_meters,omitempty"`
	CurrentFuelPercent float64   `json:"current_fuel_percent,omitempty"`
	StationID          string    `json:"station_id,omitempty"`
	HomeStationID      string    `json:"home_station_id,omitempty"`
	PricingPlanID      string    `json:"pricing_plan_id,omitempty"`
	VehicleEquipment   []string  `json:"vehicle_equipment,omitempty"`
	AvailableUntil     string    `json:"available_until,omitempty"`
}

// GetID returns a version-agnostic vehicle identifier.
func (v *Vehicle) GetID() string {
	if v.VehicleID != "" {
		return v.VehicleID
	}
	return v.BikeID
}

// SystemPricingPlans represents system_pricing_plans.json.
type SystemPricingPlans struct {
	CommonHeader
	Data PricingPlansData `json:"data"`
}

// PricingPlansData wraps pricing plan entries.
type PricingPlansData struct {
	Plans []PricingPlan `json:"plans"`
}

// PricingPlan describes a pricing plan.
type PricingPlan struct {
	PlanID                   string      `json:"plan_id"`
	URL                      string      `json:"url,omitempty"`
	Name                     interface{} `json:"name"`
	Currency                 string      `json:"currency"`
	Price                    float64     `json:"price"`
	IsTaxable                bool        `json:"is_taxable"`
	Description              interface{} `json:"description"`
	PerKmPricing             []PricingSegment `json:"per_km_pricing,omitempty"`
	PerMinPricing            []PricingSegment `json:"per_min_pricing,omitempty"`
	SurgePricing             bool        `json:"surge_pricing,omitempty"`
	ReservationPriceFlatRate float64     `json:"reservation_price_flat_rate,omitempty"`
	ReservationPricePerMin   float64     `json:"reservation_price_per_min,omitempty"`
}

// PricingSegment is a tiered pricing segment.
type PricingSegment struct {
	Start    int     `json:"start"`
	Rate     float64 `json:"rate"`
	Interval int     `json:"interval"`
	End      int     `json:"end,omitempty"`
}

// GeofencingZones represents geofencing_zones.json.
type GeofencingZones struct {
	CommonHeader
	Data GeofencingData `json:"data"`
}

// GeofencingData wraps geofencing zones and global rules.
type GeofencingData struct {
	GeofencingZones GeoJSONFeatureCollection `json:"geofencing_zones"`
	GlobalRules     []GeofencingRule         `json:"global_rules,omitempty"`
}

// GeoJSONFeatureCollection is a GeoJSON feature collection.
type GeoJSONFeatureCollection struct {
	Type     string           `json:"type"`
	Features []GeoJSONFeature `json:"features"`
}

// GeoJSONFeature is a GeoJSON feature.
type GeoJSONFeature struct {
	Type       string                 `json:"type"`
	Geometry   GeoJSON                `json:"geometry"`
	Properties GeofencingProperties   `json:"properties"`
}

// GeofencingProperties describes a geofence feature.
type GeofencingProperties struct {
	Name  interface{}      `json:"name,omitempty"`
	Start string           `json:"start,omitempty"`
	End   string           `json:"end,omitempty"`
	Rules []GeofencingRule `json:"rules,omitempty"`
}

// GeofencingRule defines rules for a geofence.
type GeofencingRule struct {
	VehicleTypeIDs    []string `json:"vehicle_type_ids,omitempty"`
	RideStartAllowed  bool     `json:"ride_start_allowed"`
	RideEndAllowed    bool     `json:"ride_end_allowed"`
	RideThroughAllowed bool    `json:"ride_through_allowed"`
	MaximumSpeedKph   int      `json:"maximum_speed_kph,omitempty"`
	StationParking    bool     `json:"station_parking,omitempty"`
}

// SystemRegions represents system_regions.json.
type SystemRegions struct {
	CommonHeader
	Data RegionsData `json:"data"`
}

// RegionsData wraps region entries.
type RegionsData struct {
	Regions []Region `json:"regions"`
}

// Region describes a system region.
type Region struct {
	RegionID string      `json:"region_id"`
	Name     interface{} `json:"name"`
}

// SystemAlerts represents system_alerts.json.
type SystemAlerts struct {
	CommonHeader
	Data AlertsData `json:"data"`
}

// AlertsData wraps alert entries.
type AlertsData struct {
	Alerts []Alert `json:"alerts"`
}

// Alert describes a system alert.
type Alert struct {
	AlertID     string       `json:"alert_id"`
	Type        string       `json:"type"`
	Times       []AlertTime  `json:"times,omitempty"`
	StationIDs  []string     `json:"station_ids,omitempty"`
	RegionIDs   []string     `json:"region_ids,omitempty"`
	URL         interface{}  `json:"url,omitempty"`
	Summary     interface{}  `json:"summary"`
	Description interface{}  `json:"description,omitempty"`
	LastUpdated Timestamp    `json:"last_updated,omitempty"`
}

// AlertTime defines a start/end window for an alert.
type AlertTime struct {
	Start Timestamp `json:"start"`
	End   Timestamp `json:"end,omitempty"`
}

// Manifest represents manifest.json.
type Manifest struct {
	CommonHeader
	Data ManifestData `json:"data"`
}

// ManifestData wraps dataset entries.
type ManifestData struct {
	Datasets []Dataset `json:"datasets"`
}

// Dataset groups system datasets by version.
type Dataset struct {
	SystemID string           `json:"system_id"`
	Versions []DatasetVersion `json:"versions"`
}

// DatasetVersion links a version to its URL.
type DatasetVersion struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

// GBFSVersions represents gbfs_versions.json.
type GBFSVersions struct {
	CommonHeader
	Data VersionsData `json:"data"`
}

// VersionsData wraps version entries.
type VersionsData struct {
	Versions []VersionInfo `json:"versions"`
}

// VersionInfo pairs a version with its URL.
type VersionInfo struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}
