package domain

import (
	"fmt"
	"strconv"
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
	TokenComma:   ",",
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
	Deps   []Dependency
	Layers []Layer
}

type ValueKind int

const (
	ValueInt ValueKind = iota
	ValueBool
	ValueDeps
)

type Value struct {
	Kind ValueKind
	Int  int
	Bool bool
	Deps []Dependency
}

// AsBool converts a Value to bool for logical operations.
// Int > 0 is true; Deps with len > 0 is true.
func (v Value) AsBool() bool {
	switch v.Kind {
	case ValueBool:
		return v.Bool
	case ValueInt:
		return v.Int > 0
	case ValueDeps:
		return len(v.Deps) > 0
	}
	return false
}

// AsInt converts a Value to int.
// Bool true is 1, false is 0; Deps len is the int value.
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

// String returns a human-readable representation of a Value.
func (v Value) String() string {
	switch v.Kind {
	case ValueInt:
		return strconv.Itoa(v.Int)
	case ValueBool:
		return strconv.FormatBool(v.Bool)
	case ValueDeps:
		return fmt.Sprintf("deps[%d]", len(v.Deps))
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
func ruleCheckMatches(rule *Rule, deps []Dependency, layers []Layer) (bool, error) {
	if rule.compiledExpr == nil {
		return false, fmt.Errorf("rule %q: check expression not compiled", rule.ID)
	}
	ctx := EvalContext{Deps: deps, Layers: layers}
	val, err := rule.compiledExpr.Eval(ctx)
	if err != nil {
		return false, fmt.Errorf("rule %q: check expression evaluation failed: %w", rule.ID, err)
	}
	return val.IsTruthy(), nil
}

// compileCheckExpression parses and caches the Check expression on the rule.
func (r *Rule) compileCheckExpression() error {
	if r.Check == "" {
		r.compiledExpr = nil
		return nil
	}
	expr, err := Parse(r.Check)
	if err != nil {
		return fmt.Errorf("invalid check expression: %w", err)
	}
	r.compiledExpr = expr
	return nil
}

// CheckExpressionIsStandalone returns true if the rule uses a Check expression
// and should bypass standard from/to logic.
func (r *Rule) CheckExpressionIsStandalone() bool {
	return r.Check != ""
}
