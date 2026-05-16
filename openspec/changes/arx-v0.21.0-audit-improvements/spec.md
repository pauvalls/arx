# Delta Spec: arx-v0.21.0-audit-improvements

---

## Domain: html-audit-report

### Requirement: Coupling Matrix in HTML Report

The system MUST render a coupling matrix table in the HTML audit report when coupling data is available.

#### Scenario: Coupling matrix rendered with data

- GIVEN an audit with coupling data
- WHEN `arx audit --format html` is executed
- THEN the HTML report includes a table with columns: From, To, Count, Percentage
- AND each row represents a directed dependency between layers

#### Scenario: Empty coupling matrix

- GIVEN an audit with no coupling data
- WHEN `arx audit --format html` is executed
- THEN the coupling matrix section displays "(no data)"
- AND the report validates as HTML5

### Requirement: Debt Score in HTML Report

The system MUST display the debt score with severity breakdown in the HTML audit report.

#### Scenario: Debt score with breakdown

- GIVEN an audit report with debt score computed
- WHEN `arx audit --format html` is executed
- THEN the HTML report shows the total debt score
- AND shows a breakdown by severity (critical, high, medium, low)

### Requirement: Trend Section in HTML Report

The system MUST include a trend section showing violation and debt deltas.

#### Scenario: Trend section with status

- GIVEN an audit report with trend data
- WHEN `arx audit --format html` is executed
- THEN the trend section displays status: improved, degraded, or unchanged
- AND shows numeric deltas for new and resolved violations

---

## Domain: json-check-improvements

### Requirement: Coupling Matrix in JSON Output

The system MUST include coupling matrix data in JSON check output when available.

#### Scenario: JSON includes coupling matrix

- GIVEN `arx check --format json` with coupling data available
- WHEN the command executes
- THEN the JSON output includes a "coupling_matrix" object
- AND the object contains from/to entries with dependency counts

### Requirement: Detector Metadata in JSON Output

The system MUST include detector metadata in JSON check output.

#### Scenario: JSON includes detector array

- GIVEN `arx check --format json`
- WHEN the command executes
- THEN the JSON output includes a "detectors" array
- AND each detector has fields: name, applicable (boolean), dep_count (integer)

#### Scenario: Backward compatibility

- GIVEN existing consumers parsing JSON check output
- WHEN the new fields are present
- THEN all existing violation fields remain unchanged
- AND consumers ignoring unknown fields continue to work

---

## Domain: quality-pass

### Requirement: Static Analysis Clean

The system MUST pass `go vet ./...` with zero warnings across all packages.

#### Scenario: go vet clean

- GIVEN the full codebase
- WHEN `go vet ./...` is run
- THEN it produces no warnings

### Requirement: Fuzz Tests Pass

The system MUST run all fuzz tests for 5 seconds with zero crashes.

#### Scenario: Fuzz tests pass

- GIVEN all fuzz targets in the codebase
- WHEN each target runs with `-fuzztime=5s`
- THEN no crashes occur

### Requirement: No Deprecated API Usage

The system MUST not use deprecated Go standard library APIs.

#### Scenario: No deprecated API usage

- GIVEN the full codebase
- WHEN audited for deprecated API usage
- THEN `strings.Title` is not used
- AND `filepath.HasPrefix` is not used
