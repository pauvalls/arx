package domain

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// ─── Tokenizer ───────────────────────────────────────────────────────────────

type TokenType int

const (
	TokenIdent TokenType = iota
	TokenNumber
	TokenGT
	TokenLT
	TokenGE
	TokenLE
	TokenEQ
	TokenNE
	TokenAnd
	TokenOr
	TokenNot
	TokenLParen
	TokenRParen
	TokenComma
	TokenString
	TokenEOF
)

var tokenNames = map[TokenType]string{
	TokenIdent:   "IDENT",
	TokenNumber:  "NUMBER",
	TokenGT:      ">",
	TokenLT:      "<",
	TokenGE:      ">=",
	TokenLE:      "<=",
	TokenEQ:      "==",
	TokenNE:      "!=",
	TokenAnd:     "&&",
	TokenOr:      "||",
	TokenNot:     "!",
	TokenLParen:  "(",
	TokenRParen:  ")",
	TokenComma:    ",",
	TokenString:   "STRING",
	TokenEOF:     "EOF",
}

func (t TokenType) String() string {
	if n, ok := tokenNames[t]; ok {
		return n
	}
	return fmt.Sprintf("Token(%d)", t)
}

type Token struct {
	Type  TokenType
	Value string
	Pos   int // byte position in input
}

func tokenize(input string) ([]Token, error) {
	var tokens []Token
	runes := []rune(input)
	i := 0

	for i < len(runes) {
		ch := runes[i]
		pos := i

		// Skip whitespace
		if unicode.IsSpace(ch) {
			i++
			continue
		}

		// Two-char operators
		if i+1 < len(runes) {
			two := string(runes[i : i+2])
			switch two {
			case ">=":
				tokens = append(tokens, Token{Type: TokenGE, Pos: pos})
				i += 2
				continue
			case "<=":
				tokens = append(tokens, Token{Type: TokenLE, Pos: pos})
				i += 2
				continue
			case "==":
				tokens = append(tokens, Token{Type: TokenEQ, Pos: pos})
				i += 2
				continue
			case "!=":
				tokens = append(tokens, Token{Type: TokenNE, Pos: pos})
				i += 2
				continue
			case "&&":
				tokens = append(tokens, Token{Type: TokenAnd, Pos: pos})
				i += 2
				continue
			case "||":
				tokens = append(tokens, Token{Type: TokenOr, Pos: pos})
				i += 2
				continue
			}
		}

		// Single-char tokens
		switch ch {
		case '>':
			tokens = append(tokens, Token{Type: TokenGT, Pos: pos})
			i++
			continue
		case '<':
			tokens = append(tokens, Token{Type: TokenLT, Pos: pos})
			i++
			continue
		case '!':
			tokens = append(tokens, Token{Type: TokenNot, Pos: pos})
			i++
			continue
		case '(':
			tokens = append(tokens, Token{Type: TokenLParen, Pos: pos})
			i++
			continue
		case ')':
			tokens = append(tokens, Token{Type: TokenRParen, Pos: pos})
			i++
			continue
		case ',':
			tokens = append(tokens, Token{Type: TokenComma, Pos: pos})
			i++
			continue
		}

		// Numbers
		if unicode.IsDigit(ch) {
			start := i
			for i < len(runes) && unicode.IsDigit(runes[i]) {
				i++
			}
			tokens = append(tokens, Token{Type: TokenNumber, Value: string(runes[start:i]), Pos: pos})
			continue
		}

		// Identifiers (and keywords)
		if unicode.IsLetter(ch) || ch == '_' {
			start := i
			for i < len(runes) && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '_') {
				i++
			}
			val := string(runes[start:i])
			tokens = append(tokens, Token{Type: TokenIdent, Value: val, Pos: pos})
			continue
		}

		// String literals (double-quoted)
		if ch == '"' {
			i++ // skip opening quote
			start := i
			for i < len(runes) && runes[i] != '"' {
				i++
			}
			if i >= len(runes) {
				return nil, fmt.Errorf("unterminated string literal starting at position %d", pos)
			}
			val := string(runes[start:i])
			i++ // skip closing quote
			tokens = append(tokens, Token{Type: TokenString, Value: val, Pos: pos})
			continue
		}

		return nil, fmt.Errorf("unexpected character %q at position %d", ch, pos)
	}

	tokens = append(tokens, Token{Type: TokenEOF, Pos: len(runes)})
	return tokens, nil
}

// ─── AST ─────────────────────────────────────────────────────────────────────

type Expr interface {
	Eval(ctx EvalContext) (Value, error)
}

type EvalContext struct {
	Deps          []Dependency
	Layers        []Layer
	Violations    []Violation
	LayerFiles    map[string][]string
	UserFunctions map[string]Expr
}

type ValueKind int

const (
	ValueInt ValueKind = iota
	ValueBool
	ValueDeps
	ValueKindList
)

type Value struct {
	Kind ValueKind
	Int  int
	Bool bool
	Deps []Dependency
	List []string
}

// AsBool converts a Value to bool for logical operations.
// Int > 0 is true; Deps with len > 0 is true; List with len > 0 is true.
func (v Value) AsBool() bool {
	switch v.Kind {
	case ValueBool:
		return v.Bool
	case ValueInt:
		return v.Int > 0
	case ValueDeps:
		return len(v.Deps) > 0
	case ValueKindList:
		return len(v.List) > 0
	}
	return false
}

// AsInt converts a Value to int.
// Bool true is 1, false is 0; Deps len is the int value; List len is the int value.
func (v Value) AsInt() int {
	switch v.Kind {
	case ValueInt:
		return v.Int
	case ValueBool:
		if v.Bool {
			return 1
		}
		return 0
	case ValueDeps:
		return len(v.Deps)
	case ValueKindList:
		return len(v.List)
	}
	return 0
}

type ComparisonExpr struct {
	Left  Expr
	Op    TokenType
	Right Expr
}

func (e *ComparisonExpr) Eval(ctx EvalContext) (Value, error) {
	left, err := e.Left.Eval(ctx)
	if err != nil {
		return Value{}, err
	}
	right, err := e.Right.Eval(ctx)
	if err != nil {
		return Value{}, err
	}

	result, err := compareValues(left, e.Op, right)
	if err != nil {
		return Value{}, err
	}
	return Value{Kind: ValueBool, Bool: result}, nil
}

type BinaryExpr struct {
	Left  Expr
	Op    TokenType
	Right Expr
}

func (e *BinaryExpr) Eval(ctx EvalContext) (Value, error) {
	left, err := e.Left.Eval(ctx)
	if err != nil {
		return Value{}, err
	}

	switch e.Op {
	case TokenAnd:
		if !left.AsBool() {
			return Value{Kind: ValueBool, Bool: false}, nil
		}
		right, err := e.Right.Eval(ctx)
		if err != nil {
			return Value{}, err
		}
		return Value{Kind: ValueBool, Bool: right.AsBool()}, nil
	case TokenOr:
		if left.AsBool() {
			return Value{Kind: ValueBool, Bool: true}, nil
		}
		right, err := e.Right.Eval(ctx)
		if err != nil {
			return Value{}, err
		}
		return Value{Kind: ValueBool, Bool: right.AsBool()}, nil
	default:
		return Value{}, fmt.Errorf("unknown binary operator %v", e.Op)
	}
}

type UnaryExpr struct {
	Op    TokenType
	Right Expr
}

func (e *UnaryExpr) Eval(ctx EvalContext) (Value, error) {
	right, err := e.Right.Eval(ctx)
	if err != nil {
		return Value{}, err
	}
	if e.Op == TokenNot {
		return Value{Kind: ValueBool, Bool: !right.AsBool()}, nil
	}
	return Value{}, fmt.Errorf("unknown unary operator %v", e.Op)
}

type FuncCallExpr struct {
	Name string
	Args []Expr
}

func (e *FuncCallExpr) Eval(ctx EvalContext) (Value, error) {
	// Check user-defined functions first, then builtins.
	if ctx.UserFunctions != nil {
		if userFn, ok := ctx.UserFunctions[e.Name]; ok {
			return userFn.Eval(ctx)
		}
	}
	fn, ok := builtins[e.Name]
	if !ok {
		return Value{}, fmt.Errorf("unknown function %q", e.Name)
	}
	return fn(e.Args, ctx)
}

type NumberLiteral struct {
	Value int
}

func (e *NumberLiteral) Eval(_ EvalContext) (Value, error) {
	return Value{Kind: ValueInt, Int: e.Value}, nil
}

type StringLiteral struct {
	Value string
}

func (e *StringLiteral) Eval(_ EvalContext) (Value, error) {
	return Value{Kind: ValueInt, Int: 0}, nil // strings are opaque; used as args
}

// ─── CheckExpr ───────────────────────────────────────────────────────────────

// CheckExpr holds a check expression from YAML config, supporting
// both single string and multi-line list forms.
type CheckExpr struct {
	Raw   string
	Expr  Expr
	items []string
}

// UnmarshalYAML accepts either a single string or a list of strings.
// For a single string, Raw is set to the input and items is empty.
// For a list, Raw is set to the " && "-joined form and items stores the originals.
func (c *CheckExpr) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err == nil {
		c.Raw = s
		c.items = nil
		return nil
	}
	var items []string
	if err := unmarshal(&items); err != nil {
		return fmt.Errorf("check expression must be a string or list of strings")
	}
	if len(items) == 0 {
		return fmt.Errorf("check expression list must not be empty")
	}
	c.Raw = strings.Join(items, " && ")
	c.items = items
	return nil
}

// Validate compiles all expressions. For list form, it builds an AND tree
// from all items. The first parse error fails the entire validation.
func (c *CheckExpr) Validate() error {
	if c.Raw == "" {
		c.Expr = nil
		return nil
	}
	if len(c.items) > 0 {
		var exprs []Expr
		for _, item := range c.items {
			expr, err := Parse(item)
			if err != nil {
				return fmt.Errorf("invalid check expression item %q: %w", item, err)
			}
			exprs = append(exprs, expr)
		}
		c.Expr = buildAndTree(exprs)
	} else {
		expr, err := Parse(c.Raw)
		if err != nil {
			return fmt.Errorf("invalid check expression: %w", err)
		}
		c.Expr = expr
	}
	return nil
}

// MarshalJSON serializes CheckExpr as a plain string for hashing.
func (c CheckExpr) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Raw)
}

// MarshalYAML serializes CheckExpr as a plain string.
func (c CheckExpr) MarshalYAML() (interface{}, error) {
	return c.Raw, nil
}

// String returns the raw expression text.
func (c CheckExpr) String() string {
	return c.Raw
}

// buildAndTree chains expressions into a left-associative AND tree.
func buildAndTree(exprs []Expr) Expr {
	if len(exprs) == 0 {
		return nil
	}
	result := exprs[0]
	for i := 1; i < len(exprs); i++ {
		result = &BinaryExpr{Left: result, Op: TokenAnd, Right: exprs[i]}
	}
	return result
}

func compareValues(left Value, op TokenType, right Value) (bool, error) {
	// For == and !=, support string comparison if either side came from a string literal
	if op == TokenEQ || op == TokenNE {
		// If one side is a string literal (has no meaningful numeric value),
		// we only support equality with another string.
		// In practice, strings are used as function arguments, not in comparisons.
		// So numeric comparison is the default.
	}

	// Numeric comparison (default)
	li := left.AsInt()
	ri := right.AsInt()

	switch op {
	case TokenGT:
		return li > ri, nil
	case TokenLT:
		return li < ri, nil
	case TokenGE:
		return li >= ri, nil
	case TokenLE:
		return li <= ri, nil
	case TokenEQ:
		return li == ri, nil
	case TokenNE:
		return li != ri, nil
	default:
		return false, fmt.Errorf("unknown comparison operator %v", op)
	}
}

// ─── Built-in Functions ──────────────────────────────────────────────────────

var builtins = map[string]func(args []Expr, ctx EvalContext) (Value, error){
	"count":        builtinCount,
	"deps":         builtinDeps,
	"layers":       builtinLayers,
	"has_circular": builtinHasCircular,
	"files":        builtinFiles,
	"ratio":        builtinRatio,
	"violations":   builtinViolations,
	"threshold":    builtinThreshold,
	"all":          builtinAll,
	"any":          builtinAny,
	"filter":       builtinFilter,
	"map":          builtinMap,
}

func builtinCount(args []Expr, ctx EvalContext) (Value, error) {
	if len(args) != 1 {
		return Value{}, fmt.Errorf("count() expects exactly 1 argument, got %d", len(args))
	}
	v, err := args[0].Eval(ctx)
	if err != nil {
		return Value{}, err
	}
	return Value{Kind: ValueInt, Int: v.AsInt()}, nil
}

func builtinDeps(args []Expr, ctx EvalContext) (Value, error) {
	if len(args) != 2 {
		return Value{}, fmt.Errorf("deps() expects exactly 2 arguments, got %d", len(args))
	}
	fromVal, err := args[0].Eval(ctx)
	if err != nil {
		return Value{}, err
	}
	toVal, err := args[1].Eval(ctx)
	if err != nil {
		return Value{}, err
	}

	fromLayer := extractStringArg(args[0], fromVal)
	toLayer := extractStringArg(args[1], toVal)

	layerMap := make(map[string]*Layer)
	for i := range ctx.Layers {
		layerMap[ctx.Layers[i].Name] = &ctx.Layers[i]
	}

	var result []Dependency
	for _, dep := range ctx.Deps {
		srcLayer := resolveLayer(dep.SourceFile, layerMap)
		if srcLayer == fromLayer && dep.ResolvedLayer == toLayer {
			result = append(result, dep)
		}
	}

	return Value{Kind: ValueDeps, Deps: result}, nil
}

// extractStringArg gets the string value from an argument expression.
// For StringLiteral it returns the literal value; for other expressions
// it falls back to the string representation of the evaluated value.
func extractStringArg(expr Expr, val Value) string {
	if s, ok := expr.(*StringLiteral); ok {
		return s.Value
	}
	// Fallback: identifiers evaluate to zero-value Value, so we need
	// to extract the identifier name directly.
	return ""
}

func builtinLayers(args []Expr, ctx EvalContext) (Value, error) {
	if len(args) != 0 {
		return Value{}, fmt.Errorf("layers() expects no arguments, got %d", len(args))
	}
	return Value{Kind: ValueInt, Int: len(ctx.Layers)}, nil
}

func builtinHasCircular(args []Expr, ctx EvalContext) (Value, error) {
	if len(args) != 0 {
		return Value{}, fmt.Errorf("has_circular() expects no arguments, got %d", len(args))
	}
	cycles := DetectCircularDependencies(ctx.Deps, ctx.Layers)
	return Value{Kind: ValueBool, Bool: len(cycles) > 0}, nil
}

func builtinFiles(args []Expr, ctx EvalContext) (Value, error) {
	if len(args) != 1 {
		return Value{}, fmt.Errorf("files() expects exactly 1 argument, got %d", len(args))
	}
	layerName, err := resolveStringArg(args[0], ctx)
	if err != nil {
		return Value{}, err
	}
	count := len(ctx.LayerFiles[layerName])
	return Value{Kind: ValueInt, Int: count}, nil
}

func builtinRatio(args []Expr, ctx EvalContext) (Value, error) {
	if len(args) != 2 {
		return Value{}, fmt.Errorf("ratio() expects exactly 2 arguments, got %d", len(args))
	}
	v1, err := args[0].Eval(ctx)
	if err != nil {
		return Value{}, err
	}
	v2, err := args[1].Eval(ctx)
	if err != nil {
		return Value{}, err
	}
	count := v1.AsInt()
	total := v2.AsInt()
	if total == 0 {
		return Value{Kind: ValueInt, Int: 0}, nil
	}
	return Value{Kind: ValueInt, Int: count / total}, nil
}

func builtinViolations(args []Expr, ctx EvalContext) (Value, error) {
	if len(args) != 1 {
		return Value{}, fmt.Errorf("violations() expects exactly 1 argument, got %d", len(args))
	}
	ruleID, err := resolveStringArg(args[0], ctx)
	if err != nil {
		return Value{}, err
	}
	count := 0
	for _, v := range ctx.Violations {
		if v.RuleID == ruleID {
			count++
		}
	}
	return Value{Kind: ValueInt, Int: count}, nil
}

func builtinThreshold(args []Expr, ctx EvalContext) (Value, error) {
	if len(args) != 3 {
		return Value{}, fmt.Errorf("threshold() expects exactly 3 arguments, got %d", len(args))
	}
	v, err := args[0].Eval(ctx)
	if err != nil {
		return Value{}, err
	}
	minVal, err := args[1].Eval(ctx)
	if err != nil {
		return Value{}, err
	}
	maxVal, err := args[2].Eval(ctx)
	if err != nil {
		return Value{}, err
	}
	val := v.AsInt()
	min := minVal.AsInt()
	max := maxVal.AsInt()
	return Value{Kind: ValueBool, Bool: val >= min && val <= max}, nil
}

func builtinAll(args []Expr, ctx EvalContext) (Value, error) {
	if len(args) != 1 {
		return Value{}, fmt.Errorf("all() expects exactly 1 argument, got %d", len(args))
	}
	v, err := args[0].Eval(ctx)
	if err != nil {
		return Value{}, err
	}
	if v.Kind != ValueDeps {
		return Value{}, fmt.Errorf("all() expects a deps() call as argument")
	}
	return Value{Kind: ValueBool, Bool: len(v.Deps) > 0}, nil
}

func builtinAny(args []Expr, ctx EvalContext) (Value, error) {
	if len(args) != 1 {
		return Value{}, fmt.Errorf("any() expects exactly 1 argument, got %d", len(args))
	}
	v, err := args[0].Eval(ctx)
	if err != nil {
		return Value{}, err
	}
	if v.Kind != ValueDeps {
		return Value{}, fmt.Errorf("any() expects a deps() call as argument")
	}
	return Value{Kind: ValueBool, Bool: len(v.Deps) > 0}, nil
}

// ─── Filter/Map Helpers ──────────────────────────────────────────────────────

// evalPredicate evaluates a simple "field op value" predicate against a Dependency.
// Predicate format: exactly 3 space-separated tokens: field, operator, value.
// Supported fields: SourceFile, ImportPath, ResolvedLayer (==/!= only), SourceLine (all ops).
func evalPredicate(dep Dependency, predicate string) (bool, error) {
	parts := strings.Fields(predicate)
	if len(parts) != 3 {
		return false, fmt.Errorf("invalid predicate %q: expected exactly 3 tokens (field op value)", predicate)
	}
	field, op, value := parts[0], parts[1], parts[2]

	switch field {
	case "SourceFile":
		if op == "==" {
			return dep.SourceFile == value, nil
		}
		if op == "!=" {
			return dep.SourceFile != value, nil
		}
		return false, fmt.Errorf("invalid operator %q for string field %q (allowed: ==, !=)", op, field)
	case "ImportPath":
		if op == "==" {
			return dep.ImportPath == value, nil
		}
		if op == "!=" {
			return dep.ImportPath != value, nil
		}
		return false, fmt.Errorf("invalid operator %q for string field %q (allowed: ==, !=)", op, field)
	case "ResolvedLayer":
		if op == "==" {
			return dep.ResolvedLayer == value, nil
		}
		if op == "!=" {
			return dep.ResolvedLayer != value, nil
		}
		return false, fmt.Errorf("invalid operator %q for string field %q (allowed: ==, !=)", op, field)
	case "SourceLine":
		target, err := strconv.Atoi(value)
		if err != nil {
			return false, fmt.Errorf("invalid number %q for SourceLine comparison", value)
		}
		switch op {
		case "==":
			return dep.SourceLine == target, nil
		case "!=":
			return dep.SourceLine != target, nil
		case ">":
			return dep.SourceLine > target, nil
		case "<":
			return dep.SourceLine < target, nil
		case ">=":
			return dep.SourceLine >= target, nil
		case "<=":
			return dep.SourceLine <= target, nil
		default:
			return false, fmt.Errorf("invalid operator %q for field %q", op, field)
		}
	default:
		return false, fmt.Errorf("unknown field %q in predicate", field)
	}
}

// depFieldByName extracts a string representation of a field from a Dependency by name.
// Supports the same field names as evalPredicate.
func depFieldByName(dep Dependency, field string) (string, error) {
	switch field {
	case "SourceFile":
		return dep.SourceFile, nil
	case "ImportPath":
		return dep.ImportPath, nil
	case "ResolvedLayer":
		return dep.ResolvedLayer, nil
	case "SourceLine":
		return strconv.Itoa(dep.SourceLine), nil
	default:
		return "", fmt.Errorf("unknown field %q", field)
	}
}

// builtinFilter filters a ValueDeps by a predicate string.
// Usage: filter(deps(from, to), "field op value")
func builtinFilter(args []Expr, ctx EvalContext) (Value, error) {
	if len(args) != 2 {
		return Value{}, fmt.Errorf("filter() expects exactly 2 arguments, got %d", len(args))
	}

	depsVal, err := args[0].Eval(ctx)
	if err != nil {
		return Value{}, err
	}
	if depsVal.Kind != ValueDeps {
		return Value{}, fmt.Errorf("filter() first argument must evaluate to deps, got %v", depsVal.Kind)
	}

	predicate, err := resolveStringArg(args[1], ctx)
	if err != nil {
		return Value{}, fmt.Errorf("filter() second argument must be a string literal (predicate)")
	}

	var matched []Dependency
	for _, dep := range depsVal.Deps {
		ok, err := evalPredicate(dep, predicate)
		if err != nil {
			return Value{}, fmt.Errorf("filter() predicate error: %w", err)
		}
		if ok {
			matched = append(matched, dep)
		}
	}

	return Value{Kind: ValueDeps, Deps: matched}, nil
}

// builtinMap extracts a field from each dep in a ValueDeps into a ValueList.
// Usage: map(deps(from, to), "FieldName")
func builtinMap(args []Expr, ctx EvalContext) (Value, error) {
	if len(args) != 2 {
		return Value{}, fmt.Errorf("map() expects exactly 2 arguments, got %d", len(args))
	}

	depsVal, err := args[0].Eval(ctx)
	if err != nil {
		return Value{}, err
	}
	if depsVal.Kind != ValueDeps {
		return Value{}, fmt.Errorf("map() first argument must evaluate to deps, got %v", depsVal.Kind)
	}

	fieldName, err := resolveStringArg(args[1], ctx)
	if err != nil {
		return Value{}, fmt.Errorf("map() second argument must be a string literal (field name)")
	}

	var result []string
	for _, dep := range depsVal.Deps {
		val, err := depFieldByName(dep, fieldName)
		if err != nil {
			return Value{}, fmt.Errorf("map() field error: %w", err)
		}
		result = append(result, val)
	}

	return Value{Kind: ValueKindList, List: result}, nil
}

// ─── Parser ──────────────────────────────────────────────────────────────────

type parser struct {
	tokens []Token
	pos    int
}

func (p *parser) current() Token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return Token{Type: TokenEOF}
}

func (p *parser) advance() Token {
	tok := p.current()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *parser) expect(tt TokenType) (Token, error) {
	tok := p.current()
	if tok.Type != tt {
		return tok, fmt.Errorf("expected %v, got %v at position %d", tt, tok.Type, tok.Pos)
	}
	p.advance()
	return tok, nil
}

// Parse parses an expression string into an AST.
func Parse(input string) (Expr, error) {
	tokens, err := tokenize(input)
	if err != nil {
		return nil, err
	}
	p := &parser{tokens: tokens}
	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if p.current().Type != TokenEOF {
		return nil, fmt.Errorf("unexpected token %v at position %d", p.current().Type, p.current().Pos)
	}
	return expr, nil
}

// Grammar:
// expr       → orExpr
// orExpr     → andExpr (("||") andExpr)*
// andExpr    → notExpr (("&&") notExpr)*
// notExpr    → "!" notExpr | comparison
// comparison → primary ((">" | "<" | ">=" | "<=" | "==" | "!=") primary)?
// primary    → call | number | ident | "(" expr ")"
// call       → ident "(" (expr ("," expr)*)? ")"

func (p *parser) parseExpr() (Expr, error) {
	return p.parseOrExpr()
}

func (p *parser) parseOrExpr() (Expr, error) {
	left, err := p.parseAndExpr()
	if err != nil {
		return nil, err
	}

	for p.current().Type == TokenOr {
		p.advance()
		right, err := p.parseAndExpr()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Left: left, Op: TokenOr, Right: right}
	}

	return left, nil
}

func (p *parser) parseAndExpr() (Expr, error) {
	left, err := p.parseNotExpr()
	if err != nil {
		return nil, err
	}

	for p.current().Type == TokenAnd {
		p.advance()
		right, err := p.parseNotExpr()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Left: left, Op: TokenAnd, Right: right}
	}

	return left, nil
}

func (p *parser) parseNotExpr() (Expr, error) {
	if p.current().Type == TokenNot {
		p.advance()
		right, err := p.parseNotExpr()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Op: TokenNot, Right: right}, nil
	}
	return p.parseComparison()
}

func (p *parser) parseComparison() (Expr, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	if isComparisonOp(p.current().Type) {
		op := p.current().Type
		p.advance()
		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		left = &ComparisonExpr{Left: left, Op: op, Right: right}
	}

	return left, nil
}

func isComparisonOp(t TokenType) bool {
	switch t {
	case TokenGT, TokenLT, TokenGE, TokenLE, TokenEQ, TokenNE:
		return true
	}
	return false
}

func (p *parser) parsePrimary() (Expr, error) {
	tok := p.current()

	switch tok.Type {
	case TokenNumber:
		p.advance()
		val, err := strconv.Atoi(tok.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid number %q at position %d", tok.Value, tok.Pos)
		}
		return &NumberLiteral{Value: val}, nil

	case TokenIdent:
		p.advance()
		// Check if this is a function call
		if p.current().Type == TokenLParen {
			return p.parseCall(tok.Value)
		}
		// Otherwise it's a string literal (layer name or identifier)
		return &StringLiteral{Value: tok.Value}, nil

	case TokenString:
		p.advance()
		return &StringLiteral{Value: tok.Value}, nil

	case TokenLParen:
		p.advance()
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenRParen); err != nil {
			return nil, err
		}
		return expr, nil

	default:
		return nil, fmt.Errorf("unexpected token %v at position %d", tok.Type, tok.Pos)
	}
}

func (p *parser) parseCall(name string) (Expr, error) {
	if _, err := p.expect(TokenLParen); err != nil {
		return nil, err
	}

	var args []Expr
	if p.current().Type != TokenRParen {
		for {
			arg, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
			if p.current().Type == TokenComma {
				p.advance()
				continue
			}
			break
		}
	}

	if _, err := p.expect(TokenRParen); err != nil {
		return nil, err
	}

	return &FuncCallExpr{Name: name, Args: args}, nil
}

// ─── Helpers for string arguments ────────────────────────────────────────────

// resolveStringArg evaluates an expression and extracts its string value.
// Used when function arguments must be layer names.
func resolveStringArg(e Expr, ctx EvalContext) (string, error) {
	if s, ok := e.(*StringLiteral); ok {
		return s.Value, nil
	}
	// If it's a function call or other expression, we can't extract a static string.
	return "", fmt.Errorf("expected string literal argument, got %T", e)
}

// ─── User Function Helpers ────────────────────────────────────────────────────

// builtinNames contains all registered builtin function names for shadow checking.
var builtinNames map[string]bool

func init() {
	builtinNames = make(map[string]bool, len(builtins))
	for name := range builtins {
		builtinNames[name] = true
	}
}

// IsBuiltinName returns true if name is a builtin function identifier.
func IsBuiltinName(name string) bool {
	return builtinNames[name]
}

// IsValidIdentifier checks if a name is a valid user function identifier.
func IsValidIdentifier(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if !unicode.IsLetter(r) && r != '_' {
				return false
			}
		} else {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
				return false
			}
		}
	}
	return true
}

// CollectFuncCalls returns all function call names found in an expression AST.
// This is used by Config validation to build the user-function call graph.
func CollectFuncCalls(expr Expr) []string {
	var names []string
	collectFuncCallsRecursive(expr, &names)
	return names
}

func collectFuncCallsRecursive(expr Expr, names *[]string) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *FuncCallExpr:
		*names = append(*names, e.Name)
		for _, arg := range e.Args {
			collectFuncCallsRecursive(arg, names)
		}
	case *BinaryExpr:
		collectFuncCallsRecursive(e.Left, names)
		collectFuncCallsRecursive(e.Right, names)
	case *UnaryExpr:
		collectFuncCallsRecursive(e.Right, names)
	case *ComparisonExpr:
		collectFuncCallsRecursive(e.Left, names)
		collectFuncCallsRecursive(e.Right, names)
	case *NumberLiteral, *StringLiteral:
		// leaf nodes, no children
	}
}

// String returns a human-readable representation of a Value.
func (v Value) String() string {
	switch v.Kind {
	case ValueInt:
		return strconv.Itoa(v.Int)
	case ValueBool:
		return strconv.FormatBool(v.Bool)
	case ValueDeps:
		return fmt.Sprintf("deps[%d]", len(v.Deps))
	case ValueKindList:
		return fmt.Sprintf("list[%d]", len(v.List))
	}
	return "unknown"
}

// IsTruthy returns true if the value is "truthy" (for rule violation checks).
func (v Value) IsTruthy() bool {
	return v.AsBool()
}

// buildExprViolationMessage creates a message for expression-based violations.
func buildExprViolationMessage(rule Rule) string {
	if rule.Explanation != "" {
		return rule.Explanation
	}
	return fmt.Sprintf("Expression check failed: %s", rule.Check)
}

// ruleCheckMatches checks if a rule with a Check expression evaluates to true.
// This is used during rule evaluation to determine if a violation should be emitted.
func ruleCheckMatches(rule *Rule, ctx EvalContext) (bool, error) {
	if rule.compiledExpr == nil {
		return false, fmt.Errorf("rule %q: check expression not compiled", rule.ID)
	}
	val, err := rule.compiledExpr.Eval(ctx)
	if err != nil {
		return false, fmt.Errorf("rule %q: check expression evaluation failed: %w", rule.ID, err)
	}
	return val.IsTruthy(), nil
}

// compileCheckExpression compiles and caches the Check expression on the rule.
func (r *Rule) compileCheckExpression() error {
	if err := r.Check.Validate(); err != nil {
		return err
	}
	r.compiledExpr = r.Check.Expr
	return nil
}

// CheckExpressionIsStandalone returns true if the rule uses a Check expression
// and should bypass standard from/to logic.
func (r *Rule) CheckExpressionIsStandalone() bool {
	return r.Check.Raw != ""
}
