# Arx — Architecture Audit CLI

## Project Charter / Carta del Proyecto

---

### EN: Vision

**Arx** is a cross-language architecture audit CLI that validates architectural rules against real codebases and explains *why* violations matter and *how* to fix them. It is not a linter, not a static analyzer, and not a code quality tool. It is an **architecture guard with a teaching soul**: every violation comes with a didactic explanation that helps developers understand architectural principles, not just fix a warning.

Arx exists because teams adopt Clean Architecture, Hexagonal Architecture, or Domain-Driven Design in name only — the code tells a different story. Domain depends on infrastructure. Application layer imports database drivers. Circular dependencies creep in unnoticed. Existing tools either ignore architecture entirely (ESLint, SonarQube) or are locked to a single language (ArchUnit for Java, Deptrac for PHP). Arx fills this gap.

### ES: Visión

**Arx** es una CLI de auditoría arquitectónica multilenguaje que valida reglas de arquitectura contra codebases reales y explica *por qué* las violaciones importan y *cómo* arreglarlas. No es un linter, no es un analizador estático, y no es una herramienta de calidad de código. Es una **guardia de arquitectura con alma de profesora**: cada violación viene con una explicación didáctica que ayuda a los desarrolladores a entender los principios arquitectónicos, no solo a corregir un aviso.

Arx existe porque los equipos adoptan Clean Architecture, Arquitectura Hexagonal o Domain-Driven Design solo de nombre — el código cuenta otra historia. El dominio depende de infraestructura. La capa de aplicación importa drivers de base de datos. Las dependencias circulares se cuelan sin que nadie lo note. Las herramientas existentes o bien ignoran la arquitectura por completo (ESLint, SonarQube) o están bloqueadas a un solo lenguaje (ArchUnit para Java, Deptrac para PHP). Arx cubre este hueco.

---

## EN: Problem Statement

1. **Architecture erodes invisibly.** No CI check catches when a domain module imports a database driver. By the time someone notices, the damage is structural.
2. **Existing tools are language-bound.** ArchUnit (Java), Deptrac (PHP), NetArchTest (.NET) — each works only within its ecosystem. Polyglot teams are left without coverage.
3. **Linters don't teach.** SonarQube says "Remove this dependency." Arx says "Your domain layer should not know about PostgreSQL because swapping your database should not require changing business logic. Here is how to fix it: define an interface in domain, implement it in infrastructure, inject via constructor."
4. **Architecture decisions are undocumented.** Teams debate Clean Architecture in meetings, then nothing enforces it. ADRs sit in a wiki, disconnected from the code they govern.
5. **No tool connects rules to code.** Architecture decision records are prose. Arx makes them executable.

### ES: Problema

1. **La arquitectura se degrada de forma invisible.** Ningún check de CI detecta cuando un módulo de dominio importa un driver de base de datos. Cuando alguien se da cuenta, el daño ya es estructural.
2. **Las herramientas existentes están ligadas a un lenguaje.** ArchUnit (Java), Deptrac (PHP), NetArchTest (.NET) — cada una funciona solo dentro de su ecosistema. Los equipos poliglotas se quedan sin cobertura.
3. **Los linters no enseñan.** SonarQube dice "Elimina esta dependencia." Arx dice "Tu capa de dominio no debería conocer PostgreSQL porque cambiar de base de datos no debería requerir modificar la lógica de negocio. Así se arregla: define una interfaz en dominio, impleméntala en infraestructura, inyéctala por constructor."
4. **Las decisiones arquitectónicas no están documentadas.** Los equipos debaten Clean Architecture en reuniones, después nada lo aplica. Los ADRs están en una wiki, desconectados del código que gobiernan.
5. **Ninguna herramienta conecta reglas con código.** Los registros de decisiones arquitectónicas son prosa. Arx los hace ejecutables.

---

## EN: Target Users

| User | Pain Point | How Arx Helps |
|------|-----------|----------------|
| Tech leads on polyglot teams | No unified architecture enforcement | One tool, all languages, same rules |
| Devs learning Clean/Hexagonal Architecture | "I read the blog post but my code still has coupling" | Didactic explanations teach the principle behind the rule |
| Solo devs on side projects | Architecture drifts silently | `arx check` in pre-commit or CI |
| OSS maintainers | PRs introduce architectural violations | `arx check --ci` fails the PR with explanations |
| Engineering managers | "Are we actually following our architecture?" | `arx audit` produces a report |

### ES: Usuarios Objetivo

| Usuario | Punto de Dolor | Cómo Ayuda Arx |
|----------|---------------|-----------------|
| Tech leads en equipos poliglotas | Sin aplicación de arquitectura unificada | Una herramienta, todos los lenguajes, mismas reglas |
| Desarrolladores aprendiendo Clean/Hexagonal Architecture | "Leí el artículo pero mi código sigue acoplado" | Explicaciones didácticas enseñan el principio detrás de la regla |
| Desarrolladores individuales en proyectos personales | La arquitectura se degrada en silencio | `arx check` en pre-commit o CI |
| Mantenedores de OSS | Los PRs introducen violaciones arquitectónicas | `arx check --ci` falla el PR con explicaciones |
| Engineering managers | "¿Estamos siguiendo realmente nuestra arquitectura?" | `arx audit` produce un informe |

---

## EN: Core Concepts

### Layer
A named grouping of source files that shares a responsibility. Layers are the fundamental unit of architectural rules. A layer is defined by a path pattern (glob) and a human-readable name.

Example: `domain` = `src/domain/**`, `infrastructure` = `src/infrastructure/**`

### Rule
A declarative constraint on how layers may relate. Rules are authored in YAML and version-controlled alongside the code they govern.

Example: `domain CANNOT depend on infrastructure`, `application MUST depend on domain`

### Violation
A specific import or dependency that breaks a rule. Each violation carries:
- The violating file and line
- The rule broken
- The layers involved
- A human-readable explanation of *why* this rule exists
- A suggested fix path

### Detector
A language-specific module that parses source files and extracts import/dependency information. Detectors are plugins. The MVP ships with detectors for Go and TypeScript; the plugin system allows community contributions.

### Report
The output of `arx check`. Available in:
- **Terminal** (default): Colored, annotated, educational
- **CI** (`--ci`): JSON or SARIF for machine consumption
- **Markdown** (`--format md`): Commitable to repo documentation

### ES: Conceptos Núcleo

### Capa (Layer)
Un agrupamiento nombrado de archivos fuente que comparte una responsabilidad. Las capas son la unidad fundamental de las reglas arquitectónicas. Una capa se define mediante un patrón de ruta (glob) y un nombre legible.

Ejemplo: `domain` = `src/domain/**`, `infrastructure` = `src/infrastructure/**`

### Regla (Rule)
Una restricción declarativa sobre cómo las capas pueden relacionarse. Las reglas se escriben en YAML y se versionan junto con el código que gobiernan.

Ejemplo: `domain NO PUEDE depender de infrastructure`, `application DEBE depender de domain`

### Violación (Violation)
Una importación o dependencia específica que rompe una regla. Cada violación incluye:
- El archivo y línea que viola
- La regla infringida
- Las capas involucradas
- Una explicación legible de *por qué* existe esta regla
- Una ruta sugerida para corregirla

### Detector
Un módulo específico de lenguaje que analiza archivos fuente y extrae información de importaciones/dependencias. Los detectores son plugins. El MVP incluye detectores para Go y TypeScript; el sistema de plugins permite contribuciones de la comunidad.

### Informe (Report)
La salida de `arx check`. Disponible en:
- **Terminal** (por defecto): Coloreado, anotado, educativo
- **CI** (`--ci`): JSON o SARIF para consumo de máquinas
- **Markdown** (`--format md`): Se puede commitar en la documentación del repositorio

---

## EN: Technical Architecture of Arx Itself

Arx follows Hexagonal Architecture. It is its own first user.

```
arx/
├── cmd/
│   └── arx/                 # CLI entrypoint (adapter: CLI)
│       └── main.go
├── internal/
│   ├── domain/              # Core business logic
│   │   ├── layer.go         # Layer entity
│   │   ├── rule.go          # Rule entity
│   │   ├── violation.go     # Violation entity
│   │   └── audit.go         # Audit orchestration (pure logic, no I/O)
│   ├── application/          # Use cases
│   │   ├── check.go         # Check command handler
│   │   ├── init.go          # Init command handler
│   │   ├── explain.go       # Explain command handler
│   │   └── audit.go         # Audit command handler
│   ├── infrastructure/
│   │   ├── detector/
│   │   │   ├── go/          # Go import detector
│   │   │   ├── typescript/  # TypeScript import detector
│   │   │   └── registry.go  # Detector plugin registry
│   │   ├── config/
│   │   │   └── yaml.go     # YAML config parser
│   │   ├── output/
│   │   │   ├── terminal.go  # Terminal (colored) output
│   │   │   ├── json.go      # JSON output
│   │   │   ├── sarif.go     # SARIF output
│   │   │   └── markdown.go  # Markdown output
│   │   └── fs/
│   │       └── walker.go    # Filesystem walker
│   └── ports/
│       ├── detector.go      # Detector interface
│       ├── config.go        # Config reader interface
│       ├── reporter.go      # Reporter interface
│       └── filewalker.go    # File walker interface
├── configs/
│   └── arx.example.yaml
├── docs/
├── test/
│   ├── unit/
│   ├── integration/
│   └── fixtures/            # Sample projects for E2E tests
└── go.mod
```

### ES: Arquitectura Técnica del Propio Arx

Arx sigue Arquitectura Hexagonal. Es su propio primer usuario.

```
arx/
├── cmd/
│   └── arx/                 # Punto de entrada CLI (adaptador: CLI)
│       └── main.go
├── internal/
│   ├── domain/              # Lógica de negocio núcleo
│   │   ├── layer.go         # Entidad Capa
│   │   ├── rule.go          # Entidad Regla
│   │   ├── violation.go     # Entidad Violación
│   │   └── audit.go         # Orquestación de auditoría (lógica pura, sin I/O)
│   ├── application/          # Casos de uso
│   │   ├── check.go         # Handler del comando check
│   │   ├── init.go          # Handler del comando init
│   │   ├── explain.go       # Handler del comando explain
│   │   └── audit.go         # Handler del comando audit
│   ├── infrastructure/
│   │   ├── detector/
│   │   │   ├── go/          # Detector de imports Go
│   │   │   ├── typescript/  # Detector de imports TypeScript
│   │   │   └── registry.go  # Registro de plugins detector
│   │   ├── config/
│   │   │   └── yaml.go     # Parser de configuración YAML
│   │   ├── output/
│   │   │   ├── terminal.go  # Salida terminal (coloreada)
│   │   │   ├── json.go     # Salida JSON
│   │   │   ├── sarif.go    # Salida SARIF
│   │   │   └── markdown.go  # Salida Markdown
│   │   └── fs/
│   │       └── walker.go    # Recorredor de sistema de archivos
│   └── ports/
│       ├── detector.go      # Interfaz Detector
│       ├── config.go        # Interfaz lector de configuración
│       ├── reporter.go      # Interfaz Reportero
│       └── filewalker.go    # Interfaz recorredor de archivos
├── configs/
│   └── arx.example.yaml
├── docs/
├── test/
│   ├── unit/
│   ├── integration/
│   └── fixtures/            # Proyectos de ejemplo para tests E2E
└── go.mod
```

---

## EN: CLI Interface Contract

### `arx init`

Scans the project directory, detects languages, infers layer structure from directory conventions, and generates an `arx.yaml` config file with sensible defaults.

```
$ arx init
Detected: Go (go.mod), TypeScript (tsconfig.json)

Suggested layers:
  domain         → internal/domain/**
  application    → internal/application/**
  infrastructure → internal/infrastructure/**
  ports          → internal/ports/**
  cmd            → cmd/**

Suggested rules:
  domain CANNOT depend on infrastructure
  domain CANNOT depend on cmd
  application MUST depend on domain
  application CANNOT depend on infrastructure
  cmd CAN depend on all

Written to arx.yaml — review and adjust.
```

### `arx check`

Runs the audit. Exits with code 0 if all rules pass, 1 if violations found.

```
$ arx check
✓ domain → (no violations)
✓ application → depends on domain (valid)
✗ infrastructure/postgres/order_repo.go:14
  violates: domain CANNOT depend on infrastructure
  import: "github.com/example/app/internal/infrastructure/postgres"

❌ [D-01] domain/order.go:14 → infrastructure/postgres.go
   ────────────────────────────────────────────────────────
   Rule: "domain" MUST NOT depend on "infrastructure"

   Why this matters:
   The domain layer is the heart of your business logic. It should
   not know HOW data is persisted — only THAT it is persisted.
   If domain imports your database driver, changing from PostgreSQL
   to MongoDB requires modifying business rules. That coupling
   direction is backwards.

   How to fix:
   1. Define an interface in domain (e.g., OrderRepository)
   2. Move the PostgreSQL implementation to infrastructure
   3. Inject the implementation via constructor (Dependency Inversion)

Found 1 violation across 47 files. Run `arx explain D-01` for details.
```

### `arx check --ci`

Machine-readable output for CI/CD pipelines.

```json
{
  "version": "1.0",
  "tool": "arx",
  "violations": [
    {
      "id": "D-01",
      "rule": "domain-cannot-depend-on-infrastructure",
      "severity": "error",
      "file": "internal/domain/order.go",
      "line": 14,
      "source_layer": "domain",
      "target_layer": "infrastructure",
      "import": "github.com/example/app/internal/infrastructure/postgres",
      "message": "domain MUST NOT depend on infrastructure"
    }
  ],
  "summary": {
    "total": 1,
    "errors": 1,
    "warnings": 0
  }
}
```

### `arx explain <violation-id>`

Shows the full didactic explanation for a specific violation, including architectural context and step-by-step fix guidance.

### `arx audit`

Produces a complete architectural health report: dependency graph, layer coupling matrix, violation counts, and trend tracking (if previous audits exist).

### ES: Contrato de Interfaz CLI

### `arx init`

Escanea el directorio del proyecto, detecta lenguajes, infiere la estructura de capas a partir de convenciones de directorios y genera un archivo de configuración `arx.yaml` con valores por defecto razonables.

```
$ arx init
Detectado: Go (go.mod), TypeScript (tsconfig.json)

Capas sugeridas:
  domain         → internal/domain/**
  application    → internal/application/**
  infrastructure → internal/infrastructure/**
  ports          → internal/ports/**
  cmd            → cmd/**

Reglas sugeridas:
  domain NO PUEDE depender de infrastructure
  domain NO PUEDE depender de cmd
  application DEBE depender de domain
  application NO PUEDE depender de infrastructure
  cmd PUEDE depender de todo

Escrito en arx.yaml — revisar y ajustar.
```

### `arx check`

Ejecuta la auditoría. Sale con código 0 si todas las reglas pasan, 1 si hay violaciones.

```
$ arx check
✓ domain → (sin violaciones)
✓ application → depende de domain (válido)
✗ infrastructure/postgres/order_repo.go:14
  viola: domain NO PUEDE depender de infrastructure
  import: "github.com/example/app/internal/infrastructure/postgres"

❌ [D-01] domain/order.go:14 → infrastructure/postgres.go
   ────────────────────────────────────────────────────────
   Regla: "domain" NO DEBE depender de "infrastructure"

   Por qué importa:
   La capa de dominio es el corazón de la lógica de negocio. No debe
   saber CÓMO se persisten los datos — solo QUE se persisten.
   Si el dominio importa el driver de la base de datos, cambiar de
   PostgreSQL a MongoDB requiere modificar reglas de negocio. Esa
   dirección de acoplamiento está invertida.

   Cómo corregirlo:
   1. Define una interfaz en domain (ej: OrderRepository)
   2. Mueve la implementación PostgreSQL a infrastructure
   3. Inyecta la implementación por constructor (Inversión de Dependencia)

Encontrada 1 violación en 47 archivos. Ejecuta `arx explain D-01` para detalles.
```

### `arx check --ci`

Salida legible por máquina para pipelines de CI/CD.

```json
{
  "version": "1.0",
  "tool": "arx",
  "violations": [
    {
      "id": "D-01",
      "rule": "domain-cannot-depend-on-infrastructure",
      "severity": "error",
      "file": "internal/domain/order.go",
      "line": 14,
      "source_layer": "domain",
      "target_layer": "infrastructure",
      "import": "github.com/example/app/internal/infrastructure/postgres",
      "message": "domain NO DEBE depender de infrastructure"
    }
  ],
  "summary": {
    "total": 1,
    "errors": 1,
    "warnings": 0
  }
}
```

### `arx explain <violation-id>`

Muestra la explicación didáctica completa para una violación específica, incluyendo contexto arquitectónico y guía paso a paso para corregirla.

### `arx audit`

Produce un informe completo de salud arquitectónica: grafo de dependencias, matriz de acoplamiento entre capas, recuento de violaciones y seguimiento de tendencias (si existen auditorías previas).

---

## EN: Configuration Schema

```yaml
# arx.yaml — Architecture rules definition
version: "1.0"

layers:
  domain:
    description: "Core business logic — no external dependencies"
    paths:
      - "internal/domain/**"
    tags: [core, business]

  application:
    description: "Use cases and orchestration — depends on domain only"
    paths:
      - "internal/application/**"
    tags: [usecase, orchestration]

  infrastructure:
    description: "External implementations — databases, APIs, frameworks"
    paths:
      - "internal/infrastructure/**"
    tags: [external, implementation]

  ports:
    description: "Interfaces and contracts that infrastructure must implement"
    paths:
      - "internal/ports/**"
    tags: [interface, contract]

  cmd:
    description: "CLI and entry points — wires everything together"
    paths:
      - "cmd/**"
    tags: [entrypoint]

rules:
  - id: domain-purity
    from: domain
    to: [infrastructure, cmd]
    type: cannot
    severity: error
    explanation: |
      The domain layer must not depend on infrastructure or entry points.
      Business rules should be expressible without knowing about databases,
      web frameworks, or CLI tools. Violations here mean your business logic
      is coupled to implementation details — changes in infrastructure
      will ripple into core business rules.

  - id: application-depends-on-domain
    from: application
    to: [domain]
    type: must
    severity: error
    explanation: |
      The application layer exists to orchestrate domain operations.
      If it doesn't depend on domain, it's not doing its job — either
      business logic has leaked into application, or the use case
      is empty.

  - id: application-infrastructure-isolation
    from: application
    to: [infrastructure]
    type: cannot
    severity: warning
    explanation: |
      Application should depend on abstractions (ports), not concrete
      infrastructure. If application imports infrastructure directly,
      the Dependency Inversion Principle is violated. Use interfaces
      defined in ports/ and inject implementations at the composition root.

  - id: cmd-can-depend-on-all
    from: cmd
    to: [domain, application, infrastructure, ports]
    type: can
    severity: info

language_overrides:
  go:
    import_style: "module_path"
    module_prefix: "github.com/example/app"
  typescript:
    import_style: "path_alias"
    path_aliases:
      "@domain": "src/domain"
      "@app": "src/application"
      "@infra": "src/infrastructure"

exclude:
  - "**/*_test.go"
  - "**/*.spec.ts"
  - "**/mock/**"
  - "vendor/**"
  - "node_modules/**"

severity_config:
  error:
    exit_code: 1
    label: "❌"
  warning:
    exit_code: 0
    label: "⚠️"
  info:
    exit_code: 0
    label: "ℹ️"
```

### ES: Esquema de Configuración

```yaml
# arx.yaml — Definición de reglas arquitectónicas
version: "1.0"

layers:
  domain:
    description: "Lógica de negocio núcleo — sin dependencias externas"
    paths:
      - "internal/domain/**"
    tags: [core, business]

  application:
    description: "Casos de uso y orquestación — depende solo de domain"
    paths:
      - "internal/application/**"
    tags: [usecase, orchestration]

  infrastructure:
    description: "Implementaciones externas — bases de datos, APIs, frameworks"
    paths:
      - "internal/infrastructure/**"
    tags: [external, implementation]

  ports:
    description: "Interfaces y contratos que la infraestructura debe implementar"
    paths:
      - "internal/ports/**"
    tags: [interface, contract]

  cmd:
    description: "CLI y puntos de entrada — conecta todo"
    paths:
      - "cmd/**"
    tags: [entrypoint]

rules:
  - id: domain-purity
    from: domain
    to: [infrastructure, cmd]
    type: cannot
    severity: error
    explanation: |
      La capa de dominio no debe depender de infraestructura ni de puntos de entrada.
      Las reglas de negocio deben poder expresarse sin conocer bases de datos,
      frameworks web ni herramientas CLI. Las violaciones aquí significan que la
      lógica de negocio está acoplada a detalles de implementación — los cambios
      en infraestructura se propagarán a las reglas de negocio núcleo.

  - id: application-depends-on-domain
    from: application
    to: [domain]
    type: must
    severity: error
    explanation: |
      La capa de aplicación existe para orquestar operaciones del dominio.
      Si no depende de domain, no está haciendo su trabajo — o la lógica
      de negocio se ha filtrado en application, o el caso de uso está vacío.

  - id: application-infrastructure-isolation
    from: application
    to: [infrastructure]
    type: cannot
    severity: warning
    explanation: |
      Application debe depender de abstracciones (ports), no de infraestructura
      concreta. Si application importa infraestructura directamente, se viola
      el Principio de Inversión de Dependencias. Usa interfaces definidas en
      ports/ e inyecta implementaciones en la raíz de composición.

  - id: cmd-can-depend-on-all
    from: cmd
    to: [domain, application, infrastructure, ports]
    type: can
    severity: info

language_overrides:
  go:
    import_style: "module_path"
    module_prefix: "github.com/example/app"
  typescript:
    import_style: "path_alias"
    path_aliases:
      "@domain": "src/domain"
      "@app": "src/application"
      "@infra": "src/infrastructure"

exclude:
  - "**/*_test.go"
  - "**/*.spec.ts"
  - "**/mock/**"
  - "vendor/**"
  - "node_modules/**"

severity_config:
  error:
    exit_code: 1
    label: "❌"
  warning:
    exit_code: 0
    label: "⚠️"
  info:
    exit_code: 0
    label: "ℹ️"
```

---

## EN: Language Detection Strategy

Each detector plugin is responsible for:

1. **Identifying if it applies** — given a project directory, can this detector find files it understands?
2. **Extracting imports** — parse source files and return a list of `(file, line, import_path)` tuples.
3. **Resolving import paths to layers** — given an import path and the layer definitions, determine which layer (if any) the import belongs to.

### MVP Detectors

| Language | Import Parsing Method | Priority |
|----------|----------------------|----------|
| Go | AST via `go/ast` standard library | P0 |
| TypeScript | Regex-based import extraction (fast), optional AST via `tree-sitter` for precision | P0 |
| Python | Regex + `ast` module | P1 |
| Java | `package` + `import` regex | P1 |
| Rust | `use` statement regex | P2 |

### Detector Interface

```go
// Port: detector.go
type Detector interface {
    // Name returns the language name (e.g., "go", "typescript")
    Name() string
    
    // Detect checks if this detector applies to the given project
    Detect(ctx context.Context, projectRoot string) (bool, error)
    
    // ExtractImports parses source files and returns dependency information
    ExtractImports(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Violation, error)
}
```

### ES: Estrategia de Detección de Lenguajes

Cada plugin detector es responsable de:

1. **Identificar si aplica** — dado un directorio de proyecto, ¿puede este detector encontrar archivos que entiende?
2. **Extraer imports** — analizar archivos fuente y devolver una lista de tuplas `(archivo, línea, ruta_import)`.
3. **Resolver rutas de import a capas** — dada una ruta de import y las definiciones de capas, determinar a qué capa (si alguna) pertenece el import.

### Detectores MVP

| Lenguaje | Método de Parseo de Imports | Prioridad |
|----------|-----------------------------|-----------|
| Go | AST vía librería estándar `go/ast` | P0 |
| TypeScript | Extracción por regex (rápido), opcionalmente AST vía `tree-sitter` para precisión | P0 |
| Python | Regex + módulo `ast` | P1 |
| Java | `package` + `import` por regex | P1 |
| Rust | Regex sobre statements `use` | P2 |

### Interfaz del Detector

```go
// Puerto: detector.go
type Detector interface {
    // Name devuelve el nombre del lenguaje (ej: "go", "typescript")
    Name() string
    
    // Detect comprueba si este detector aplica al proyecto dado
    Detect(ctx context.Context, projectRoot string) (bool, error)
    
    // ExtractImports analiza archivos fuente y devuelve información de dependencias
    ExtractImports(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Violation, error)
}
```

---

## EN: Rule Types

| Type | Semantics | Example |
|------|-----------|---------|
| `cannot` | Source layer MUST NOT import target layer | domain CANNOT depend on infrastructure |
| `must` | Source layer MUST import target layer (at least one file) | application MUST depend on domain |
| `can` | Source layer is ALLOWED to import target layer (informational, never fails) | cmd CAN depend on all |
| `must_not_circular` | No circular dependencies between listed layers | domain and infrastructure MUST NOT be circular |

Each rule carries:
- `id`: Unique identifier (e.g., `domain-purity`)
- `from`: Source layer
- `to`: Target layer(s)
- `type`: One of `cannot`, `must`, `can`, `must_not_circular`
- `severity`: `error`, `warning`, or `info`
- `explanation`: Human-readable text explaining the architectural reasoning

### ES: Tipos de Regla

| Tipo | Semántica | Ejemplo |
|------|-----------|---------|
| `cannot` | La capa origen NO DEBE importar la capa destino | domain CANNOT depend on infrastructure |
| `must` | La capa origen DEBE importar la capa destino (al menos un archivo) | application MUST depend on domain |
| `can` | La capa origen PUEDE importar la capa destino (informativo, nunca falla) | cmd CAN depend on all |
| `must_not_circular` | No hay dependencias circulares entre las capas listadas | domain e infrastructure NO DEBEN ser circulares |

Cada regla incluye:
- `id`: Identificador único (ej: `domain-purity`)
- `from`: Capa origen
- `to`: Capa(s) destino
- `type`: Uno de `cannot`, `must`, `can`, `must_not_circular`
- `severity`: `error`, `warning` o `info`
- `explanation`: Texto legible explicando el razonamiento arquitectónico

---

## EN: Rule Explanations Library

Arx ships with a built-in library of architectural explanations for common rules. When a user writes a rule with an `id` matching a known pattern, Arx uses the built-in explanation unless overridden by the user's `explanation` field.

Built-in explanation catalog:

| Rule ID Pattern | Explanation Summary |
|-----------------|---------------------|
| `domain-*` | Explains dependency inversion, business logic purity, and the stability of the domain layer |
| `application-*` | Explains use case orchestration, why application depends on domain abstractions, not concrete infrastructure |
| `infrastructure-*` | Explains the adapter pattern, why infrastructure implements ports rather than being imported |
| `*-circular` | Explains why circular dependencies create maintenance nightmares and refactoring traps |
| `*-must` | Explains why layers must depend on their designated collaborators (e.g., application without domain is a sign of logic leakage) |

Users can also contribute explanations via plugins (future: `arx explanation add`).

### ES: Librería de Explicaciones de Reglas

Arx incluye una librería integrada de explicaciones arquitectónicas para reglas comunes. Cuando un usuario escribe una regla con un `id` que coincide con un patrón conocido, Arx usa la explicación integrada a menos que el usuario la sobreescriba con su campo `explanation`.

Catálogo de explicaciones integrado:

| Patrón de ID de Regla | Resumen de Explicación |
|----------------------|----------------------|
| `domain-*` | Explica la inversión de dependencias, la pureza de la lógica de negocio y la estabilidad de la capa de dominio |
| `application-*` | Explica la orquestación de casos de uso, por qué application depende de abstracciones del dominio, no de infraestructura concreta |
| `infrastructure-*` | Explica el patrón adaptador, por qué la infraestructura implementa puertos en vez de ser importada |
| `*-circular` | Explica por qué las dependencias circulares crean pesadillas de mantenimiento y trampas de refactorización |
| `*-must` | Explica por qué las capas deben depender de sus colaboradores designados (ej: application sin domain es señal de fuga de lógica) |

Los usuarios también pueden contribuir explicaciones vía plugins (futuro: `arx explanation add`).

---

## EN: Output Formats

### Terminal (default)

Colored, human-readable, educational. Uses ANSI color codes. Each violation shows:
- Violation ID (e.g., `D-01`)
- Rule broken
- File and line
- Why it matters (from the rule's `explanation`)
- How to fix it (from the rule's `explanation` or built-in pattern)

### JSON (`--format json`)

Structured JSON output. Suitable for tooling integration, custom reporters, or further processing.

### SARIF (`--format sarif`)

Static Analysis Results Interchange Format. Integrates with GitHub code scanning, Azure DevOps, and VS Code SARIF Viewer.

### Markdown (`--format md`)

Produces a markdown report suitable for committing to the repository as architecture documentation.

### ES: Formatos de Salida

### Terminal (por defecto)

Coloreado, legible, educativo. Usa códigos de color ANSI. Cada violación muestra:
- ID de violación (ej: `D-01`)
- Regla infringida
- Archivo y línea
- Por qué importa (del campo `explanation` de la regla)
- Cómo corregirlo (del campo `explanation` de la regla o patrón integrado)

### JSON (`--format json`)

Salida JSON estructurada. Adecuada para integración con herramientas, reporters personalizados o procesamiento adicional.

### SARIF (`--format sarif`)

Static Analysis Results Interchange Format. Se integra con GitHub code scanning, Azure DevOps y VS Code SARIF Viewer.

### Markdown (`--format md`)

Produce un informe en markdown adecuado para commitar al repositorio como documentación de arquitectura.

---

## EN: Technical Decisions

### 1. Language: Go

**Decision**: Arx is implemented in Go.
**Rationale**: Single binary distribution (no runtime), fast compilation, excellent CLI ecosystem (cobra, viper), cross-platform, easy `go install` for users. The Go standard library includes `go/ast` for parsing Go source natively.
**Alternatives considered**: Rust (steeper learning curve, slower compilation), TypeScript (requires Node runtime).

### 2. Configuration: YAML

**Decision**: Configuration is defined in `arx.yaml`, version-controlled alongside the code.
**Rationale**: YAML is the de-facto standard for CI configuration (GitHub Actions, GitLab CI). Teams are familiar with it. It supports comments (unlike JSON). It is human-readable and diff-friendly.
**Alternatives considered**: TOML (less common in CI), JSON (no comments), DSL in Go (too complex for MVP).

### 3. Plugin System: Go Interface + Registry

**Decision**: Detectors are Go plugins implementing a `Detector` interface, registered at compile time.
**Rationale**: For the MVP, compile-time registration is simpler and more reliable than dynamic loading. Community contributions are PRs to the repository. Future versions may support dynamic plugins.
**Alternatives considered**: WASM plugins (complex for MVP), Lua scripts (added runtime dependency).

### 4. Import Extraction: AST Where Available, Regex Fallback

**Decision**: Use language AST parsers when available (Go), regex-based extraction as fallback (TypeScript, Python, Java, Rust for MVP).
**Rationale**: Go has `go/ast` in the standard library — it would be wasteful not to use it. For other languages, regex handles 95% of import patterns. Tree-sitter integration can come later for precision.
**Alternatives considered**: Tree-sitter for all languages (adds CGO dependency, complex build), only regex (loss of precision in Go).

### 5. Architecture Output: SARIF

**Decision**: SARIF is a first-class output format alongside terminal and JSON.
**Rationale**: SARIF is supported by GitHub code scanning, Azure DevOps, and VS Code. This gives Arx immediate CI integration value without custom GitHub Actions.
**Alternatives considered**: Custom GitHub Action only (less portable), JUnit XML (java-centric).

### 6. Teaching Over Enforcement

**Decision**: Every rule MUST have an `explanation` field. The default output includes didactic content. The `--ci` flag reduces output for machines but does not remove explanations from the data model.
**Rationale**: Arx's differentiator is education. A tool that only says "violation" is a linter. A tool that says "violation + why it matters + how to fix it" is an architecture mentor.
**Alternatives considered**: terse output only (loses differentiator), educational mode as opt-in (defeats the purpose).

### ES: Decisiones Técnicas

### 1. Lenguaje: Go

**Decisión**: Arx se implementa en Go.
**Justificación**: Distribución como binario único (sin runtime), compilación rápida, ecosistema CLI excelente (cobra, viper), multiplataforma, fácil `go install` para usuarios. La librería estándar de Go incluye `go/ast` para parsear código Go nativamente.
**Alternativas consideradas**: Rust (curva de aprendizaje más pronunciada, compilación más lenta), TypeScript (requiere runtime de Node).

### 2. Configuración: YAML

**Decisión**: La configuración se define en `arx.yaml`, versionada junto con el código.
**Justificación**: YAML es el estándar de facto para configuración de CI (GitHub Actions, GitLab CI). Los equipos lo conocen. Soporta comentarios (a diferencia de JSON). Es legible y amigable para diffs.
**Alternativas consideradas**: TOML (menos común en CI), JSON (sin comentarios), DSL en Go (demasiado complejo para MVP).

### 3. Sistema de Plugins: Interfaz Go + Registro

**Decisión**: Los detectores son plugins de Go que implementan una interfaz `Detector`, registrados en tiempo de compilación.
**Justificación**: Para el MVP, el registro en tiempo de compilación es más simple y fiable que la carga dinámica. Las contribuciones de la comunidad son PRs al repositorio. Versiones futuras pueden soportar plugins dinámicos.
**Alternativas consideradas**: Plugins WASM (complejo para MVP), scripts Lua (añade dependencia de runtime).

### 4. Extracción de Imports: AST Donde Disponible, Regex como Fallback

**Decisión**: Usar parsers AST del lenguaje cuando estén disponibles (Go), extracción basada en regex como fallback (TypeScript, Python, Java, Rust en el MVP).
**Justificación**: Go tiene `go/ast` en la librería estándar — sería un desperdicio no usarlo. Para otros lenguajes, regex maneja el 95% de los patrones de import. La integración con tree-sitter puede venir después para mayor precisión.
**Alternativas consideradas**: Tree-sitter para todos los lenguajes (añade dependencia CGO, compilación compleja), solo regex (pérdida de precisión en Go).

### 5. Salida de Arquitectura: SARIF

**Decisión**: SARIF es un formato de salida de primera clase junto con terminal y JSON.
**Justificación**: SARIF está soportado por GitHub code scanning, Azure DevOps y VS Code. Esto da a Arx valor inmediato de integración con CI sin necesidad de GitHub Actions personalizados.
**Alternativas consideradas**: GitHub Action personalizado solo (menos portable), JUnit XML (centrado en Java).

### 6. Enseñar por Encima de Aplicar

**Decisión**: Cada regla DEBE tener un campo `explanation`. La salida por defecto incluye contenido didáctico. El flag `--ci` reduce la salida para máquinas pero no elimina las explicaciones del modelo de datos.
**Justificación**: El diferenciador de Arx es la educación. Una herramienta que solo dice "violación" es un linter. Una herramienta que dice "violación + por qué importa + cómo corregirlo" es un mentor de arquitectura.
**Alternativas consideradas**: salida concisa solo (pierde diferenciador), modo educativo como opt-in (frustra el propósito).

---

## EN: Roadmap

### v0.1.0 — Foundation

- `arx init` — project detection, language discovery, `arx.yaml` generation
- `arx check` — rule evaluation with terminal output (colored, educational)
- `arx check --ci` — JSON output for CI/CD
- Go detector (AST-based)
- TypeScript detector (regex-based)
- YAML configuration schema
- Built-in explanation library for common architectural rules
- Unit tests, integration tests with fixture projects

### v0.2.0 — Teaching Mode

- `arx explain <violation-id>` — detailed violation breakdown
- Auto-detection of layer conventions (Hexagonal, Clean, DDD folder patterns)
- `arx check --format sarif` — SARIF output for GitHub code scanning
- `arx check --format md` — Markdown report for repo documentation
- Circular dependency detection
- Warning vs. error severity levels

### v0.3.0 — Ecosystem

- Python detector
- Java detector
- `arx diagram` — generate dependency graph (Graphviz DOT output)
- `arx audit` — architectural health report with trend tracking
- Configuration validation (`arx validate`)

### v0.4.0 — Intelligence

- `arx rules suggest` — analyze project structure and suggest architectural rules
- Layer coupling matrix visualization
- GitHub Action for CI integration
- Git diff tracking (only check changed files in PRs)

### ES: Hoja de Ruta

### v0.1.0 — Fundación

- `arx init` — detección de proyecto, descubrimiento de lenguajes, generación de `arx.yaml`
- `arx check` — evaluación de reglas con salida en terminal (coloreada, educativa)
- `arx check --ci` — salida JSON para CI/CD
- Detector Go (basado en AST)
- Detector TypeScript (basado en regex)
- Esquema de configuración YAML
- Librería de explicaciones integrada para reglas arquitectónicas comunes
- Tests unitarios, tests de integración con proyectos de fixture

### v0.2.0 — Modo Enseñanza

- `arx explain <violation-id>` — desglose detallado de violaciones
- Auto-detección de convenciones de capas (patrones de carpetas Hexagonal, Clean, DDD)
- `arx check --format sarif` — salida SARIF para GitHub code scanning
- `arx check --format md` — informe Markdown para documentación del repositorio
- Detección de dependencias circulares
- Niveles de severidad warning vs. error

### v0.3.0 — Ecosistema

- Detector Python
- Detector Java
- `arx diagram` — genera grafo de dependencias (salida Graphviz DOT)
- `arx audit` — informe de salud arquitectónica con seguimiento de tendencias
- Validación de configuración (`arx validate`)

### v0.4.0 — Inteligencia

- `arx rules suggest` — analiza la estructura del proyecto y sugiere reglas arquitectónicas
- Visualización de matriz de acoplamiento entre capas
- GitHub Action para integración con CI
- Seguimiento de diff de Git (verificar solo archivos cambiados en PRs)

---

## EN: Why Go? Deeper Rationale

| Factor | Go | Rust | TypeScript |
|--------|-----|------|------------|
| Distribution | Single binary, `go install` | Single binary, but slower compile | Requires Node.js runtime |
| Install for CI | `go install` in one line | Build from source or download binary | `npm install -g` + Node |
| Cross-compile | `GOOS=linux GOARCH=amd64 go build` | Cross-compile supported but complex | Platform-specific bundles |
| Performance | Fast enough for CLI (ms parsing) | Faster, but overkill for CLI | Acceptable for small projects |
| Ecosystem | Cobra, Viper, Bubbletea, Lip Gloss | Clap, Serde, excellent but smaller | Commander.js, Ink, large ecosystem |
| Go self-parsing | `go/ast` in stdlib | N/A | N/A |
| Learning curve | Moderate | Steep | Low (but TypeScript itself is complex) |
| Installation friction | Zero (single binary) | Low (single binary) | High (needs Node + npm) |

Go is the pragmatic choice: it compiles to a single static binary, has first-class CLI libraries, and can parse its own language natively. The target audience (dev инфраструктура, architects) already has Go installed. A tool that requires `npm install` is friction for CI pipelines.

### ES: ¿Por Qué Go? Justificación en Profundidad

| Factor | Go | Rust | TypeScript |
|--------|-----|------|------------|
| Distribución | Binario único, `go install` | Binario único, pero compilación más lenta | Requiere runtime de Node.js |
| Instalación en CI | `go install` en una línea | Compilar desde fuente o descargar binario | `npm install -g` + Node |
| Compilación cruzada | `GOOS=linux GOARCH=amd64 go build` | Soportada pero compleja | Bundles específicos por plataforma |
| Rendimiento | Suficientemente rápido para CLI (ms de parseo) | Más rápido, pero excesivo para CLI | Aceptable para proyectos pequeños |
| Ecosistema | Cobra, Viper, Bubbletea, Lip Gloss | Clap, Serde, excelente pero más pequeño | Commander.js, Ink, ecosistema grande |
| Autoparseo Go | `go/ast` en stdlib | N/A | N/A |
| Curva de aprendizaje | Moderada | Pronunciada | Baja (pero TypeScript mismo es complejo) |
| Fricción de instalación | Cero (binario único) | Baja (binario único) | Alta (necesita Node + npm) |

Go es la opción pragmática: compila a un binario estático único, tiene librerías CLI de primera clase y puede parsear su propio lenguaje nativamente. El público objetivo (infraestructura, arquitectos) ya tiene Go instalado. Una herramienta que requiere `npm install` genera fricción en pipelines de CI.

---

## EN: License

**Mozilla Public License 2.0 (MPL-2.0)**

Rationale: MPL-2.0 is a weak copyleft license. It allows proprietary projects to use Arx as a CLI tool without license concerns (the CLI is a separate work), but requires any modifications to Arx's source code to be shared. This protects the project from corporate take-without-give while remaining business-friendly. It is the same license used by Firefox, LibreOffice, and many AWS SDKs.

Key properties:
- ✅ Can be used in proprietary projects (the tool is a CLI, not a library)
- ✅ Modifications to Arx source must be shared (prevents free-riding)
- ✅ Not viral like GPL (does not infect the projects it audits)
- ✅ Business-friendly for CI/CD integration

### ES: Licencia

**Mozilla Public License 2.0 (MPL-2.0)**

Justificación: MPL-2.0 es una licencia copyleft débil. Permite que proyectos propietarios usen Arx como herramienta CLI sin preocupaciones de licencia (el CLI es una obra separada), pero requiere que cualquier modificación al código fuente de Arx se comparta. Esto protege al proyecto de la apropiación sin reciprocity mientras sigue siendo amigable para negocios. Es la misma licencia usada por Firefox, LibreOffice y muchos SDKs de AWS.

Propiedades clave:
- ✅ Puede usarse en proyectos propietarios (la herramienta es un CLI, no una librería)
- ✅ Las modificaciones al código fuente de Arx deben compartirse (previene aprovechamiento sin contribución)
- ✅ No es viral como GPL (no infecta los proyectos que audita)
- ✅ Amigable para negocios e integración en CI/CD

---

## EN: Naming & Branding

**Name**: Arx (pronounced /ɑːrks/, like "arks")

**Etymology**: Latin *arx* — citadel, fortress, stronghold. The highest point of a Roman city, the architectural anchor. Also an abbreviation of "architecture".

**Logo concept**: A simplified Roman citadel silhouette — two towers connected by a wall, forming a gate (the "Gateway to good architecture"). Minimal, geometric, works at 16x16px.

**Color palette**: Slate blue (#4A5568) + amber (#F59E0B). Professional but approachable.

**Tagline**: *"Architecture you can trust."* / *"Arquitectura en la que puedes confiar."*

### ES: Nombre y Marca

**Nombre**: Arx (se pronuncia /ɑːrks/, como "arks")

**Etimología**: Latín *arx* — ciudadela, fortaleza, baluarte. El punto más alto de una ciudad romana, el ancla arquitectónica. También abreviatura de "architecture".

**Concepto de logo**: Silueta simplificada de una ciudadela romana — dos torres conectadas por un muro, formando una puerta (la "Puerta a la buena arquitectura"). Mínimo, geométrico, funciona a 16x16px.

**Paleta de colores**: Azul pizarra (#4A5568) + ámbar (#F59E0B). Profesional pero cercano.

**Lema**: *"Architecture you can trust."* / *"Arquitectura en la que puedes confiar."*

---

## EN: Differentiation from Existing Tools

| Tool | Language | Scope | Teaching | Cross-language | Price |
|------|----------|-------|----------|----------------|-------|
| ArchUnit | Java only | Architecture rules | No | No | Free |
| Deptrac | PHP only | Dependency layers | No | No | Free |
| NetArchTest | .NET only | Architecture rules | No | No | Free |
| SonarQube | All (via plugins) | Code quality + some architecture | Minimal | Partial | Free/Paid |
| Checkstyle | Java only | Code style | No | No | Free |
| N Depend | .NET only | Architecture rules | No | No | Paid |
| **Arx** | **Go, TypeScript, Python, Java, Rust (extensible)** | **Architecture enforcement** | **Yes — core feature** | **Yes — by design** | **Free (MPL-2.0)** |

### ES: Diferenciación Frente a Herramientas Existentes

| Herramienta | Lenguaje | Alcance | Enseñanza | Multilenguaje | Precio |
|-------------|----------|---------|-----------|---------------|--------|
| ArchUnit | Solo Java | Reglas de arquitectura | No | No | Gratuito |
| Deptrac | Solo PHP | Capas de dependencia | No | No | Gratuito |
| NetArchTest | Solo .NET | Reglas de arquitectura | No | No | Gratuito |
| SonarQube | Todos (vía plugins) | Calidad de código + algo de arquitectura | Mínimo | Parcial | Gratuito/De pago |
| Checkstyle | Solo Java | Estilo de código | No | No | Gratuito |
| N Depend | Solo .NET | Reglas de arquitectura | No | No | De pago |
| **Arx** | **Go, TypeScript, Python, Java, Rust (extensible)** | **Aplicación de arquitectura** | **Sí — característica central** | **Sí — por diseño** | **Gratuito (MPL-2.0)** |

---

## EN: Success Metrics

| Metric | Target (v0.1.0) | Target (v0.4.0) |
|--------|------------------|------------------|
| GitHub stars | 500 | 5,000 |
| Languages supported | 2 (Go, TypeScript) | 5 (Go, TS, Python, Java, Rust) |
| Built-in rule explanations | 15 patterns | 40 patterns |
| CI integrations | GitHub Actions (via SARIF) | GitHub Actions + GitLab CI |
| Contributors | 3-5 core | 20+ |
| Downloads (Go binaries) | 1,000/month | 10,000/month |
| Time to first check on new project | < 30 seconds | < 10 seconds |

### ES: Métricas de Éxito

| Métrica | Objetivo (v0.1.0) | Objetivo (v0.4.0) |
|---------|-------------------|-------------------|
| Estrellas en GitHub | 500 | 5.000 |
| Lenguajes soportados | 2 (Go, TypeScript) | 5 (Go, TS, Python, Java, Rust) |
| Explicaciones de reglas integradas | 15 patrones | 40 patrones |
| Integraciones CI | GitHub Actions (vía SARIF) | GitHub Actions + GitLab CI |
| Contribuidores | 3-5 núcleo | 20+ |
| Descargas (binarios Go) | 1.000/mes | 10.000/mes |
| Tiempo hasta primer check en proyecto nuevo | < 30 segundos | < 10 segundos |

---

## EN: Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Regex-based import parsing misses edge cases (TypeScript barrel files, Python relative imports) | High | Medium | Start with common patterns, add tree-sitter for precision in v0.3. Test coverage for edge cases. |
| Community doesn't contribute detectors for new languages | Medium | High | Ship high-quality detectors for Go and TypeScript as examples. Write a "Writing a Detector" guide. Make the interface simple. |
| Users find configuration too complex | Medium | Medium | `arx init` generates sensible defaults. Provide presets for common architectures (Clean, Hexagonal, DDD). |
| Rule explanations feel patronizing to experienced devs | Medium | Low | `explanation` field is optional in config. `--ci` mode suppresses didactic output. Severity levels let users choose error vs. warning. |
| Maintenance burden for solo developer | High | High | Strict scope control. v0.1.0 is Go + TypeScript only. Community governance from day one. |

### ES: Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|-------------|---------|------------|
| El parseo de imports basado en regex pierde casos límite (barrel files en TypeScript, imports relativos en Python) | Alta | Media | Empezar con patrones comunes, añadir tree-sitter para precisión en v0.3. Cobertura de tests para casos límite. |
| La comunidad no contribuye detectores para nuevos lenguajes | Media | Alta | Incluir detectores de alta calidad para Go y TypeScript como ejemplos. Escribir una guía "Escribir un Detector". Mantener la interfaz simple. |
| Los usuarios encuentran la configuración demasiado compleja | Media | Media | `arx init` genera valores por defecto razonables. Proporcionar presets para arquitecturas comunes (Clean, Hexagonal, DDD). |
| Las explicaciones de reglas resultan condescendientes para desarrolladores experimentados | Media | Baja | El campo `explanation` es opcional en la configuración. El modo `--ci` suprime la salida didáctica. Los niveles de severidad permiten elegir entre error y warning. |
| Carga de mantenimiento para un único desarrollador | Alta | Alta | Control estricto de alcance. v0.1.0 es solo Go + TypeScript. Gobernanza comunitaria desde el primer día. |

---

*This document defines Arx in full. It is the single source of truth for the project's scope, architecture, and roadmap. All SDD artifacts will derive from here.*

*Este documento define Arx en su totalidad. Es la fuente única de verdad para el alcance, arquitectura y hoja de ruta del proyecto. Todos los artefactos SDD se derivarán de aquí.*