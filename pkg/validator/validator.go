// Package validator validates GBFS feeds.
package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/gbfs-validator-go/pkg/coerce"
	"github.com/gbfs-validator-go/pkg/fetcher"
	"github.com/gbfs-validator-go/pkg/gbfs"
	"github.com/gbfs-validator-go/pkg/version"
)

// ValidationSeverity indicates the severity of a validation issue.
type ValidationSeverity string

const (
	SeverityError   ValidationSeverity = "error"
	SeverityWarning ValidationSeverity = "warning"
	SeverityInfo    ValidationSeverity = "info"
)

// ValidationError represents a single validation issue.
type ValidationError struct {
	Severity     ValidationSeverity `json:"severity"`
	Message      string             `json:"message"`
	InstancePath string             `json:"instancePath,omitempty"`
	SchemaPath   string             `json:"schemaPath,omitempty"`
	Keyword      string             `json:"keyword,omitempty"`
}

// FileValidationResult holds validation results for a file.
type FileValidationResult struct {
	File           string            `json:"file"`
	URL            string            `json:"url,omitempty"`
	Required       bool              `json:"required"`
	Recommended    bool              `json:"recommended,omitempty"`
	Exists         bool              `json:"exists"`
	HasErrors      bool              `json:"hasErrors"`
	ErrorsCount    int               `json:"errorsCount"`
	Errors         []ValidationError `json:"errors,omitempty"`
	RawData        json.RawMessage   `json:"-"`
	CoercedData    json.RawMessage   `json:"-"`
	CoercionCount  int               `json:"coercionCount,omitempty"`
}

// ValidationSummary summarizes a validation run.
type ValidationSummary struct {
	ValidatorVersion     string           `json:"validatorVersion"`
	Version              VersionInfo      `json:"version"`
	HasErrors            bool             `json:"hasErrors"`
	ErrorsCount          int              `json:"errorsCount"`
	VersionUnimplemented bool             `json:"versionUnimplemented,omitempty"`
	LenientMode          bool             `json:"lenientMode,omitempty"`
	CoercionSummary      *CoercionSummary `json:"coercionSummary,omitempty"`
}

// CoercionSummary summarizes applied coercions.
type CoercionSummary struct {
	TotalCoercions int            `json:"totalCoercions"`
	ByField        map[string]int `json:"byField"`
}

// VersionInfo tracks detected and validated versions.
type VersionInfo struct {
	Detected  string `json:"detected"`
	Validated string `json:"validated"`
}

// ValidationResult is the full validation output.
type ValidationResult struct {
	Summary ValidationSummary      `json:"summary"`
	Files   []FileValidationResult `json:"files"`
}

// Options configures validator behavior.
type Options struct {
	Docked       bool   `json:"docked"`
	Freefloating bool   `json:"freefloating"`
	Version      string `json:"version"`
	
	LenientMode bool `json:"lenientMode"`
	
	CoerceOptions *CoerceOptions `json:"coerceOptions,omitempty"`
}

// CoerceOptions selects coercions for lenient mode.
type CoerceOptions struct {
	CoerceBooleans bool `json:"coerceBooleans"`

	CoerceTimestamps bool `json:"coerceTimestamps"`

	CoerceNumericStrings bool `json:"coerceNumericStrings"`

	CoerceCoordinates bool `json:"coerceCoordinates"`

	TreatNullAsAbsent bool `json:"treatNullAsAbsent"`
}

// DefaultCoerceOptions returns the default lenient coercions.
func DefaultCoerceOptions() *CoerceOptions {
	return &CoerceOptions{
		CoerceBooleans:       true,
		CoerceTimestamps:     true,
		CoerceNumericStrings: true,
		CoerceCoordinates:    true,
		TreatNullAsAbsent:    true,
	}
}

// Validator validates GBFS feeds.
type Validator struct {
	fetcher *fetcher.Fetcher
	options Options
	coercer *coerce.Coercer
}

// New constructs a Validator.
func New(f *fetcher.Fetcher, opts Options) *Validator {
	v := &Validator{
		fetcher: f,
		options: opts,
	}
	
	if opts.LenientMode {
		coerceOpts := coerce.DefaultLenientOptions()
		if opts.CoerceOptions != nil {
			coerceOpts = coerce.Options{
				CoerceBooleans:       opts.CoerceOptions.CoerceBooleans,
				CoerceTimestamps:     opts.CoerceOptions.CoerceTimestamps,
				CoerceNumericStrings: opts.CoerceOptions.CoerceNumericStrings,
				CoerceCoordinates:    opts.CoerceOptions.CoerceCoordinates,
				TreatNullAsAbsent:    opts.CoerceOptions.TreatNullAsAbsent,
			}
		}
		v.coercer = coerce.New(coerceOpts)
	}
	
	return v
}

// Validate performs a full feed validation.
func (v *Validator) Validate(ctx context.Context, gbfsURL string) (*ValidationResult, error) {
	result := &ValidationResult{
		Summary: ValidationSummary{
			ValidatorVersion: "1.0.0",
			LenientMode:      v.options.LenientMode,
		},
		Files: []FileValidationResult{},
	}

	gbfsResult, gbfsFeed, err := v.validateGBFS(ctx, gbfsURL)
	if err != nil || gbfsFeed == nil {
		if gbfsResult != nil {
			result.Files = append(result.Files, *gbfsResult)
		}
		result.Summary.VersionUnimplemented = true
		return result, nil
	}

	result.Files = append(result.Files, *gbfsResult)

	detectedVersion := gbfsFeed.Version
	if detectedVersion == "" {
		detectedVersion = "1.0"
	}
	validatedVersion := v.options.Version
	if validatedVersion == "" {
		validatedVersion = detectedVersion
	}

	result.Summary.Version = VersionInfo{
		Detected:  detectedVersion,
		Validated: validatedVersion,
	}

	feedURLs := v.buildFeedURLMap(gbfsFeed, gbfsURL)

	requirements := version.GetFileRequirements(validatedVersion, version.Options{
		Docked:       v.options.Docked,
		Freefloating: v.options.Freefloating,
	})

	fileResults := v.validateFiles(ctx, feedURLs, requirements, validatedVersion)

	v.crossValidate(fileResults, validatedVersion)

	totalCoercions := 0
	coercionsByField := make(map[string]int)
	
	for _, fr := range fileResults {
		result.Files = append(result.Files, *fr)
		if fr.HasErrors {
			result.Summary.HasErrors = true
		}
		result.Summary.ErrorsCount += fr.ErrorsCount
		
		if fr.CoercionCount > 0 {
			totalCoercions += fr.CoercionCount
		}
	}

	if gbfsResult.HasErrors {
		result.Summary.HasErrors = true
	}

	if v.options.LenientMode && totalCoercions > 0 {
		result.Summary.CoercionSummary = &CoercionSummary{
			TotalCoercions: totalCoercions,
			ByField:        coercionsByField,
		}
	}

	return result, nil
}

// validateGBFS fetches and validates gbfs.json.
func (v *Validator) validateGBFS(ctx context.Context, gbfsURL string) (*FileValidationResult, *gbfs.GBFSFeed, error) {
	result := &FileValidationResult{
		File:        "gbfs.json",
		URL:         gbfsURL,
		Required:    true,
		Recommended: true,
	}

	fetchResult := v.fetcher.Fetch(ctx, gbfsURL)
	if fetchResult.Error != nil {
		if !strings.HasSuffix(gbfsURL, "gbfs.json") {
			altURL := fetcher.BuildFeedURL(gbfsURL, "gbfs")
			fetchResult = v.fetcher.Fetch(ctx, altURL)
			if fetchResult.Error == nil {
				result.URL = altURL
			}
		}
	}

	if fetchResult.Error != nil || !fetchResult.Exists {
		result.Exists = false
		if version.IsGBFSRequired(v.options.Version) {
			result.HasErrors = true
			result.ErrorsCount = 1
			result.Errors = []ValidationError{{
				Severity: SeverityError,
				Message:  "gbfs.json is required but not found",
			}}
		}
		return result, nil, fmt.Errorf("gbfs.json not found")
	}

	result.Exists = true
	result.RawData = fetchResult.Body

	var feed gbfs.GBFSFeed
	if err := json.Unmarshal(fetchResult.Body, &feed); err != nil {
		result.HasErrors = true
		result.ErrorsCount = 1
		result.Errors = []ValidationError{{
			Severity: SeverityError,
			Message:  fmt.Sprintf("Failed to parse gbfs.json: %v", err),
		}}
		return result, nil, err
	}

	schemaErrors := v.validateGBFSStructure(&feed)
	if len(schemaErrors) > 0 {
		result.HasErrors = true
		result.Errors = schemaErrors
		result.ErrorsCount = len(schemaErrors)
	}

	return result, &feed, nil
}

// validateGBFSStructure checks gbfs.json structure.
func (v *Validator) validateGBFSStructure(feed *gbfs.GBFSFeed) []ValidationError {
	var errors []ValidationError

	if feed.TTL < 0 {
		errors = append(errors, ValidationError{
			Severity:     SeverityError,
			Message:      "ttl must be non-negative",
			InstancePath: "/ttl",
		})
	}

	if len(feed.Data.Feeds) == 0 {
		errors = append(errors, ValidationError{
			Severity:     SeverityError,
			Message:      "data.feeds array is required and must not be empty",
			InstancePath: "/data/feeds",
		})
	}

	for i, f := range feed.Data.Feeds {
		if f.Name == "" {
			errors = append(errors, ValidationError{
				Severity:     SeverityError,
				Message:      "feed name is required",
				InstancePath: fmt.Sprintf("/data/feeds/%d/name", i),
			})
		}
		if f.URL == "" {
			errors = append(errors, ValidationError{
				Severity:     SeverityError,
				Message:      "feed url is required",
				InstancePath: fmt.Sprintf("/data/feeds/%d/url", i),
			})
		}
	}

	return errors
}

// buildFeedURLMap maps feed names to URLs.
func (v *Validator) buildFeedURLMap(feed *gbfs.GBFSFeed, baseURL string) map[string]string {
	urls := make(map[string]string)

	for _, f := range feed.Data.Feeds {
		urls[f.Name] = f.URL
	}

	return urls
}

// validateFiles fetches and validates required files.
func (v *Validator) validateFiles(ctx context.Context, feedURLs map[string]string, requirements []version.FileRequirement, ver string) map[string]*FileValidationResult {
	results := make(map[string]*FileValidationResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, req := range requirements {
		req := req
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			result := &FileValidationResult{
				File:     req.File + ".json",
				Required: req.Required,
			}

			url, exists := feedURLs[req.File]
			if !exists {
				result.Exists = false
				if req.Required {
					result.HasErrors = true
					result.ErrorsCount = 1
					result.Errors = []ValidationError{{
						Severity: SeverityError,
						Message:  fmt.Sprintf("Required file %s.json not found in autodiscovery", req.File),
					}}
				}
				mu.Lock()
				results[req.File] = result
				mu.Unlock()
				return
			}

			result.URL = url

			fetchResult := v.fetcher.Fetch(ctx, url)
			if fetchResult.Error != nil || !fetchResult.Exists {
				result.Exists = false
				if req.Required {
					result.HasErrors = true
					result.ErrorsCount = 1
					result.Errors = []ValidationError{{
						Severity: SeverityError,
						Message:  fmt.Sprintf("Required file %s.json could not be fetched: %v", req.File, fetchResult.Error),
					}}
				}
				mu.Lock()
				results[req.File] = result
				mu.Unlock()
				return
			}

			result.Exists = true
			result.RawData = fetchResult.Body

			dataToValidate := fetchResult.Body
			if v.coercer != nil {
				coerceResult, err := v.coercer.Coerce(fetchResult.Body, req.File)
				if err == nil {
					dataToValidate = coerceResult.Data
					result.CoercedData = coerceResult.Data
					result.CoercionCount = len(coerceResult.Log.Coercions)
				}
			}

			schemaErrors := v.validateFileStructure(dataToValidate, req.File, ver)
			if len(schemaErrors) > 0 {
				result.HasErrors = true
				result.Errors = schemaErrors
				result.ErrorsCount = len(schemaErrors)
			}

			mu.Lock()
			results[req.File] = result
			mu.Unlock()
		}()
	}

	wg.Wait()
	return results
}

// validateFileStructure checks a feed file's basic structure.
func (v *Validator) validateFileStructure(data []byte, feedType, ver string) []ValidationError {
	var errors []ValidationError

	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		errors = append(errors, ValidationError{
			Severity: SeverityError,
			Message:  fmt.Sprintf("Invalid JSON: %v", err),
		})
		return errors
	}

	if _, ok := jsonData["last_updated"]; !ok {
		errors = append(errors, ValidationError{
			Severity:     SeverityError,
			Message:      "last_updated is required",
			InstancePath: "/last_updated",
		})
	}

	if _, ok := jsonData["ttl"]; !ok {
		errors = append(errors, ValidationError{
			Severity:     SeverityWarning,
			Message:      "ttl is recommended",
			InstancePath: "/ttl",
		})
	}

	if _, ok := jsonData["data"]; !ok {
		errors = append(errors, ValidationError{
			Severity:     SeverityError,
			Message:      "data object is required",
			InstancePath: "/data",
		})
	}

	switch feedType {
	case "system_information":
		errors = append(errors, v.validateSystemInformation(jsonData, ver)...)
	case "station_information":
		errors = append(errors, v.validateStationInformation(jsonData, ver)...)
	case "station_status":
		errors = append(errors, v.validateStationStatus(jsonData, ver)...)
	case "vehicle_status", "free_bike_status":
		errors = append(errors, v.validateVehicleStatus(jsonData, ver)...)
	case "vehicle_types":
		errors = append(errors, v.validateVehicleTypes(jsonData, ver)...)
	}

	return errors
}

// validateSystemInformation checks system_information.json structure.
func (v *Validator) validateSystemInformation(data map[string]interface{}, ver string) []ValidationError {
	var errors []ValidationError

	dataObj, ok := data["data"].(map[string]interface{})
	if !ok {
		return errors
	}

	if _, ok := dataObj["system_id"]; !ok {
		errors = append(errors, ValidationError{
			Severity:     SeverityError,
			Message:      "system_id is required",
			InstancePath: "/data/system_id",
		})
	}

	if _, ok := dataObj["timezone"]; !ok {
		errors = append(errors, ValidationError{
			Severity:     SeverityError,
			Message:      "timezone is required",
			InstancePath: "/data/timezone",
		})
	}

	if _, ok := dataObj["name"]; !ok {
		errors = append(errors, ValidationError{
			Severity:     SeverityError,
			Message:      "name is required",
			InstancePath: "/data/name",
		})
	}

	return errors
}

// validateStationInformation checks station_information.json structure.
func (v *Validator) validateStationInformation(data map[string]interface{}, ver string) []ValidationError {
	var errors []ValidationError

	dataObj, ok := data["data"].(map[string]interface{})
	if !ok {
		return errors
	}

	stations, ok := dataObj["stations"].([]interface{})
	if !ok {
		errors = append(errors, ValidationError{
			Severity:     SeverityError,
			Message:      "stations array is required",
			InstancePath: "/data/stations",
		})
		return errors
	}

	for i, s := range stations {
		station, ok := s.(map[string]interface{})
		if !ok {
			continue
		}

		if _, ok := station["station_id"]; !ok {
			errors = append(errors, ValidationError{
				Severity:     SeverityError,
				Message:      "station_id is required",
				InstancePath: fmt.Sprintf("/data/stations/%d/station_id", i),
			})
		}

		if _, ok := station["lat"]; !ok {
			errors = append(errors, ValidationError{
				Severity:     SeverityError,
				Message:      "lat is required",
				InstancePath: fmt.Sprintf("/data/stations/%d/lat", i),
			})
		}

		if _, ok := station["lon"]; !ok {
			errors = append(errors, ValidationError{
				Severity:     SeverityError,
				Message:      "lon is required",
				InstancePath: fmt.Sprintf("/data/stations/%d/lon", i),
			})
		}
	}

	return errors
}

// validateStationStatus checks station_status.json structure.
func (v *Validator) validateStationStatus(data map[string]interface{}, ver string) []ValidationError {
	var errors []ValidationError

	dataObj, ok := data["data"].(map[string]interface{})
	if !ok {
		return errors
	}

	stations, ok := dataObj["stations"].([]interface{})
	if !ok {
		errors = append(errors, ValidationError{
			Severity:     SeverityError,
			Message:      "stations array is required",
			InstancePath: "/data/stations",
		})
		return errors
	}

	for i, s := range stations {
		station, ok := s.(map[string]interface{})
		if !ok {
			continue
		}

		if _, ok := station["station_id"]; !ok {
			errors = append(errors, ValidationError{
				Severity:     SeverityError,
				Message:      "station_id is required",
				InstancePath: fmt.Sprintf("/data/stations/%d/station_id", i),
			})
		}

		for _, field := range []string{"is_installed", "is_renting", "is_returning"} {
			if val, ok := station[field]; ok {
				if _, isBool := val.(bool); !isBool {
					errors = append(errors, ValidationError{
						Severity:     SeverityError,
						Message:      fmt.Sprintf("%s must be a boolean", field),
						InstancePath: fmt.Sprintf("/data/stations/%d/%s", i, field),
					})
				}
			}
		}
	}

	return errors
}

// validateVehicleStatus checks vehicle status feed structure.
func (v *Validator) validateVehicleStatus(data map[string]interface{}, ver string) []ValidationError {
	var errors []ValidationError

	dataObj, ok := data["data"].(map[string]interface{})
	if !ok {
		return errors
	}

	var vehicles []interface{}
	if v, ok := dataObj["vehicles"].([]interface{}); ok {
		vehicles = v
	} else if b, ok := dataObj["bikes"].([]interface{}); ok {
		vehicles = b
	} else {
		errors = append(errors, ValidationError{
			Severity:     SeverityError,
			Message:      "vehicles or bikes array is required",
			InstancePath: "/data/vehicles",
		})
		return errors
	}

	for i, v := range vehicles {
		vehicle, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		hasID := false
		if _, ok := vehicle["vehicle_id"]; ok {
			hasID = true
		}
		if _, ok := vehicle["bike_id"]; ok {
			hasID = true
		}
		if !hasID {
			errors = append(errors, ValidationError{
				Severity:     SeverityError,
				Message:      "vehicle_id or bike_id is required",
				InstancePath: fmt.Sprintf("/data/vehicles/%d", i),
			})
		}
	}

	return errors
}

// validateVehicleTypes checks vehicle_types.json structure.
func (v *Validator) validateVehicleTypes(data map[string]interface{}, ver string) []ValidationError {
	var errors []ValidationError

	dataObj, ok := data["data"].(map[string]interface{})
	if !ok {
		return errors
	}

	vehicleTypes, ok := dataObj["vehicle_types"].([]interface{})
	if !ok {
		errors = append(errors, ValidationError{
			Severity:     SeverityError,
			Message:      "vehicle_types array is required",
			InstancePath: "/data/vehicle_types",
		})
		return errors
	}

	for i, vt := range vehicleTypes {
		vehicleType, ok := vt.(map[string]interface{})
		if !ok {
			continue
		}

		if _, ok := vehicleType["vehicle_type_id"]; !ok {
			errors = append(errors, ValidationError{
				Severity:     SeverityError,
				Message:      "vehicle_type_id is required",
				InstancePath: fmt.Sprintf("/data/vehicle_types/%d/vehicle_type_id", i),
			})
		}

		if _, ok := vehicleType["form_factor"]; !ok {
			errors = append(errors, ValidationError{
				Severity:     SeverityError,
				Message:      "form_factor is required",
				InstancePath: fmt.Sprintf("/data/vehicle_types/%d/form_factor", i),
			})
		}

		if _, ok := vehicleType["propulsion_type"]; !ok {
			errors = append(errors, ValidationError{
				Severity:     SeverityError,
				Message:      "propulsion_type is required",
				InstancePath: fmt.Sprintf("/data/vehicle_types/%d/propulsion_type", i),
			})
		}

		if pt, ok := vehicleType["propulsion_type"].(string); ok {
			if isMotorized(pt) {
				if _, ok := vehicleType["max_range_meters"]; !ok {
					errors = append(errors, ValidationError{
						Severity:     SeverityWarning,
						Message:      "max_range_meters is required for motorized vehicles",
						InstancePath: fmt.Sprintf("/data/vehicle_types/%d/max_range_meters", i),
					})
				}
			}
		}
	}

	return errors
}

// crossValidate performs referential checks across files.
func (v *Validator) crossValidate(results map[string]*FileValidationResult, ver string) {
	vehicleTypes := v.extractVehicleTypes(results)
	pricingPlans := v.extractPricingPlans(results)
	stationIDs := v.extractStationIDs(results)

	v.validateVehicleTypeReferences(results, vehicleTypes, ver)

	v.validatePricingPlanReferences(results, pricingPlans, ver)

	v.validateStationIDReferences(results, stationIDs, ver)

	v.checkConditionalVehicleTypes(results, ver)

	v.checkConditionalPricingPlans(results, ver)
}

// extractVehicleTypes reads vehicle types from vehicle_types.json.
func (v *Validator) extractVehicleTypes(results map[string]*FileValidationResult) map[string]gbfs.VehicleType {
	types := make(map[string]gbfs.VehicleType)

	result, ok := results["vehicle_types"]
	if !ok || !result.Exists || result.RawData == nil {
		return types
	}

	var vt gbfs.VehicleTypes
	if err := json.Unmarshal(result.RawData, &vt); err != nil {
		return types
	}

	for _, t := range vt.Data.VehicleTypes {
		types[t.VehicleTypeID] = t
	}

	return types
}

// extractPricingPlans reads pricing plans from system_pricing_plans.json.
func (v *Validator) extractPricingPlans(results map[string]*FileValidationResult) map[string]gbfs.PricingPlan {
	plans := make(map[string]gbfs.PricingPlan)

	result, ok := results["system_pricing_plans"]
	if !ok || !result.Exists || result.RawData == nil {
		return plans
	}

	var pp gbfs.SystemPricingPlans
	if err := json.Unmarshal(result.RawData, &pp); err != nil {
		return plans
	}

	for _, p := range pp.Data.Plans {
		plans[p.PlanID] = p
	}

	return plans
}

// extractStationIDs reads station IDs from station_information.json.
func (v *Validator) extractStationIDs(results map[string]*FileValidationResult) map[string]bool {
	ids := make(map[string]bool)

	result, ok := results["station_information"]
	if !ok || !result.Exists || result.RawData == nil {
		return ids
	}

	var si gbfs.StationInformation
	if err := json.Unmarshal(result.RawData, &si); err != nil {
		return ids
	}

	for _, s := range si.Data.Stations {
		ids[s.StationID] = true
	}

	return ids
}

// validateVehicleTypeReferences verifies vehicle_type_id references.
func (v *Validator) validateVehicleTypeReferences(results map[string]*FileValidationResult, vehicleTypes map[string]gbfs.VehicleType, ver string) {
	fileName := version.GetVehicleStatusFileName(ver)
	result, ok := results[fileName]
	if !ok || !result.Exists || result.RawData == nil {
		return
	}

	if len(vehicleTypes) == 0 {
		return
	}

	var vs gbfs.VehicleStatus
	if err := json.Unmarshal(result.RawData, &vs); err != nil {
		return
	}

	vehicles := vs.Data.GetVehicles()
	for i, vehicle := range vehicles {
		if vehicle.VehicleTypeID != "" {
			if _, exists := vehicleTypes[vehicle.VehicleTypeID]; !exists {
				result.Errors = append(result.Errors, ValidationError{
					Severity:     SeverityError,
					InstancePath: fmt.Sprintf("/data/vehicles/%d/vehicle_type_id", i),
					Message:      fmt.Sprintf("vehicle_type_id '%s' not found in vehicle_types.json", vehicle.VehicleTypeID),
				})
				result.HasErrors = true
				result.ErrorsCount++
			}

			vt := vehicleTypes[vehicle.VehicleTypeID]
			if isMotorized(vt.PropulsionType) && vehicle.CurrentRangeMeters == 0 {
				result.Errors = append(result.Errors, ValidationError{
					Severity:     SeverityWarning,
					InstancePath: fmt.Sprintf("/data/vehicles/%d", i),
					Message:      "current_range_meters is recommended for motorized vehicles",
				})
			}
		}
	}
}

// validatePricingPlanReferences verifies pricing_plan_id references.
func (v *Validator) validatePricingPlanReferences(results map[string]*FileValidationResult, pricingPlans map[string]gbfs.PricingPlan, ver string) {
	if len(pricingPlans) == 0 {
		return
	}

	vtResult, ok := results["vehicle_types"]
	if ok && vtResult.Exists && vtResult.RawData != nil {
		var vt gbfs.VehicleTypes
		if err := json.Unmarshal(vtResult.RawData, &vt); err == nil {
			for i, t := range vt.Data.VehicleTypes {
				if t.DefaultPricingPlanID != "" {
					if _, exists := pricingPlans[t.DefaultPricingPlanID]; !exists {
						vtResult.Errors = append(vtResult.Errors, ValidationError{
							Severity:     SeverityError,
							InstancePath: fmt.Sprintf("/data/vehicle_types/%d/default_pricing_plan_id", i),
							Message:      fmt.Sprintf("default_pricing_plan_id '%s' not found in system_pricing_plans.json", t.DefaultPricingPlanID),
						})
						vtResult.HasErrors = true
						vtResult.ErrorsCount++
					}
				}
			}
		}
	}
}

// validateStationIDReferences verifies station_id references.
func (v *Validator) validateStationIDReferences(results map[string]*FileValidationResult, stationIDs map[string]bool, ver string) {
	if len(stationIDs) == 0 {
		return
	}

	ssResult, ok := results["station_status"]
	if ok && ssResult.Exists && ssResult.RawData != nil {
		var ss gbfs.StationStatus
		if err := json.Unmarshal(ssResult.RawData, &ss); err == nil {
			for i, s := range ss.Data.Stations {
				if !stationIDs[s.StationID] {
					ssResult.Errors = append(ssResult.Errors, ValidationError{
						Severity:     SeverityError,
						InstancePath: fmt.Sprintf("/data/stations/%d/station_id", i),
						Message:      fmt.Sprintf("station_id '%s' not found in station_information.json", s.StationID),
					})
					ssResult.HasErrors = true
					ssResult.ErrorsCount++
				}
			}
		}
	}
}

// checkConditionalVehicleTypes enforces vehicle_types.json requirement.
func (v *Validator) checkConditionalVehicleTypes(results map[string]*FileValidationResult, ver string) {
	fileName := version.GetVehicleStatusFileName(ver)
	vsResult, ok := results[fileName]
	if !ok || !vsResult.Exists || vsResult.RawData == nil {
		return
	}

	var vs gbfs.VehicleStatus
	if err := json.Unmarshal(vsResult.RawData, &vs); err != nil {
		return
	}

	hasVehicleTypeID := false
	for _, vehicle := range vs.Data.GetVehicles() {
		if vehicle.VehicleTypeID != "" {
			hasVehicleTypeID = true
			break
		}
	}

	if hasVehicleTypeID {
		vtResult, ok := results["vehicle_types"]
		if !ok || !vtResult.Exists {
			if vtResult == nil {
				results["vehicle_types"] = &FileValidationResult{
					File:     "vehicle_types.json",
					Required: true,
					Exists:   false,
				}
				vtResult = results["vehicle_types"]
			}
			vtResult.Required = true
			vtResult.HasErrors = true
			vtResult.ErrorsCount++
			vtResult.Errors = append(vtResult.Errors, ValidationError{
				Severity: SeverityError,
				Message:  "vehicle_types.json is required when vehicle_type_id is used in " + fileName + ".json",
			})
		}
	}
}

// checkConditionalPricingPlans enforces system_pricing_plans.json requirement.
func (v *Validator) checkConditionalPricingPlans(results map[string]*FileValidationResult, ver string) {
	fileName := version.GetVehicleStatusFileName(ver)
	vsResult, ok := results[fileName]
	if !ok || !vsResult.Exists || vsResult.RawData == nil {
		return
	}

	var vs gbfs.VehicleStatus
	if err := json.Unmarshal(vsResult.RawData, &vs); err != nil {
		return
	}

	hasPricingPlanID := false
	for _, vehicle := range vs.Data.GetVehicles() {
		if vehicle.PricingPlanID != "" {
			hasPricingPlanID = true
			break
		}
	}

	if hasPricingPlanID {
		ppResult, ok := results["system_pricing_plans"]
		if !ok || !ppResult.Exists {
			if ppResult == nil {
				results["system_pricing_plans"] = &FileValidationResult{
					File:     "system_pricing_plans.json",
					Required: true,
					Exists:   false,
				}
				ppResult = results["system_pricing_plans"]
			}
			ppResult.Required = true
			ppResult.HasErrors = true
			ppResult.ErrorsCount++
			ppResult.Errors = append(ppResult.Errors, ValidationError{
				Severity: SeverityError,
				Message:  "system_pricing_plans.json is required when pricing_plan_id is used in " + fileName + ".json",
			})
		}
	}
}

// isMotorized reports whether a propulsion type is motorized.
func isMotorized(propulsionType string) bool {
	switch propulsionType {
	case "electric", "electric_assist", "combustion", "combustion_diesel", "hybrid", "plug_in_hybrid", "hydrogen_fuel_cell":
		return true
	default:
		return false
	}
}
