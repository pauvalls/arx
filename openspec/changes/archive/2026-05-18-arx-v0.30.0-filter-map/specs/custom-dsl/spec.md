# Delta Spec: filter()/map() for Custom DSL

## Req 1: filter(deps, predicate)

- `filter(deps(from, to), "field op value")` → ValueDeps
- Predicate format: `field op value` (space-separated)
- Supported fields: SourceFile, SourceLine, ImportPath, ResolvedLayer
- String fields (SourceFile, ImportPath, ResolvedLayer): == and != only
- Numeric field (SourceLine): ==, !=, >, <, >=, <=
- Invalid predicate → eval error
- Unknown field name → eval error

## Req 2: map(deps, field)

- `map(deps(from, to), "field")` → ValueList
- Supported fields: SourceFile, SourceLine, ImportPath, ResolvedLayer
- SourceLine extracted as string representation
- Unknown field → eval error

## Req 3: ValueList type

- ValueKindList stores `[]string`
- `count(ValueList)` returns `len(list)`
- `==`/`!=` between ValueLists compares by `len()` (via AsInt)
- No other comparison operators supported between lists
