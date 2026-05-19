package ruletest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pauvalls/arx/internal/ruletest"
	"gopkg.in/yaml.v3"
)

// testFile represents the YAML structure of a single test file
type testFile struct {
	Tests []testCaseYAML `yaml:"tests"`
}

// testCaseYAML is the intermediate YAML unmarshal target
type testCaseYAML struct {
	Name    string        `yaml:"name"`
	Fixture string        `yaml:"fixture,omitempty"`
	RuleID  string        `yaml:"rule,omitempty"`
	Expect  expectationYAML `yaml:"expect"`
}

// expectationYAML is the intermediate YAML unmarshal target for expectations
type expectationYAML struct {
	Violations *int                   `yaml:"violations,omitempty"`
	Files      []string               `yaml:"files,omitempty"`
	Layers     []layerExpectationYAML `yaml:"layers,omitempty"`
	Patterns   []string               `yaml:"patterns,omitempty"`
}

// layerExpectationYAML is the intermediate YAML unmarshal target for layer expectations
type layerExpectationYAML struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
}

// Parser parses YAML test definition files into domain TestSuite types
type Parser struct{}

// NewParser creates a new YAML test parser
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses a path (file or directory) and returns test suites.
// If path is a directory, all *.yaml and *.yml files are parsed.
func (p *Parser) Parse(path string) ([]ruletest.TestSuite, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("accessing path %s: %w", path, err)
	}

	if info.IsDir() {
		return p.ParseDir(path)
	}

	return p.ParseFile(path)
}

// ParseFile parses a single YAML file and returns test suites.
func (p *Parser) ParseFile(path string) ([]ruletest.TestSuite, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var tf testFile
	if err := yaml.Unmarshal(data, &tf); err != nil {
		return nil, fmt.Errorf("parsing YAML in %s: %w", path, err)
	}

	if len(tf.Tests) == 0 {
		return nil, fmt.Errorf("%s: no test cases defined", path)
	}

	seen := make(map[string]bool)
	cases := make([]ruletest.TestCase, 0, len(tf.Tests))
	for _, tcYAML := range tf.Tests {
		if tcYAML.Name == "" {
			return nil, fmt.Errorf("%s: test case name is required", path)
		}
		if seen[tcYAML.Name] {
			return nil, fmt.Errorf("%s: duplicate test name %q", path, tcYAML.Name)
		}
		seen[tcYAML.Name] = true

		// Convert expectation
		exp := ruletest.Expectation{
			Violations: tcYAML.Expect.Violations,
			Files:      tcYAML.Expect.Files,
			Patterns:   tcYAML.Expect.Patterns,
		}
		for _, ly := range tcYAML.Expect.Layers {
			exp.Layers = append(exp.Layers, ruletest.LayerExpectation{
				Source: ly.Source,
				Target: ly.Target,
			})
		}

		tc := ruletest.TestCase{
			Name:    tcYAML.Name,
			Fixture: tcYAML.Fixture,
			RuleID:  tcYAML.RuleID,
			Expect:  exp,
		}

		if !tc.Expect.HasExpectations() {
			return nil, fmt.Errorf("%s: test case %q has no expectations set (violations, files, layers, or patterns required)", path, tc.Name)
		}

		cases = append(cases, tc)
	}

	// Create a suite named after the file (without extension)
	suiteName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	suite := ruletest.TestSuite{
		Name:  suiteName,
		Tests: cases,
	}

	if err := suite.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	return []ruletest.TestSuite{suite}, nil
}

// ParseDir parses all YAML files in a directory and returns test suites.
// Only files with .yaml or .yml extensions are parsed.
func (p *Parser) ParseDir(dir string) ([]ruletest.TestSuite, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	var suites []ruletest.TestSuite
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		fileSuites, err := p.ParseFile(path)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		suites = append(suites, fileSuites...)
	}

	if len(suites) == 0 {
		return suites, nil
	}

	return suites, nil
}
