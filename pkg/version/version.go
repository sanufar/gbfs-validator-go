// Package version defines GBFS version-specific requirements.
package version

// FileRequirement declares a feed file and its requirement status.
type FileRequirement struct {
	File     string
	Required bool
}

// Options selects docked/free-floating requirements.
type Options struct {
	Docked       bool
	Freefloating bool
}

// Config describes requirements for a GBFS version.
type Config struct {
	Version      string
	GBFSRequired bool // Whether gbfs.json is required (became required in v2.0)
	Files        func(opts Options) []FileRequirement
}

// Configs maps GBFS versions to their requirements.
var Configs = map[string]Config{
	"1.0": {
		Version:      "1.0",
		GBFSRequired: false,
		Files: func(opts Options) []FileRequirement {
			return []FileRequirement{
				{File: "system_information", Required: true},
				{File: "station_information", Required: opts.Docked},
				{File: "station_status", Required: opts.Docked},
				{File: "free_bike_status", Required: opts.Freefloating},
				{File: "system_hours", Required: false},
				{File: "system_calendar", Required: false},
				{File: "system_regions", Required: false},
				{File: "system_pricing_plans", Required: false},
				{File: "system_alerts", Required: false},
			}
		},
	},
	"1.1": {
		Version:      "1.1",
		GBFSRequired: false,
		Files: func(opts Options) []FileRequirement {
			return []FileRequirement{
				{File: "gbfs_versions", Required: false},
				{File: "system_information", Required: true},
				{File: "station_information", Required: opts.Docked},
				{File: "station_status", Required: opts.Docked},
				{File: "free_bike_status", Required: opts.Freefloating},
				{File: "system_hours", Required: false},
				{File: "system_calendar", Required: false},
				{File: "system_regions", Required: false},
				{File: "system_pricing_plans", Required: false},
				{File: "system_alerts", Required: false},
			}
		},
	},
	"2.0": {
		Version:      "2.0",
		GBFSRequired: true,
		Files: func(opts Options) []FileRequirement {
			return []FileRequirement{
				{File: "gbfs_versions", Required: false},
				{File: "system_information", Required: true},
				{File: "station_information", Required: opts.Docked},
				{File: "station_status", Required: opts.Docked},
				{File: "free_bike_status", Required: opts.Freefloating},
				{File: "system_hours", Required: false},
				{File: "system_calendar", Required: false},
				{File: "system_regions", Required: false},
				{File: "system_pricing_plans", Required: false},
				{File: "system_alerts", Required: false},
			}
		},
	},
	"2.1": {
		Version:      "2.1",
		GBFSRequired: true,
		Files: func(opts Options) []FileRequirement {
			return []FileRequirement{
				{File: "gbfs_versions", Required: false},
				{File: "system_information", Required: true},
				{File: "vehicle_types", Required: false}, // Conditionally required
				{File: "station_information", Required: opts.Docked},
				{File: "station_status", Required: opts.Docked},
				{File: "free_bike_status", Required: opts.Freefloating},
				{File: "system_hours", Required: false},
				{File: "system_calendar", Required: false},
				{File: "system_regions", Required: false},
				{File: "system_pricing_plans", Required: false},
				{File: "system_alerts", Required: false},
				{File: "geofencing_zones", Required: false},
			}
		},
	},
	"2.2": {
		Version:      "2.2",
		GBFSRequired: true,
		Files: func(opts Options) []FileRequirement {
			return []FileRequirement{
				{File: "gbfs_versions", Required: false},
				{File: "system_information", Required: true},
				{File: "vehicle_types", Required: false}, // Conditionally required
				{File: "station_information", Required: opts.Docked},
				{File: "station_status", Required: opts.Docked},
				{File: "free_bike_status", Required: opts.Freefloating},
				{File: "system_hours", Required: false},
				{File: "system_calendar", Required: false},
				{File: "system_regions", Required: false},
				{File: "system_pricing_plans", Required: false},
				{File: "system_alerts", Required: false},
				{File: "geofencing_zones", Required: false},
			}
		},
	},
	"2.3": {
		Version:      "2.3",
		GBFSRequired: true,
		Files: func(opts Options) []FileRequirement {
			return []FileRequirement{
				{File: "gbfs_versions", Required: false},
				{File: "system_information", Required: true},
				{File: "vehicle_types", Required: false}, // Conditionally required
				{File: "station_information", Required: opts.Docked},
				{File: "station_status", Required: opts.Docked},
				{File: "free_bike_status", Required: opts.Freefloating},
				{File: "system_hours", Required: false},
				{File: "system_calendar", Required: false},
				{File: "system_regions", Required: false},
				{File: "system_pricing_plans", Required: false},
				{File: "system_alerts", Required: false},
				{File: "geofencing_zones", Required: false},
			}
		},
	},
	"3.0": {
		Version:      "3.0",
		GBFSRequired: true,
		Files: func(opts Options) []FileRequirement {
			return []FileRequirement{
				{File: "manifest", Required: false},
				{File: "gbfs_versions", Required: false},
				{File: "system_information", Required: true},
				{File: "vehicle_types", Required: false},
				{File: "station_information", Required: opts.Docked},
				{File: "station_status", Required: opts.Docked},
				{File: "vehicle_status", Required: opts.Freefloating}, // Renamed from free_bike_status
				{File: "system_regions", Required: false},
				{File: "system_pricing_plans", Required: false},
				{File: "system_alerts", Required: false},
				{File: "geofencing_zones", Required: false},
			}
		},
	},
	"3.1-RC2": {
		Version:      "3.1-RC2",
		GBFSRequired: true,
		Files: func(opts Options) []FileRequirement {
			return []FileRequirement{
				{File: "manifest", Required: false},
				{File: "gbfs_versions", Required: false},
				{File: "system_information", Required: true},
				{File: "vehicle_types", Required: false},
				{File: "station_information", Required: opts.Docked},
				{File: "station_status", Required: opts.Docked},
				{File: "vehicle_status", Required: opts.Freefloating},
				{File: "vehicle_availability", Required: false},
				{File: "system_regions", Required: false},
				{File: "system_pricing_plans", Required: false},
				{File: "system_alerts", Required: false},
				{File: "geofencing_zones", Required: false},
			}
		},
	},
}

// GetConfig returns a version configuration.
func GetConfig(version string) (Config, bool) {
	cfg, ok := Configs[version]
	return cfg, ok
}

// GetFileRequirements returns file requirements for a version.
func GetFileRequirements(version string, opts Options) []FileRequirement {
	cfg, ok := Configs[version]
	if !ok {
		cfg = Configs["3.0"]
	}
	return cfg.Files(opts)
}

// IsGBFSRequired reports whether gbfs.json is required.
func IsGBFSRequired(version string) bool {
	cfg, ok := Configs[version]
	if !ok {
		return true
	}
	return cfg.GBFSRequired
}

// SupportedVersions lists supported GBFS versions.
func SupportedVersions() []string {
	return []string{"1.0", "1.1", "2.0", "2.1", "2.2", "2.3", "3.0", "3.1-RC2"}
}

// IsV3OrLater reports whether a version uses v3+ file layouts.
func IsV3OrLater(version string) bool {
	switch version {
	case "3.0", "3.1-RC2", "3.1":
		return true
	default:
		return false
	}
}

// GetVehicleStatusFileName returns the vehicle status filename for a version.
func GetVehicleStatusFileName(version string) string {
	if IsV3OrLater(version) {
		return "vehicle_status"
	}
	return "free_bike_status"
}
