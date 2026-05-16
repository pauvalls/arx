package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// SARIFReporter implements ports.Reporter for SARIF 2.1.0 output
type SARIFReporter struct {
	tool    string
	version string
}

// NewSARIFReporter creates a new SARIF reporter
func NewSARIFReporter() *SARIFReporter {
	return &SARIFReporter{
		tool:    "arx",
		version: "2.1.0",
	}
}

// SARIFLog represents the complete SARIF log
type SARIFLog struct {
	Schema  string `json:"$schema"`
	Version string `json:"version"`
	Runs    []Run  `json:"runs"`
}

// Run represents a single run of the tool
type Run struct {
	Tool    Tool     `json:"tool"`
	Results []Result `json:"results"`
}

// Tool represents the tool information
type Tool struct {
	Driver Driver `json:"driver"`
}

// Driver represents the tool driver information
type Driver struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	InformationURI string `json:"informationUri,omitempty"`
}

// Result represents a single result/violation
type Result struct {
	RuleID    string   `json:"ruleId"`
	Level     string   `json:"level"`
	Message   Message  `json:"message"`
	Locations []Location `json:"locations"`
	Properties Properties `json:"properties,omitempty"`
}

// Message represents the result message
type Message struct {
	Text string `json:"text"`
}

// Location represents the location of the violation
type Location struct {
	PhysicalLocation PhysicalLocation `json:"physicalLocation"`
}

// PhysicalLocation represents the physical location
type PhysicalLocation struct {
	ArtifactLocation ArtifactLocation `json:"artifactLocation"`
	Region           Region           `json:"region"`
}

// ArtifactLocation represents the artifact location
type ArtifactLocation struct {
	URI string `json:"uri"`
}

// Region represents the region in the file
type Region struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn,omitempty"`
}

// extractFilePath extracts a clean file path from a violation File field.
// File may contain "layer (path/file.go:line)" format (circular deps).
func extractFilePath(file string) string {
	if idx := strings.Index(file, "("); idx != -1 {
		// Format: "layer (path/file.go:line)"
		inner := file[idx+1:]
		if endIdx := strings.LastIndex(inner, ")"); endIdx != -1 {
			inner = inner[:endIdx]
		}
		// inner is now "path/file.go:line"
		if colonIdx := strings.LastIndex(inner, ":"); colonIdx != -1 {
			return inner[:colonIdx]
		}
		return inner
	}
	return file
}
type Properties struct {
	SourceLayer  string `json:"source_layer,omitempty"`
	TargetLayer  string `json:"target_layer,omitempty"`
	Import       string `json:"import,omitempty"`
	Explanation  string `json:"explanation,omitempty"`
}

// Report implements ports.Reporter interface
func (r *SARIFReporter) Report(violations []domain.Violation, format ports.OutputFormat) error {
	log := r.buildSARIFLog(violations)

	data, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal SARIF: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

// buildSARIFLog constructs the SARIF log from violations
func (r *SARIFReporter) buildSARIFLog(violations []domain.Violation) SARIFLog {
	results := make([]Result, 0, len(violations))

	for _, v := range violations {
		level := "error"
		if strings.ToLower(string(v.Severity)) == "warning" {
			level = "warning"
		}

		result := Result{
			RuleID:  v.RuleID,
			Level:   level,
			Message: Message{Text: v.Message},
			Locations: []Location{
				{
					PhysicalLocation: PhysicalLocation{
						ArtifactLocation: ArtifactLocation{
							URI: extractFilePath(v.File),
						},
						Region: Region{
							StartLine: v.Line,
						},
					},
				},
			},
			Properties: Properties{
				SourceLayer: v.SourceLayer,
				TargetLayer: v.TargetLayer,
				Import:      v.Import,
			},
		}
		results = append(results, result)
	}

	return SARIFLog{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: r.version,
		Runs: []Run{
			{
				Tool: Tool{
					Driver: Driver{
						Name:      r.tool,
						Version:   "0.2.0",
					},
				},
				Results: results,
			},
		},
	}
}

// ViolationCache stores violations for later lookup
type ViolationCache struct {
	Violations  []CachedViolation `json:"violations"`
	Timestamp   time.Time         `json:"timestamp"`
	ProjectRoot string            `json:"project_root"`
}

// CachedViolation is a violation stored in cache
type CachedViolation struct {
	ID           string `json:"id"`
	RuleID       string `json:"rule_id"`
	Severity     string `json:"severity"`
	File         string `json:"file"`
	Line         int    `json:"line"`
	SourceLayer  string `json:"source_layer"`
	TargetLayer  string `json:"target_layer"`
	Import       string `json:"import"`
	Message      string `json:"message"`
	Explanation  string `json:"explanation"`
}

// CacheViolations saves violations to cache file
func CacheViolations(violations []domain.Violation, projectRoot string) error {
	cacheDir := ".arx-cache"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	cached := make([]CachedViolation, 0, len(violations))
	for i, v := range violations {
		cv := CachedViolation{
			ID:          fmt.Sprintf("D-%02d", i+1),
			RuleID:      v.RuleID,
			Severity:    string(v.Severity),
			File:        v.File,
			Line:        v.Line,
			SourceLayer: v.SourceLayer,
			TargetLayer: v.TargetLayer,
			Import:      v.Import,
			Message:     v.Message,
		}
		cached = append(cached, cv)
	}

	cache := ViolationCache{
		Violations:  cached,
		Timestamp:   time.Now(),
		ProjectRoot: projectRoot,
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	cacheFile := fmt.Sprintf("%s/violations.json", cacheDir)
	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// LoadViolations loads violations from cache
func LoadViolations() (*ViolationCache, error) {
	cacheFile := ".arx-cache/violations.json"

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no cached violations found - run 'arx check' first")
		}
		return nil, fmt.Errorf("failed to read cache: %w", err)
	}

	var cache ViolationCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to parse cache: %w", err)
	}

	// Check if cache is expired (24 hours TTL)
	if time.Since(cache.Timestamp) > 24*time.Hour {
		return nil, fmt.Errorf("cache expired - run 'arx check' to refresh")
	}

	return &cache, nil
}

// GetViolationByID finds a violation by ID from cache
func GetViolationByID(cache *ViolationCache, id string) (*CachedViolation, error) {
	for _, v := range cache.Violations {
		if v.ID == id {
			return &v, nil
		}
	}

	return nil, fmt.Errorf("violation %q not found in cache", id)
}
