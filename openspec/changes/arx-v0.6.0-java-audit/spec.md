# Arx v0.6.0 — Java Detector + Arx Audit Specification

## Purpose

Esta especificación define los requirements funcionales para:
1. **Java Detector**: Soporte para proyectos Java (Maven/Gradle)
2. **Arx Audit**: Comando `arx audit` con health reports, trends y métricas de deuda técnica

---

## Domain: java-detector

### Requirement: Java Project Detection

El sistema DEBE detectar proyectos Java buscando archivos de build en el project root.

#### Scenario: Maven project detection

- GIVEN un directorio con `pom.xml` en el root
- WHEN se ejecuta `JavaDetector.Detect()`
- THEN devuelve `true, nil` indicando que es proyecto Java
- AND marca el proyecto como tipo Maven

#### Scenario: Gradle project detection

- GIVEN un directorio con `build.gradle` en el root
- WHEN se ejecuta `JavaDetector.Detect()`
- THEN devuelve `true, nil` indicando que es proyecto Java
- AND marca el proyecto como tipo Gradle

#### Scenario: Multi-module Maven detection

- GIVEN un `pom.xml` con módulos hijos (`<modules>`)
- WHEN se ejecuta `Detect()`
- THEN identifica los subdirectorios de cada módulo
- AND aplica el detector recursivamente en cada módulo

#### Scenario: Non-Java project

- GIVEN un directorio sin `pom.xml` ni `build.gradle`
- WHEN se ejecuta `JavaDetector.Detect()`
- THEN devuelve `false, nil`

### Requirement: Java Import Extraction

El sistema DEBE extraer declaraciones `package` e `import` de archivos `.java` usando regex.

#### Scenario: Standard import parsing

- GIVEN un archivo `OrderService.java` con `import java.util.List;`
- WHEN se ejecuta `ExtractImports()`
- THEN extrae `(OrderService.java, línea, "java.util.List")`

#### Scenario: Package declaration parsing

- GIVEN un archivo con `package com.example.domain.order;`
- WHEN se ejecuta `ExtractImports()`
- THEN identifica el package raíz para resolución de layers

#### Scenario: Static import parsing

- GIVEN un archivo con `import static java.lang.Math.PI;`
- WHEN se ejecuta `ExtractImports()`
- THEN extrae el import estático completo

#### Scenario: Wildcard import parsing

- GIVEN un archivo con `import com.example.domain.*;`
- WHEN se ejecuta `ExtractImports()`
- THEN extrae el wildcard import (resolución a layer requiere matching de prefijo)

#### Scenario: Skip test files

- GIVEN un archivo `OrderServiceTest.java` en configuración `exclude: ["**/*Test.java"]`
- WHEN se ejecuta `ExtractImports()`
- THEN omite el archivo del análisis

#### Scenario: Skip generated directories

- GIVEN archivos en `target/` o `build/`
- WHEN se ejecuta `ExtractImports()`
- THEN excluye automáticamente estos directorios

### Requirement: Layer Resolution for Java

El sistema DEBE resolver imports Java a layers definidos en `arx.yaml`.

#### Scenario: Direct package match

- GIVEN layer `domain` con path `src/main/java/com/example/domain/**`
- GIVEN import `com.example.domain.order.Order`
- WHEN se resuelve el import
- THEN asigna el import al layer `domain`

#### Scenario: External dependency skip

- GIVEN import `org.springframework.boot.SpringApplication`
- GIVEN ningún layer cubre `org.springframework`
- WHEN se resuelve el import
- THEN marca como dependencia externa (no viola reglas internas)

---

## Domain: audit-command

### Requirement: Audit Command Interface

El sistema DEBE proveer el comando `arx audit` con flags de configuración.

#### Scenario: Basic audit execution

- GIVEN un proyecto con `arx.yaml` configurado
- WHEN se ejecuta `arx audit`
- THEN genera health report en terminal
- AND guarda snapshot en `.arx-history/audit-YYYY-MM-DD.json`

#### Scenario: JSON output format

- GIVEN un proyecto con violaciones
- WHEN se ejecuta `arx audit --output json`
- THEN produce JSON válido con métricas completas
- AND el JSON incluye coupling matrix y debt score

#### Scenario: Trend comparison

- GIVEN al menos 2 auditorías previas en `.arx-history/`
- WHEN se ejecuta `arx audit --trend`
- THEN muestra delta de violaciones vs auditoría anterior
- AND indica si la arquitectura mejoró o degradó

#### Scenario: Since date filter

- GIVEN auditorías desde hace 30 días
- WHEN se ejecuta `arx audit --since 2026-04-14`
- THEN filtra trends solo para el período especificado

### Requirement: Health Report Generation

El sistema DEBE generar un reporte de salud arquitectónica con métricas específicas.

#### Scenario: Violation summary

- GIVEN 15 violaciones (10 error, 5 warning)
- WHEN se genera el health report
- THEN muestra total violaciones por severidad
- AND muestra densidad (violaciones / 1000 líneas de código)

#### Scenario: Layer health score

- GIVEN layer `domain` con 0 violaciones
- GIVEN layer `infrastructure` con 8 violaciones
- WHEN se genera el health report
- THEN asigna score 100% a domain
- AND asigna score proporcional a infrastructure

#### Scenario: Coupling matrix display

- GIVEN 4 layers: domain, application, infrastructure, cmd
- WHEN se genera el health report
- THEN muestra matriz 4x4 con counts de dependencias
- AND resalta en rojo acoplamientos que violan reglas

---

## Domain: coupling-matrix

### Requirement: Coupling Calculation

El sistema DEBE calcular métricas de acoplamiento entre cada par de layers.

#### Scenario: Direct dependency count

- GIVEN 5 archivos en `application` importan `domain`
- WHEN se calcula coupling matrix
- THEN celda `application→domain` = 5

#### Scenario: Percentage calculation

- GIVEN 50 imports totales desde `application`
- GIVEN 5 imports hacia `domain`
- WHEN se calcula coupling matrix
- THEN celda `application→domain` muestra `5 (10%)`

#### Scenario: Circular dependency detection

- GIVEN `domain` importa `infrastructure` (1 vez)
- GIVEN `infrastructure` importa `domain` (3 veces)
- WHEN se calcula coupling matrix
- THEN marca la relación como circular
- AND suma al debt score con peso ×2

### Requirement: Coupling Matrix Output

El sistema DEBE renderizar la matriz en formatos legibles.

#### Scenario: Terminal ASCII table

- GIVEN 4 layers
- WHEN se renderiza en terminal
- THEN produce tabla ASCII con bordes (Lip Gloss)
- AND usa colores: verde (≤5%), amarillo (5-15%), rojo (>15%)

#### Scenario: JSON matrix structure

- GIVEN 4 layers con acoplamientos
- WHEN se exporta `--output json`
- THEN produce objeto `{ "from_layer": { "to_layer": count } }`

---

## Domain: debt-estimation

### Requirement: Technical Debt Score Calculation

El sistema DEBE calcular un score numérico de deuda técnica arquitectónica.

#### Scenario: Base score formula

- GIVEN 10 violaciones `error` (peso=3) y 5 violaciones `warning` (peso=1)
- WHEN se calcula debt score
- THEN score = (10×3) + (5×1) = 35 puntos

#### Scenario: Trend multiplier

- GIVEN debt score actual = 35
- GIVEN debt score anterior = 28
- WHEN se calcula trend
- THEN delta = +7 (deuda aumentó)
- AND aplica multiplier ×1.5 al delta positivo

#### Scenario: Circular dependency penalty

- GIVEN 2 dependencias circulares detectadas
- WHEN se calcula debt score
- THEN suma penalty = 2 × 5 = 10 puntos extra

#### Scenario: Debt density metric

- GIVEN debt score = 45, proyecto = 9000 LOC
- WHEN se calcula debt density
- THEN density = 45 / 9 = 5 puntos por KLOC

### Requirement: Debt Trend Tracking

El sistema DEBE mostrar evolución de deuda técnica en el tiempo.

#### Scenario: Improvement trend

- GIVEN scores históricos: [50, 42, 35]
- WHEN se renderiza trend
- THEN muestra gráfico ASCII descendente
- AND mensaje "Deuda reducida 30% en 3 auditorías"

#### Scenario: Degradation alert

- GIVEN score actual > score anterior + 20%
- WHEN se renderiza trend
- THEN muestra alerta roja "⚠️ Deuda aumentó significativamente"

---

## Domain: audit-history

### Requirement: Audit Persistence

El sistema DEBE guardar auditorías en `.arx-history/` para comparación histórica.

#### Scenario: Audit file naming

- GIVEN auditoría ejecutada el 2026-05-14
- WHEN se guarda el snapshot
- THEN archivo = `.arx-history/audit-2026-05-14.json`

#### Scenario: Last audit symlink

- GIVEN múltiples auditorías guardadas
- WHEN se completa nueva auditoría
- THEN crea/actualiza `.arx-history/last-audit.json` como referencia

#### Scenario: Retention policy

- GIVEN 15 auditorías en `.arx-history/`
- WHEN se guarda auditoría #16
- THEN borra la auditoría más antigua
- AND mantiene máximo 10 auditorías

### Requirement: Historical Data Loading

El sistema DEBE cargar auditorías previas para cálculo de trends.

#### Scenario: Load previous audit

- GIVEN `.arx-history/last-audit.json` existe
- WHEN se ejecuta `arx audit --trend`
- THEN carga datos de la auditoría anterior
- AND calcula deltas correctamente

#### Scenario: Missing history handling

- GIVEN `.arx-history/` vacío o inexistente
- WHEN se ejecuta `arx audit --trend`
- THEN muestra mensaje "Sin historial previo — trends disponibles desde próxima auditoría"
- AND no falla el comando

#### Scenario: Corrupted file handling

- GIVEN `.arx-history/last-audit.json` con JSON inválido
- WHEN se intenta cargar
- THEN muestra warning y omite el archivo corrupto
- AND continúa con auditoría actual

---

## Domain: detector-registry (Modified)

### Requirement: Detector Registration

El sistema DEBE registrar JavaDetector junto a los detectores existentes.

#### Scenario: Java detector in registry

- GIVEN proyecto con archivos `.java` y `pom.xml`
- WHEN se ejecuta `arx check` o `arx audit`
- THEN DetectorRegistry incluye JavaDetector
- AND ejecuta `JavaDetector.ExtractImports()` junto a Go/TS detectors

#### Scenario: Multi-language project

- GIVEN proyecto con `.go`, `.ts`, y `.java`
- WHEN se ejecuta `arx audit`
- THEN todos los detectores aplicables se ejecutan
- AND resultados se consolidan en unificado health report

---

## Acceptance Criteria

### Functional Criteria

| ID | Criterion | Verification |
|----|-----------|--------------|
| AC-01 | JavaDetector detecta Maven (`pom.xml`) | Test fixture con pom.xml → Detect() = true |
| AC-02 | JavaDetector detecta Gradle (`build.gradle`) | Test fixture con build.gradle → Detect() = true |
| AC-03 | JavaDetector extrae imports con ≥90% precisión | Test con 100 imports conocidos → ≥90 correctos |
| AC-04 | `arx audit` genera coupling matrix en terminal | Output visual con tabla ASCII |
| AC-05 | `arx audit --output json` produce JSON válido | `jq` parsea sin errores |
| AC-06 | Trend comparison muestra delta vs auditoría previa | Test con 2 audits históricas → delta calculado |
| AC-07 | Debt score es reproducible (misma fórmula) | 2 ejecuciones → mismo score |
| AC-08 | Retention policy mantiene máx 10 auditorías | Test con 11 audits → 10 archivos en disco |

### Non-Functional Criteria

| ID | Criterion | Target |
|----|-----------|--------|
| NFC-01 | Performance: audit en proyecto 10K LOC | < 5 segundos |
| NFC-02 | Memory usage | < 100 MB RAM |
| NFC-03 | Test coverage | ≥80% en audit.go y java_detector.go |
| NFC-04 | Binary size increase | < 2 MB vs v0.5.0 |

---

## Artifacts

| Artifact | Path | Description |
|----------|------|-------------|
| Spec document | `openspec/changes/arx-v0.6.0-java-audit/spec.md` | Este documento |
| Java detector | `internal/infrastructure/detector/java_detector.go` | Implementación |
| Audit service | `internal/application/audit.go` | Lógica de auditoría |
| History storage | `internal/infrastructure/history/json_history.go` | Persistencia |
| Audit command | `cmd/arx/audit.go` | Cobra command |
| Test fixtures | `test/fixtures/java-maven/`, `test/fixtures/java-gradle/` | Proyectos de prueba |

---

## Next Step

Ready for **design phase** (sdd-design). El diseño debe definir:
- Estructura de datos para `AuditReport`, `CouplingMatrix`, `DebtScore`
- Algoritmo de cálculo de trends
- Regex patterns específicos para Java imports
- Schema JSON para `.arx-history/audit-*.json`
