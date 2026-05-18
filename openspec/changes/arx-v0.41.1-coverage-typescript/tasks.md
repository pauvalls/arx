# Tasks: TypeScript coverage to >50%

## Phase 1: Core tests
1. Test Name() returns "typescript"
2. Test Detect() with tsconfig.json, package.json, neither
3. Test loadTsConfig() with valid/invalid config
4. Test ExtractImports() full pipeline (temp dir with TS files)

## Phase 2: Edge cases
5. Test resolveAlias with various tsconfig compilerOptions.paths
6. Test resolveLayer with local imports, relative imports, node_modules
7. Test ExtractImports cancellation
8. Test ExtractImports with empty directory
