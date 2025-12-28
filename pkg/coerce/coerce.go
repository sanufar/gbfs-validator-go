// Package coerce normalizes GBFS feeds for lenient validation.
package coerce

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Options selects which coercions to apply.
type Options struct {
	CoerceBooleans bool `json:"coerceBooleans"`

	CoerceTimestamps bool `json:"coerceTimestamps"`

	CoerceNumericStrings bool `json:"coerceNumericStrings"`

	CoerceCoordinates bool `json:"coerceCoordinates"`

	TreatNullAsAbsent bool `json:"treatNullAsAbsent"`

	AllowExtraFields bool `json:"allowExtraFields"`

	CoerceEmptyArrays bool `json:"coerceEmptyArrays"`
}

// DefaultLenientOptions returns permissive coercion defaults.
func DefaultLenientOptions() Options {
	return Options{
		CoerceBooleans:       true,
		CoerceTimestamps:     true,
		CoerceNumericStrings: true,
		CoerceCoordinates:    true,
		TreatNullAsAbsent:    true,
		AllowExtraFields:     true,
		CoerceEmptyArrays:    true,
	}
}

// StrictOptions returns strict coercion defaults.
func StrictOptions() Options {
	return Options{}
}

// CoercionLog records applied coercions.
type CoercionLog struct {
	Coercions []Coercion `json:"coercions"`
}

// Coercion captures a single change.
type Coercion struct {
	Path     string      `json:"path"`
	Field    string      `json:"field"`
	FromType string      `json:"fromType"`
	ToType   string      `json:"toType"`
	From     interface{} `json:"from"`
	To       interface{} `json:"to"`
}

// Result holds coerced data and the change log.
type Result struct {
	Data []byte       `json:"data"`
	Log  CoercionLog  `json:"log"`
}

// Coercer applies configured coercions.
type Coercer struct {
	opts Options
	log  CoercionLog
}

// New constructs a Coercer.
func New(opts Options) *Coercer {
	return &Coercer{
		opts: opts,
		log:  CoercionLog{Coercions: []Coercion{}},
	}
}

// Coerce normalizes JSON data for a feed type.
func (c *Coercer) Coerce(data []byte, feedType string) (*Result, error) {
	c.log = CoercionLog{Coercions: []Coercion{}}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	c.coerceCommonFields(jsonData, "")
	
	switch feedType {
	case "station_status":
		c.coerceStationStatus(jsonData)
	case "station_information":
		c.coerceStationInformation(jsonData)
	case "vehicle_status", "free_bike_status":
		c.coerceVehicleStatus(jsonData)
	case "vehicle_types":
		c.coerceVehicleTypes(jsonData)
	case "system_information":
		c.coerceSystemInformation(jsonData)
	case "geofencing_zones":
		c.coerceGeofencingZones(jsonData)
	}

	coercedData, err := json.Marshal(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize coerced data: %w", err)
	}

	return &Result{
		Data: coercedData,
		Log:  c.log,
	}, nil
}

// coerceCommonFields normalizes shared fields.
func (c *Coercer) coerceCommonFields(data map[string]interface{}, path string) {
	if c.opts.CoerceTimestamps {
		if val, ok := data["last_updated"]; ok {
			if coerced, changed := c.coerceTimestamp(val); changed {
				c.logCoercion(path, "last_updated", val, coerced)
				data["last_updated"] = coerced
			}
		}
	}

	if c.opts.CoerceNumericStrings {
		if val, ok := data["ttl"]; ok {
			if coerced, changed := c.coerceToInt(val); changed {
				c.logCoercion(path, "ttl", val, coerced)
				data["ttl"] = coerced
			}
		}
	}

	if c.opts.TreatNullAsAbsent {
		for k, v := range data {
			if v == nil {
				delete(data, k)
			}
		}
	}
}

// coerceStationStatus normalizes station_status.json.
func (c *Coercer) coerceStationStatus(data map[string]interface{}) {
	dataObj, ok := data["data"].(map[string]interface{})
	if !ok {
		return
	}

	stations, ok := dataObj["stations"].([]interface{})
	if !ok {
		return
	}

	boolFields := []string{
		"is_installed", "is_renting", "is_returning",
		"is_charging_station",
	}

	numericFields := []string{
		"num_bikes_available", "num_bikes_disabled",
		"num_docks_available", "num_docks_disabled",
		"num_vehicles_available", "num_vehicles_disabled",
		"last_reported",
	}

	for i, s := range stations {
		station, ok := s.(map[string]interface{})
		if !ok {
			continue
		}

		path := fmt.Sprintf("/data/stations/%d", i)

		if c.opts.CoerceBooleans {
			for _, field := range boolFields {
				if val, ok := station[field]; ok {
					if coerced, changed := c.coerceToBool(val); changed {
						c.logCoercion(path, field, val, coerced)
						station[field] = coerced
					}
				}
			}
		}

		if c.opts.CoerceNumericStrings {
			for _, field := range numericFields {
				if val, ok := station[field]; ok {
					if coerced, changed := c.coerceToInt(val); changed {
						c.logCoercion(path, field, val, coerced)
						station[field] = coerced
					}
				}
			}
		}

		if c.opts.CoerceTimestamps {
			if val, ok := station["last_reported"]; ok {
				if coerced, changed := c.coerceTimestamp(val); changed {
					c.logCoercion(path, "last_reported", val, coerced)
					station["last_reported"] = coerced
				}
			}
		}

		if c.opts.TreatNullAsAbsent {
			for k, v := range station {
				if v == nil {
					delete(station, k)
				}
			}
		}
	}
}

// coerceStationInformation normalizes station_information.json.
func (c *Coercer) coerceStationInformation(data map[string]interface{}) {
	dataObj, ok := data["data"].(map[string]interface{})
	if !ok {
		return
	}

	stations, ok := dataObj["stations"].([]interface{})
	if !ok {
		return
	}

	boolFields := []string{
		"is_valet_station", "is_virtual_station", "is_charging_station",
	}

	for i, s := range stations {
		station, ok := s.(map[string]interface{})
		if !ok {
			continue
		}

		path := fmt.Sprintf("/data/stations/%d", i)

		if c.opts.CoerceCoordinates {
			for _, field := range []string{"lat", "lon"} {
				if val, ok := station[field]; ok {
					if coerced, changed := c.coerceToFloat(val); changed {
						c.logCoercion(path, field, val, coerced)
						station[field] = coerced
					}
				}
			}
		}

		if c.opts.CoerceNumericStrings {
			if val, ok := station["capacity"]; ok {
				if coerced, changed := c.coerceToInt(val); changed {
					c.logCoercion(path, "capacity", val, coerced)
					station["capacity"] = coerced
				}
			}
		}

		if c.opts.CoerceBooleans {
			for _, field := range boolFields {
				if val, ok := station[field]; ok {
					if coerced, changed := c.coerceToBool(val); changed {
						c.logCoercion(path, field, val, coerced)
						station[field] = coerced
					}
				}
			}
		}
	}
}

// coerceVehicleStatus normalizes vehicle_status.json or free_bike_status.json.
func (c *Coercer) coerceVehicleStatus(data map[string]interface{}) {
	dataObj, ok := data["data"].(map[string]interface{})
	if !ok {
		return
	}

	var vehicles []interface{}
	var vehiclesKey string
	if v, ok := dataObj["vehicles"].([]interface{}); ok {
		vehicles = v
		vehiclesKey = "vehicles"
	} else if b, ok := dataObj["bikes"].([]interface{}); ok {
		vehicles = b
		vehiclesKey = "bikes"
	} else {
		return
	}

	boolFields := []string{
		"is_reserved", "is_disabled",
	}

	numericFields := []string{
		"current_range_meters", "current_fuel_percent",
		"last_reported",
	}

	for i, v := range vehicles {
		vehicle, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		path := fmt.Sprintf("/data/%s/%d", vehiclesKey, i)

		if c.opts.CoerceCoordinates {
			for _, field := range []string{"lat", "lon"} {
				if val, ok := vehicle[field]; ok {
					if coerced, changed := c.coerceToFloat(val); changed {
						c.logCoercion(path, field, val, coerced)
						vehicle[field] = coerced
					}
				}
			}
		}

		if c.opts.CoerceBooleans {
			for _, field := range boolFields {
				if val, ok := vehicle[field]; ok {
					if coerced, changed := c.coerceToBool(val); changed {
						c.logCoercion(path, field, val, coerced)
						vehicle[field] = coerced
					}
				}
			}
		}

		if c.opts.CoerceNumericStrings {
			for _, field := range numericFields {
				if val, ok := vehicle[field]; ok {
					if coerced, changed := c.coerceToInt(val); changed {
						c.logCoercion(path, field, val, coerced)
						vehicle[field] = coerced
					}
				}
			}
		}

		if c.opts.CoerceTimestamps {
			if val, ok := vehicle["last_reported"]; ok {
				if coerced, changed := c.coerceTimestamp(val); changed {
					c.logCoercion(path, "last_reported", val, coerced)
					vehicle["last_reported"] = coerced
				}
			}
		}
	}
}

// coerceVehicleTypes normalizes vehicle_types.json.
func (c *Coercer) coerceVehicleTypes(data map[string]interface{}) {
	dataObj, ok := data["data"].(map[string]interface{})
	if !ok {
		return
	}

	vehicleTypes, ok := dataObj["vehicle_types"].([]interface{})
	if !ok {
		return
	}

	numericFields := []string{
		"max_range_meters", "wheel_count", "max_permitted_speed",
		"rated_power", "default_reserve_time", "cargo_volume_capacity",
		"cargo_load_capacity",
	}

	for i, vt := range vehicleTypes {
		vehicleType, ok := vt.(map[string]interface{})
		if !ok {
			continue
		}

		path := fmt.Sprintf("/data/vehicle_types/%d", i)

		if c.opts.CoerceNumericStrings {
			for _, field := range numericFields {
				if val, ok := vehicleType[field]; ok {
					if coerced, changed := c.coerceToNumber(val); changed {
						c.logCoercion(path, field, val, coerced)
						vehicleType[field] = coerced
					}
				}
			}
		}
	}
}

// coerceSystemInformation normalizes system_information.json.
func (c *Coercer) coerceSystemInformation(data map[string]interface{}) {
	dataObj, ok := data["data"].(map[string]interface{})
	if !ok {
		return
	}

	if c.opts.CoerceTimestamps {
		for _, field := range []string{"start_date", "end_date"} {
			if val, ok := dataObj[field]; ok {
				if coerced, changed := c.coerceTimestamp(val); changed {
					c.logCoercion("/data", field, val, coerced)
					dataObj[field] = coerced
				}
			}
		}
	}
}

// coerceGeofencingZones normalizes geofencing_zones.json.
func (c *Coercer) coerceGeofencingZones(data map[string]interface{}) {
	dataObj, ok := data["data"].(map[string]interface{})
	if !ok {
		return
	}

	zonesFC, ok := dataObj["geofencing_zones"].(map[string]interface{})
	if !ok {
		return
	}

	features, ok := zonesFC["features"].([]interface{})
	if !ok {
		return
	}

	for i, f := range features {
		feature, ok := f.(map[string]interface{})
		if !ok {
			continue
		}

		props, ok := feature["properties"].(map[string]interface{})
		if !ok {
			continue
		}

		path := fmt.Sprintf("/data/geofencing_zones/features/%d/properties", i)

		if rules, ok := props["rules"].([]interface{}); ok {
			for j, r := range rules {
				rule, ok := r.(map[string]interface{})
				if !ok {
					continue
				}

				rulePath := fmt.Sprintf("%s/rules/%d", path, j)

				if c.opts.CoerceBooleans {
					if val, ok := rule["ride_through_allowed"]; ok {
						if coerced, changed := c.coerceToBool(val); changed {
							c.logCoercion(rulePath, "ride_through_allowed", val, coerced)
							rule["ride_through_allowed"] = coerced
						}
					}
				}

				if c.opts.CoerceNumericStrings {
					for _, field := range []string{"maximum_speed_kph", "station_parking"} {
						if val, ok := rule[field]; ok {
							if coerced, changed := c.coerceToNumber(val); changed {
								c.logCoercion(rulePath, field, val, coerced)
								rule[field] = coerced
							}
						}
					}
				}
			}
		}
	}
}

// coerceToBool converts a value to bool when possible.
func (c *Coercer) coerceToBool(val interface{}) (bool, bool) {
	switch v := val.(type) {
	case bool:
		return v, false
	case float64:
		return v != 0, true
	case int:
		return v != 0, true
	case int64:
		return v != 0, true
	case string:
		lower := strings.ToLower(strings.TrimSpace(v))
		switch lower {
		case "true", "1", "yes", "on":
			return true, true
		case "false", "0", "no", "off", "":
			return false, true
		}
	}
	return false, false
}

// coerceToInt converts a value to int64 when possible.
func (c *Coercer) coerceToInt(val interface{}) (int64, bool) {
	switch v := val.(type) {
	case float64:
		return int64(v), false
	case int:
		return int64(v), false
	case int64:
		return v, false
	case string:
		if i, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
			return i, true
		}
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			return int64(f), true
		}
	}
	return 0, false
}

// coerceToFloat converts a value to float64 when possible.
func (c *Coercer) coerceToFloat(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, false
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// coerceToNumber converts a value to int64 or float64 when possible.
func (c *Coercer) coerceToNumber(val interface{}) (interface{}, bool) {
	switch v := val.(type) {
	case float64, int, int64:
		return v, false
	case string:
		s := strings.TrimSpace(v)
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i, true
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f, true
		}
	}
	return val, false
}

// coerceTimestamp normalizes POSIX and RFC3339 timestamps.
func (c *Coercer) coerceTimestamp(val interface{}) (interface{}, bool) {
	switch v := val.(type) {
	case float64:
		return int64(v), v != float64(int64(v))
	case int:
		return int64(v), false
	case int64:
		return v, false
	case string:
		s := strings.TrimSpace(v)
		
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i, true
		}
		
		if _, err := time.Parse(time.RFC3339, s); err == nil {
			return s, false
		}
		
		formats := []string{
			"2006-01-02T15:04:05Z07:00",
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, s); err == nil {
				return t.Unix(), true
			}
		}
	}
	return val, false
}

// logCoercion appends a coercion record.
func (c *Coercer) logCoercion(path, field string, from, to interface{}) {
	c.log.Coercions = append(c.log.Coercions, Coercion{
		Path:     path,
		Field:    field,
		FromType: fmt.Sprintf("%T", from),
		ToType:   fmt.Sprintf("%T", to),
		From:     from,
		To:       to,
	})
}

// GetLog returns the current coercion log.
func (c *Coercer) GetLog() CoercionLog {
	return c.log
}

// CoercionSummary aggregates coercion counts.
type CoercionSummary struct {
	TotalCoercions int                       `json:"totalCoercions"`
	ByType         map[string]int            `json:"byType"`
	ByField        map[string]int            `json:"byField"`
	Details        []CoercionSummaryDetail   `json:"details,omitempty"`
}

// CoercionSummaryDetail is a grouped coercion count.
type CoercionSummaryDetail struct {
	Field    string `json:"field"`
	FromType string `json:"fromType"`
	ToType   string `json:"toType"`
	Count    int    `json:"count"`
}

// Summarize aggregates coercions into counts and details.
func (log *CoercionLog) Summarize() CoercionSummary {
	summary := CoercionSummary{
		TotalCoercions: len(log.Coercions),
		ByType:         make(map[string]int),
		ByField:        make(map[string]int),
	}

	detailMap := make(map[string]*CoercionSummaryDetail)
	
	for _, c := range log.Coercions {
		typeKey := fmt.Sprintf("%s->%s", c.FromType, c.ToType)
		summary.ByType[typeKey]++
		
		summary.ByField[c.Field]++
		
		detailKey := fmt.Sprintf("%s:%s:%s", c.Field, c.FromType, c.ToType)
		if d, ok := detailMap[detailKey]; ok {
			d.Count++
		} else {
			detailMap[detailKey] = &CoercionSummaryDetail{
				Field:    c.Field,
				FromType: c.FromType,
				ToType:   c.ToType,
				Count:    1,
			}
		}
	}

	for _, d := range detailMap {
		summary.Details = append(summary.Details, *d)
	}

	return summary
}
