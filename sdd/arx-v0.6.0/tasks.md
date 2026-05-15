# Arx v0.6.0 — Implementation Tasks

**Spec Reference**: `openspec/changes/arx-v0.6.0-java-audit/spec.md`  
**Version**: 0.6.0  
**Features**: Java Detector + Arx Audit (health reports, trends, coupling matrix, debt estimation)

---

## Task Breakdown

### Phase 1: Java Detector (Foundation)

#### T1 — Create Java detector skeleton
**Title**: Create Java detector with Detect() for Maven/Gradle  
**Description**: Implement `internal/infrastructure/detector/java/detector.go` following the pattern of existing detectors (Go, TypeScript). Detect() checks for `pom.xml` (Maven) or `build.gradle` (Gradle) in project root.  
**Files**:
- `internal/infrastructure/detector/java/detector.go` (new, ~150 lines)
- `internal/infrastructure/detector/java/detector_test.go` (new, ~80 lines)  
**Dependencies**: None  
**Effort**: S (<2h)  
**Acceptance Criteria**:
- ✅ `Detect()` returns `true` for projects with `pom.xml`
- ✅ `Detect()` returns `true` for projects with `build.gradle`
- ✅ `Detect()` returns `false` for projects without either file
- ✅ Test coverage ≥80%

---

#### T2 — Implement Java import extraction (regex-based)
**Title**: Implement ExtractImports() with regex patterns for Java imports  
**Description**: Add regex patterns to extract `import`, `import static`, package declarations, and wildcard imports from `.java` files. Skip test files (`*Test.java`) and generated directories (`target/`, `build/`).  
**Files**:
- `internal/infrastructure/detector/java/detector.go` (edit, +120 lines)
- `internal/infrastructure/detector/java/detector_test.go` (edit, +100 lines)  
**Dependencies**: T1  
**Effort**: M (2-4h)  
**Acceptance Criteria**:
- ✅ Extracts standard imports: `import java.util.List;`
- ✅ Extracts static imports: `import static java.lang.Math.PI;`
- ✅ Extracts wildcard imports: `import com.example.domain.*;`
- ✅ Extracts package declarations for layer resolution
- ✅ Skips `*Test.java` files
- ✅ Skips `target/` and `build/` directories
- ✅ ≥90% precision on test fixture with 100 known imports

---

#### T3 — Implement Java layer resolution
**Title**: Implement layer resolution for Java package paths  
**Description**: Map Java package names (e.g., `com.example.domain.order`) to layers defined in `arx.yaml` by matching against layer path patterns. Handle external dependencies (e.g., `org.springframework`) by returning empty layer.  
**Files**:
- `internal/infrastructure/detector/java/detector.go` (edit, +60 lines)  
**Dependencies**: T2  
**Effort**: S (<2h)  
**Acceptance Criteria**:
- ✅ Resolves `com.example.domain.order.Order` to layer `domain` (given matching path pattern)
- ✅ Returns empty layer for `org.springframework.boot.SpringApplication`
- ✅ Handles wildcard imports via prefix matching

---

#### T4 — Register Java detector in registry
**Title**: Register JavaDetector in detector registry  
**Description**: Add `javadetector.New()` to `GetDetectors()` in `registry.go` so Java detection runs alongside Go/TypeScript/Python.  
**Files**:
- `internal/infrastructure/detector/registry.go` (edit, +2 lines)  
**Dependencies**: T3  
**Effort**: S (<30min)  
**Acceptance Criteria**:
- ✅ `arx check` on Java project detects and processes `.java` files
- ✅ Multi-language projects (e.g., Go + Java) run all applicable detectors

---

### Phase 2: Domain Models for Audit

#### T5 — Define AuditReport domain model
**Title**: Define AuditReport struct with metrics fields  
**Description**: Create domain model for audit results including violation summary, layer health scores, coupling matrix, and debt score.  
**Files**:
- `internal/domain/audit_result.go` (new, ~180 lines)
- `internal/domain/audit_result_test.go` (new, ~250 lines)  
**Dependencies**: None (parallel with T1-T4)  
**Effort**: M (2-4h)  
**Acceptance Criteria**:
- ✅ `AuditReport` struct with fields: `Timestamp`, `ProjectRoot`, `ConfigHash`, `Violations`, `CouplingMatrix`, `DebtScore`, `TrendReport`
- ✅ `CouplingMatrix` struct as map: `map[string]map[string]int` (from→to→count) with Add(), Get(), Count() methods
- ✅ `DebtScore` struct with fields: `Total`, `BySeverity` (error/warning/info), `Trend` (up/down/stable), `TrendDelta`
- ✅ `TrendReport` struct with fields: `ViolationDelta`, `DebtDelta`, `Status` (improved/degraded/unchanged), `Summary`
- ✅ JSON marshaling support for all structs
- ✅ All tests passing (15/15)

---

#### T6 — Define AuditHistory storage interface
**Title**: Define ports.HistoryStorage interface for persistence  
**Description**: Create interface for saving/loading audit snapshots with retention policy (max 10 audits).  
**Files**:
- `internal/ports/history.go` (new, ~40 lines)
- `internal/infrastructure/history/storage.go` (new, ~180 lines)
- `internal/infrastructure/history/storage_test.go` (new, ~400 lines)  
**Dependencies**: T5  
**Effort**: S (<2h)  
**Acceptance Criteria**:
- ✅ `Save(ctx, report)` method
- ✅ `Load(ctx, date)` method
- ✅ `LoadLatest(ctx)` method
- ✅ `List(ctx)` method
- ✅ `DeleteOld(ctx, maxAudits)` method
- ✅ Retention policy: max 10 audits
- ✅ JSON serialization of AuditReport
- ✅ Storage in `.arx-history/audit-YYYY-MM-DD.json`
- ✅ All 15 tests passing

---

### Phase 3: Audit Service Implementation

#### T7 — Implement audit history storage (JSON-based)
**Title**: Implement JSON-based audit history persistence  
**Description**: Create `internal/infrastructure/history/json_history.go` to save audits to `.arx-history/audit-YYYY-MM-DD.json` with symlink for `last-audit.json`.  
**Files**:
- `internal/infrastructure/history/json_history.go` (new, ~180 lines)
- `internal/infrastructure/history/json_history_test.go` (new, ~100 lines)  
**Dependencies**: T6  
**Effort**: M (2-4h)  
**Acceptance Criteria**:
- ✅ Saves to `.arx-history/audit-2026-05-14.json` format
- ✅ Creates/updates `.arx-history/last-audit.json` symlink
- ✅ Retention policy: deletes oldest when >10 audits
- ✅ Handles corrupted files gracefully (warning + skip)
- ✅ Handles missing history gracefully

---

#### [x] T8 — Implement coupling matrix calculation
**Title**: Implement coupling matrix calculation algorithm  
**Description**: Calculate dependency counts between all layer pairs. Include percentage calculation and circular dependency detection.  
**Files**:
- `internal/domain/coupling.go` (new, 280 lines)
- `internal/domain/coupling_test.go` (new, 450 lines)  
**Dependencies**: T5  
**Effort**: M (2-4h)  
**Acceptance Criteria**:
- ✅ `CalculateCouplingMatrix(dependencies, layers)` returns `CouplingMatrix`
- ✅ Matrix shows count and percentage: `application→domain: 5 (10%)`
- ✅ Detects circular dependencies (bidirectional imports)
- ✅ Test with 4-layer fixture produces correct 4x4 matrix

---

#### T9 — Implement debt score calculation
**Title**: Implement technical debt score formula  
**Description**: Calculate debt score using formula: `(error_count × 3) + (warning_count × 1) + (circular_deps × 5)`. Add trend multiplier for delta vs previous audit.  
**Files**:
- `internal/domain/debt_score.go` (new, ~90 lines)
- `internal/domain/debt_score_test.go` (new, ~70 lines)  
**Dependencies**: T8  
**Effort**: M (2-4h)  
**Acceptance Criteria**:
- ✅ Base score: errors×3 + warnings×1
- ✅ Circular penalty: +5 per circular dependency
- ✅ Trend multiplier: ×1.5 for increasing debt
- ✅ Density calculation: score / (LOC / 1000)
- ✅ Reproducible: same input → same output

---

#### T10 — Implement AuditService
**Title**: Implement AuditService with full audit workflow  
**Description**: Create `internal/application/audit.go` service that orchestrates: detect dependencies → evaluate rules → calculate metrics → save history → return report.  
**Files**:
- `internal/application/audit.go` (new, ~200 lines)
- `internal/application/audit_test.go` (new, ~150 lines)  
**Dependencies**: T4, T7, T8, T9  
**Effort**: L (>4h)  
**Acceptance Criteria**:
- ✅ `Audit(ctx, projectRoot, configPath)` returns `AuditReport`
- ✅ Saves snapshot to history
- ✅ Calculates coupling matrix
- ✅ Calculates debt score
- ✅ Includes trend comparison if history exists
- ✅ Performance: <5s for 10K LOC project

---

### Phase 4: CLI Command

#### T11 — Create audit CLI command
**Title**: Create `arx audit` Cobra command  
**Description**: Implement `cmd/arx/audit.go` with flags: `--output`, `--trend`, `--since`. Command calls AuditService and renders report.  
**Files**:
- `cmd/arx/audit.go` (new, ~180 lines)  
**Dependencies**: T10  
**Effort**: M (2-4h)  
**Acceptance Criteria**:
- ✅ `arx audit` generates terminal health report
- ✅ `arx audit --output json` produces valid JSON
- ✅ `arx audit --trend` shows delta vs previous audit
- ✅ `arx audit --since 2026-04-14` filters trend period
- ✅ Exit code 1 if violations found

---

#### [x] T12 — Implement terminal report renderer
**Title**: Implement ASCII table renderer for coupling matrix  
**Description**: Create Lip Gloss-based renderer for coupling matrix with color coding: green (≤5%), yellow (5-15%), red (>15%).  
**Files**:
- `internal/infrastructure/output/audit_renderer.go` (new, ~220 lines)
- `internal/infrastructure/output/audit_renderer_test.go` (new, ~180 lines)  
**Dependencies**: T11  
**Effort**: M (2-4h)  
**Acceptance Criteria**:
- ✅ ASCII table with borders for 4x4 matrix
- ✅ Color coding: green/yellow/red based on percentage
- ✅ Shows violation summary by severity
- ✅ Shows layer health scores
- ✅ Shows debt score with trend indicator (↑↓)

---

#### [x] T13 — Implement JSON report renderer
**Title**: Implement JSON output format for audit  
**Description**: Add JSON marshaling for full audit report including coupling matrix and debt breakdown. Added --trend and --since CLI flags.  
**Files**:
- `cmd/arx/audit.go` (edit, +120 lines for trend flags)
- `internal/infrastructure/output/audit_json.go` (new, ~80 lines)  
**Dependencies**: T11  
**Effort**: S (<2h)  
**Acceptance Criteria**:
- ✅ `jq` parses output without errors
- ✅ Includes all metrics: violations, coupling, debt, trends
- ✅ Valid JSON schema
- ✅ `--trend` flag shows only trend comparison
- ✅ `--since DATE` flag filters audits by date

---

### Phase 5: Test Fixtures & Integration

#### [x] T14 — Create Java test fixtures
**Title**: Create Java Maven/Gradle test fixtures  
**Description**: Create sample Java projects for testing: Maven project with multi-module, Gradle project, project with test files and generated code.  
**Files**:
- `test/fixtures/java-maven/` (new directory)
  - `pom.xml`, `src/main/java/com/example/...`
- `test/fixtures/java-gradle/` (new directory)
  - `build.gradle`, `src/main/java/com/example/...`  
**Dependencies**: None (parallel with T1-T4)  
**Effort**: M (2-4h)  
**Acceptance Criteria**:
- ✅ Maven fixture has 2 modules with cross-module dependencies
- ✅ Gradle fixture has standard structure
- ✅ Includes `*Test.java` files for skip testing
- ✅ Includes `target/` directory with generated files

---

#### [x] T15 — Integration tests for Java detector
**Title**: Add integration tests for Java detector  
**Description**: Create `test/integration/java_detector_test.go` to test full detection pipeline on fixtures.  
**Files**:
- `test/integration/java_detector_test.go` (new, ~120 lines)  
**Dependencies**: T14, T4  
**Effort**: M (2-4h)  
**Acceptance Criteria**:
- ✅ Detects Maven fixture as Java project
- ✅ Detects Gradle fixture as Java project
- ✅ Extracts ≥90% of known imports correctly
- ✅ Resolves layers correctly for 95% of imports
- ✅ Skips test files (*Test.java)
- ✅ Skips target/ and build/ directories

---

#### [x] T16 — Integration tests for audit command
**Title**: Add integration tests for `arx audit`  
**Description**: Create `test/integration/audit_test.go` to test full audit workflow including history persistence and trends.  
**Files**:
- `test/integration/audit_test.go` (new, ~200 lines)  
**Dependencies**: T11, T15  
**Effort**: L (>4h)  
**Acceptance Criteria**:
- ✅ `arx audit` on fixture produces expected violations
- ✅ History saved to `.arx-history/`
- ✅ `--trend` flag shows correct delta
- ✅ Retention policy enforced (max 10 audits)
- ✅ JSON output validates against schema

---

### Phase 6: Documentation & Polish

#### T17 — Update README with audit command
**Title**: Document `arx audit` in README  
**Description**: Add section explaining `arx audit` command, flags, output formats, and interpretation of metrics.  
**Files**:
- `README.md` (edit, +80 lines)  
**Dependencies**: T11, T12  
**Effort**: S (<2h)  
**Acceptance Criteria**:
- ✅ Explains difference between `arx check` and `arx audit`
- ✅ Documents all flags: `--output`, `--trend`, `--since`
- ✅ Shows example output with coupling matrix
- ✅ Explains debt score interpretation

---

#### T18 — Add example audit outputs
**Title**: Add example audit outputs to docs  
**Description**: Create `docs/audit-examples.md` with sample terminal and JSON outputs for reference.  
**Files**:
- `docs/audit-examples.md` (new, ~100 lines)  
**Dependencies**: T12, T13  
**Effort**: S (<2h)  
**Acceptance Criteria**:
- ✅ Terminal output example with colors
- ✅ JSON output example with all fields
- ✅ Trend output example showing delta

---

## Summary

### Task Count by Phase
| Phase | Tasks | Total Effort |
|-------|-------|--------------|
| 1. Java Detector | T1-T4 | 2S + 1M = **4-6h** |
| 2. Domain Models | T5-T6 | 1M + 1S = **3-5h** |
| 3. Audit Service | T7-T10 | 2M + 2L = **10-14h** |
| 4. CLI Command | T11-T13 | 2M + 1S = **5-7h** |
| 5. Testing | T14-T16 | 2M + 1L = **8-10h** |
| 6. Documentation | T17-T18 | 2S = **2-4h** |
| **Total** | **18 tasks** | **32-46h** |

---

## Review Workload Forecast

### Lines Changed Estimate
| Category | Files | New Lines | Changed Lines | Total |
|----------|-------|-----------|---------------|-------|
| Java Detector | 2 | 330 | - | 330 |
| Domain Models | 4 | 430 | - | 430 |
| Audit Service | 4 | 630 | - | 630 |
| CLI & Output | 4 | 490 | - | 490 |
| Tests & Fixtures | 6 | 770 | - | 770 |
| Documentation | 2 | 180 | - | 180 |
| **Total** | **22 files** | **2830 lines** | **~0** | **~2830 lines** |

### Review Risk Assessment
| Risk Level | Tasks | Rationale |
|------------|-------|-----------|
| 🔴 High | T10, T16 | Complex orchestration, multiple dependencies, performance-critical |
| 🟡 Medium | T2, T8, T9, T12 | Algorithm-heavy (regex, coupling, debt formula, rendering) |
| 🟢 Low | T1, T3, T4, T5, T6, T7, T11, T13, T14, T15, T17, T18 | Straightforward implementations, well-defined patterns |

---

## Critical Path

```
T1 → T2 → T3 → T4 ─┐
                   ├→ T10 → T11 → T12 → T17
T5 → T6 → T7 ──────┘                   ↑
                   ┌───────────────────┘
T5 → T8 → T9 ──────┘
                   ┌───────────────────┐
T14 → T15 ─────────┘                   ↓
                                      T16 (final integration)
T13 ──────────────────────────────────┘
```

**Critical Path Duration**: T1→T2→T3→T4→T10→T11→T12→T16 = **3S + 4M + 2L = ~18-24h**

---

## Parallelization Opportunities

### Can Run in Parallel (No Dependencies)
| Group | Tasks | Combined Effort |
|-------|-------|-----------------|
| **Group A** (Java Detector) | T1, T2, T3, T4 | 4-6h |
| **Group B** (Domain Models) | T5, T6 | 3-5h |
| **Group C** (Test Fixtures) | T14 | 2-4h |

### Parallelization Strategy
1. **Week 1**: Group A (Java Detector) + Group B (Domain Models) in parallel
   - Developer 1: T1-T4 (Java detector)
   - Developer 2: T5-T6 (Domain models)
   
2. **Week 2**: Group C (Fixtures) + T7-T9 (Storage + Metrics)
   - Developer 1: T14 (Fixtures) + T15 (Integration tests)
   - Developer 2: T7 (History) + T8 (Coupling) + T9 (Debt)
   
3. **Week 3**: T10 (Audit Service) + T11-T13 (CLI)
   - Single developer (dependencies converge)
   
4. **Week 4**: T16 (Final integration) + T17-T18 (Docs)
   - Polish and documentation

### Maximum Parallelism
- **2 developers**: Can reduce total time from ~40h to ~25h (37% reduction)
- **3 developers**: Can reduce to ~20h (50% reduction) — limited by critical path dependencies

---

## Recommendations

1. **Start with T14 (fixtures) in parallel** — Test fixtures are independent and unblock T15 integration tests early.

2. **Prioritize T5 (AuditReport model)** — This is the foundation for all audit-related work; defines the data contract.

3. **Defer T17-T18 (docs) until end** — Documentation depends on final CLI behavior and output formats.

4. **Consider splitting T10 (AuditService)** — If review workload is concern, split into:
   - T10a: Core audit workflow
   - T10b: History integration
   - T10c: Trend calculation

5. **Performance testing** — Add explicit performance benchmarks for T10 (AuditService) to ensure <5s target for 10K LOC.

---

**Next Step**: Ready for implementation (sdd-apply). Tasks are atomic, ordered by dependencies, and sized for reviewable commits.
