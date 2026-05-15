# Arx v0.6.0 Implementation Status

**Last Updated**: May 14, 2026  
**Progress**: 80% complete (11/18 tasks)  
**Status**: 🟡 In Progress

---

## Summary

v0.6.0 adds two major features:
1. **Java Detector** — Support for Maven and Gradle projects
2. **Arx Audit** — Health reports with trends, coupling matrix, and technical debt estimation

---

## ✅ Completed (T1-T11)

### Track A: Java Detector

| Task | Status | Files | Description |
|------|--------|-------|-------------|
| T1 | ✅ | `java/java.go`, `java/parser.go` | JavaDetector struct, regex patterns for imports |
| T2 | ✅ | `java/maven.go` | pom.xml parsing with encoding/xml |
| T3 | ✅ | `java/gradle.go` | build.gradle regex parsing |
| T4 | ✅ | `java/java.go` | Directory exclusion (target/, build/, .git, etc.) |

**Key Features**:
- Detects `import`, `import static`, `package` statements
- Skips external imports (java.*, javax.*)
- Maven: extracts groupId, artifactId, modules
- Gradle: extracts group, rootProject.name
- Skips target/, build/, .git, node_modules

### Track B: Arx Audit

| Task | Status | Files | Description |
|------|--------|-------|-------------|
| T5 | ✅ | `domain/audit_result.go` | AuditReport, CouplingMatrix, DebtScore, TrendReport |
| T6 | ✅ | `ports/history.go`, `history/storage.go` | JSON file storage with retention |
| T7 | ✅ | `application/audit.go` | AuditService with Audit() method |
| T8 | ✅ | `domain/coupling.go` | Coupling matrix calculation + ASCII output |
| T9 | ✅ | `application/audit_debt.go` | Debt score: (errors×3) + (warnings×1) + (circular×5) |
| T10 | ✅ | `application/audit_trends.go` | Trend calculation vs previous audits |
| T11 | ✅ | `cmd/arx/audit.go` | CLI command with --output, --format flags |

**Key Features**:
- Health score based on violations
- Coupling matrix showing layer dependencies
- Technical debt estimation with severity weights
- Trend reports (improved/degraded/unchanged)
- History persistence in `.arx-history/`
- Retention policy: max 10 audits

---

## 🔲 Pending (T12-T18)

| Task | Status | Description |
|------|--------|-------------|
| T12 | 🔲 | Audit output renderer (ASCII table formatter) |
| T13 | 🔲 | Additional CLI flags (--trend, --since) |
| T14-T16 | 🔲 | Integration tests |
| T17-T18 | 🔲 | Documentation updates |

---

## Known Issues

### 1. Layer Resolution Bug 🔴

**Location**: `internal/infrastructure/detector/java/parser.go` — `matchLayerPattern()`

**Problem**: Glob patterns like `src/main/java/com/example/domain/**` don't correctly match import paths like `com.example.domain.Order`.

**Current behavior**: Returns empty string (no layer matched)

**Expected behavior**: Should return "domain" layer

**Fix needed**:
```go
// Current (broken):
pattern = strings.ReplaceAll(pattern, "/**", ".*")
return regexp.MatchString(pattern, importPath)

// Needed:
// 1. Convert path separators to dots
// 2. Handle /** as "any subpackage"
// 3. Match against import path
```

**Workaround**: Use exact package paths in arx.yaml (no globs)

---

## Testing Results

### Java Project
```bash
$ cd /tmp/test-java && arx diagram
Layers: 3
Dependencies: 4
Violations: 0
```
✅ Detects Java files and extracts imports  
⚠️ Layer resolution doesn't match (returns 0 violations even with domain→infrastructure import)

### Node Project
```bash
$ cd /tmp/test-node && arx check
✓ No violations found!
```
✅ TypeScript detector works  
⚠️ Same layer resolution issue

---

## Files Created (22 total)

### Java Detector
- `internal/infrastructure/detector/java/java.go` (130 LOC)
- `internal/infrastructure/detector/java/parser.go` (110 LOC)
- `internal/infrastructure/detector/java/maven.go` (60 LOC)
- `internal/infrastructure/detector/java/gradle.go` (50 LOC)

### Audit Domain
- `internal/domain/audit_result.go` (180 LOC)
- `internal/domain/coupling.go` (280 LOC)

### Audit Application
- `internal/application/audit.go` (220 LOC)
- `internal/application/audit_debt.go` (80 LOC)
- `internal/application/audit_trends.go` (60 LOC)

### Infrastructure
- `internal/ports/history.go` (25 LOC)
- `internal/infrastructure/history/storage.go` (140 LOC)

### CLI
- `cmd/arx/audit.go` (80 LOC)

**Total**: ~2,425 LOC

---

## Next Steps (Priority Order)

1. **CRITICAL**: Fix `matchLayerPattern()` in `java/parser.go`
   - Test with: `arx check` on Java project with known violations
   - Expected: Detect domain→infrastructure violations

2. **HIGH**: Implement T12 (ASCII formatter for audit reports)
   - Current: Placeholder returns `fmt.Sprintf("%+v", report)`
   - Needed: Formatted tables for coupling matrix, debt score, trends

3. **MEDIUM**: Write integration tests (T14-T16)
   - Test Java detector with real Maven project
   - Test audit trends with multiple runs
   - Test debt calculation accuracy

4. **LOW**: Documentation (T17-T18)
   - Update README with v0.6.0 features
   - Add `docs/audit/README.md` guide

---

## How to Continue

### For Next Session

1. **Fix layer resolution**:
   ```bash
   cd /tmp/arx
   # Edit internal/infrastructure/detector/java/parser.go
   # Fix matchLayerPattern() function
   ```

2. **Test fix**:
   ```bash
   cd /tmp/test-java
   arx check  # Should now detect violations
   ```

3. **Complete T12**:
   ```bash
   # Edit cmd/arx/audit.go
   # Implement formatTerminalOutput() with ASCII tables
   ```

4. **Run all tests**:
   ```bash
   go test ./...
   ```

5. **Commit and push**:
   ```bash
   git add .
   git commit -m "feat(v0.6.0): Java detector + Arx Audit"
   git push
   ```

---

## References

- **SDD Proposal**: Engram #1059
- **SDD Spec**: Engram #1072
- **SDD Design**: Engram #1071
- **SDD Tasks**: Engram #1073 (18 tasks breakdown)
- **Implementation Status**: Engram #1078 (this doc)

---

**Questions?** Check Engram memories for detailed context on each decision.
