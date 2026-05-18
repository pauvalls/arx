package domain

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// ─── Tokenizer Tests ─────────────────────────────────────────────────────────

func TestTokenize_BasicExpression(t *testing.T) {
	input := "count(deps(domain, infra)) > 3"
	tokens, err := tokenize(input)
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}

	want := []TokenType{
		TokenIdent,  // count
		TokenLParen, // (
		TokenIdent,  // deps
		TokenLParen, // (
		TokenIdent,  // domain
		TokenComma,  // ,
		TokenIdent,  // infra
		TokenRParen, // )
		TokenRParen, // )
		TokenGT,     // >
		TokenNumber, // 3
		TokenEOF,
	}

	if len(tokens) != len(want) {
		t.Fatalf("expected %d tokens, got %d: %v", len(want), len(tokens), tokens)
	}
	for i, tt := range want {
		if tokens[i].Type != tt {
			t.Errorf("token[%d]: expected %v, got %v (value=%q)", i, tt, tokens[i].Type, tokens[i].Value)
		}
	}
}

func TestTokenize_AllOperators(t *testing.T) {
	input := "> < >= <= == != && || !"
	tokens, err := tokenize(input)
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}

	want := []TokenType{TokenGT, TokenLT, TokenGE, TokenLE, TokenEQ, TokenNE, TokenAnd, TokenOr, TokenNot, TokenEOF}
	if len(tokens) != len(want) {
		t.Fatalf("expected %d tokens, got %d", len(want), len(tokens))
	}
	for i, tt := range want {
		if tokens[i].Type != tt {
			t.Errorf("token[%d]: expected %v, got %v", i, tt, tokens[i].Type)
		}
	}
}

func TestTokenize_ParensAndComma(t *testing.T) {
	input := "(a, b)"
	tokens, err := tokenize(input)
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}

	want := []TokenType{TokenLParen, TokenIdent, TokenComma, TokenIdent, TokenRParen, TokenEOF}
	if len(tokens) != len(want) {
		t.Fatalf("expected %d tokens, got %d", len(want), len(tokens))
	}
	for i, tt := range want {
		if tokens[i].Type != tt {
			t.Errorf("token[%d]: expected %v, got %v", i, tt, tokens[i].Type)
		}
	}
}

func TestTokenize_Numbers(t *testing.T) {
	input := "0 42 100"
	tokens, err := tokenize(input)
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}

	want := []TokenType{TokenNumber, TokenNumber, TokenNumber, TokenEOF}
	if len(tokens) != len(want) {
		t.Fatalf("expected %d tokens, got %d", len(want), len(tokens))
	}
	wantVals := []string{"0", "42", "100"}
	for i, val := range wantVals {
		if tokens[i].Value != val {
			t.Errorf("token[%d]: expected value %q, got %q", i, val, tokens[i].Value)
		}
	}
}

func TestTokenize_IdentifiersWithUnderscore(t *testing.T) {
	input := "has_circular _private ident123"
	tokens, err := tokenize(input)
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}

	want := []string{"has_circular", "_private", "ident123"}
	if len(tokens) != len(want)+1 { // +1 for EOF
		t.Fatalf("expected %d tokens, got %d", len(want)+1, len(tokens))
	}
	for i, val := range want {
		if tokens[i].Value != val {
			t.Errorf("token[%d]: expected value %q, got %q", i, val, tokens[i].Value)
		}
	}
}

func TestTokenize_UnexpectedCharacter(t *testing.T) {
	input := "a @ b"
	_, err := tokenize(input)
	if err == nil {
		t.Fatal("expected error for unexpected character, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected character") {
		t.Errorf("error message should contain 'unexpected character', got: %v", err)
	}
}

func TestTokenize_EmptyInput(t *testing.T) {
	tokens, err := tokenize("")
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}
	if len(tokens) != 1 || tokens[0].Type != TokenEOF {
		t.Fatalf("expected single EOF token, got %v", tokens)
	}
}

// ─── Parser Tests ────────────────────────────────────────────────────────────

func TestParse_ValidExpressions(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"simple comparison", "count(deps(domain, infra)) > 3"},
		{"boolean and", "count(deps(api, db)) > 2 && !has_circular()"},
		{"boolean or", "count(deps(a, b)) > 1 || count(deps(c, d)) > 2"},
		{"parenthesized", "(count(deps(a, b)) > 1)"},
		{"complex", "count(deps(x, y)) >= 0 && (layers() < 10 || has_circular())"},
		{"equality", "count(deps(a, b)) == 0"},
		{"not equal", "count(deps(a, b)) != 1"},
		{"not operator", "!has_circular()"},
		{"double not", "!!has_circular()"},
		{"nested call", "count(deps(domain, infra))"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			if err != nil {
				t.Errorf("Parse(%q) error: %v", tt.input, err)
			}
		})
	}
}

func TestParse_InvalidSyntax(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"unclosed paren", "count(deps(domain, infra)"},
		{"missing operand", "count() >"},
		{"empty", ""},
		{"unexpected token", "@"},
		{"trailing operator", "count(deps(a, b)) >"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			if err == nil {
				t.Errorf("Parse(%q) expected error, got nil", tt.input)
			}
		})
	}
}

func TestParse_ASTStructure(t *testing.T) {
	expr, err := Parse("count(deps(domain, infra)) > 3")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	cmp, ok := expr.(*ComparisonExpr)
	if !ok {
		t.Fatalf("expected *ComparisonExpr, got %T", expr)
	}
	if cmp.Op != TokenGT {
		t.Errorf("expected operator >, got %v", cmp.Op)
	}

	call, ok := cmp.Left.(*FuncCallExpr)
	if !ok {
		t.Fatalf("expected *FuncCallExpr on left, got %T", cmp.Left)
	}
	if call.Name != "count" {
		t.Errorf("expected function 'count', got %q", call.Name)
	}
	if len(call.Args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(call.Args))
	}
}

// ─── Evaluator Tests ─────────────────────────────────────────────────────────

func TestEval_NumberLiteral(t *testing.T) {
	expr := &NumberLiteral{Value: 42}
	val, err := expr.Eval(EvalContext{})
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if val.Kind != ValueInt || val.Int != 42 {
		t.Errorf("expected int 42, got %v", val)
	}
}

func TestEval_StringLiteral(t *testing.T) {
	expr := &StringLiteral{Value: "domain"}
	val, err := expr.Eval(EvalContext{})
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if val.Kind != ValueInt {
		t.Errorf("expected ValueInt kind for string literal (zero-value), got %v", val.Kind)
	}
}

func TestEval_Comparison(t *testing.T) {
	tests := []struct {
		name     string
		expr     Expr
		expected bool
	}{
		{
			name:     "5 > 3",
			expr:     &ComparisonExpr{Left: &NumberLiteral{5}, Op: TokenGT, Right: &NumberLiteral{3}},
			expected: true,
		},
		{
			name:     "3 > 5",
			expr:     &ComparisonExpr{Left: &NumberLiteral{3}, Op: TokenGT, Right: &NumberLiteral{5}},
			expected: false,
		},
		{
			name:     "3 == 3",
			expr:     &ComparisonExpr{Left: &NumberLiteral{3}, Op: TokenEQ, Right: &NumberLiteral{3}},
			expected: true,
		},
		{
			name:     "3 != 3",
			expr:     &ComparisonExpr{Left: &NumberLiteral{3}, Op: TokenNE, Right: &NumberLiteral{3}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := tt.expr.Eval(EvalContext{})
			if err != nil {
				t.Fatalf("eval error: %v", err)
			}
			if val.Kind != ValueBool || val.Bool != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, val)
			}
		})
	}
}

func TestEval_BinaryAnd(t *testing.T) {
	expr := &BinaryExpr{
		Left:  &ComparisonExpr{Left: &NumberLiteral{5}, Op: TokenGT, Right: &NumberLiteral{3}},
		Op:    TokenAnd,
		Right: &ComparisonExpr{Left: &NumberLiteral{2}, Op: TokenGT, Right: &NumberLiteral{1}},
	}
	val, err := expr.Eval(EvalContext{})
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if !val.Bool {
		t.Errorf("expected true, got %v", val)
	}
}

func TestEval_BinaryOr(t *testing.T) {
	expr := &BinaryExpr{
		Left:  &ComparisonExpr{Left: &NumberLiteral{1}, Op: TokenGT, Right: &NumberLiteral{5}},
		Op:    TokenOr,
		Right: &ComparisonExpr{Left: &NumberLiteral{2}, Op: TokenGT, Right: &NumberLiteral{1}},
	}
	val, err := expr.Eval(EvalContext{})
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if !val.Bool {
		t.Errorf("expected true, got %v", val)
	}
}

func TestEval_UnaryNot(t *testing.T) {
	expr := &UnaryExpr{
		Op:    TokenNot,
		Right: &ComparisonExpr{Left: &NumberLiteral{5}, Op: TokenGT, Right: &NumberLiteral{3}},
	}
	val, err := expr.Eval(EvalContext{})
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if val.Bool {
		t.Errorf("expected false, got %v", val)
	}
}

func TestEval_BuiltinCount(t *testing.T) {
	deps := []Dependency{
		{SourceFile: "domain/a.go", ResolvedLayer: "infra"},
		{SourceFile: "domain/b.go", ResolvedLayer: "infra"},
		{SourceFile: "domain/c.go", ResolvedLayer: "infra"},
	}
	layers := []Layer{
		{Name: "domain", Paths: []string{"domain/"}},
		{Name: "infra", Paths: []string{"infra/"}},
	}
	ctx := EvalContext{Deps: deps, Layers: layers}

	expr := &FuncCallExpr{
		Name: "count",
		Args: []Expr{
			&FuncCallExpr{
				Name: "deps",
				Args: []Expr{
					&StringLiteral{Value: "domain"},
					&StringLiteral{Value: "infra"},
				},
			},
		},
	}

	val, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if val.Kind != ValueInt || val.Int != 3 {
		t.Errorf("expected count=3, got %v", val)
	}
}

func TestEval_BuiltinLayers(t *testing.T) {
	ctx := EvalContext{Layers: []Layer{
		{Name: "a"},
		{Name: "b"},
		{Name: "c"},
	}}
	expr := &FuncCallExpr{Name: "layers", Args: []Expr{}}
	val, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if val.Kind != ValueInt || val.Int != 3 {
		t.Errorf("expected layers=3, got %v", val)
	}
}

func TestEval_BuiltinHasCircular(t *testing.T) {
	// No circular deps
	ctx := EvalContext{
		Deps: []Dependency{
			{SourceFile: "domain/a.go", ResolvedLayer: "infra"},
		},
		Layers: []Layer{
			{Name: "domain", Paths: []string{"domain/"}},
			{Name: "infra", Paths: []string{"infra/"}},
		},
	}
	expr := &FuncCallExpr{Name: "has_circular", Args: []Expr{}}
	val, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if val.Kind != ValueBool || val.Bool {
		t.Errorf("expected no circular deps, got %v", val)
	}
}

func TestEval_FullExpression(t *testing.T) {
	// count(deps(domain, infra)) > 2
	expr, err := Parse("count(deps(domain, infra)) > 2")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	deps := []Dependency{
		{SourceFile: "domain/a.go", ResolvedLayer: "infra"},
		{SourceFile: "domain/b.go", ResolvedLayer: "infra"},
		{SourceFile: "domain/c.go", ResolvedLayer: "infra"},
	}
	layers := []Layer{
		{Name: "domain", Paths: []string{"domain/"}},
		{Name: "infra", Paths: []string{"infra/"}},
	}
	ctx := EvalContext{Deps: deps, Layers: layers}

	val, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if !val.Bool {
		t.Errorf("expected true (3 > 2), got %v", val)
	}
}

func TestEval_BooleanExpression(t *testing.T) {
	// count(deps(api, db)) > 2 && !has_circular()
	expr, err := Parse("count(deps(api, db)) > 2 && !has_circular()")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	deps := []Dependency{
		{SourceFile: "api/a.go", ResolvedLayer: "db"},
		{SourceFile: "api/b.go", ResolvedLayer: "db"},
		{SourceFile: "api/c.go", ResolvedLayer: "db"},
	}
	layers := []Layer{
		{Name: "api", Paths: []string{"api/"}},
		{Name: "db", Paths: []string{"db/"}},
	}
	ctx := EvalContext{Deps: deps, Layers: layers}

	val, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if !val.Bool {
		t.Errorf("expected true (3 > 2 && !circular), got %v", val)
	}
}

// ─── Integration Tests ───────────────────────────────────────────────────────

func TestRule_CheckField_Valid(t *testing.T) {
	rule := Rule{
		ID:       "R-EXPR",
		Check:    CheckExpr{Raw: "count(deps(domain, infra)) > 3"},
		Severity: SeverityError,
	}
	err := rule.Validate()
	if err != nil {
		t.Fatalf("Rule.Validate() error: %v", err)
	}
	if rule.compiledExpr == nil {
		t.Error("expected compiledExpr to be set")
	}
}

func TestRule_CheckField_InvalidExpression(t *testing.T) {
	rule := Rule{
		ID:       "R-EXPR-BAD",
		Check:    CheckExpr{Raw: "count(invalid_syntax"},
		Severity: SeverityError,
	}
	err := rule.Validate()
	if err == nil {
		t.Fatal("expected validation error for invalid expression")
	}
	if !strings.Contains(err.Error(), "invalid check expression") {
		t.Errorf("error should contain 'invalid check expression', got: %v", err)
	}
}

func TestConfig_Validate_CheckRuleStandalone(t *testing.T) {
	// Valid: check-only rule
	config := Config{
		Version: "1.0.0",
		Layers: []Layer{
			{Name: "domain", Paths: []string{"internal/domain"}},
			{Name: "infra", Paths: []string{"internal/infrastructure"}},
		},
		Rules: []Rule{
			{
				ID:       "R-CHECK",
				Check:    CheckExpr{Raw: "count(deps(domain, infra)) > 3"},
				Severity: SeverityError,
			},
		},
	}
	err := config.Validate()
	if err != nil {
		t.Fatalf("Config.Validate() error: %v", err)
	}
}

func TestConfig_Validate_CheckRuleWithFromRejected(t *testing.T) {
	config := Config{
		Version: "1.0.0",
		Layers: []Layer{
			{Name: "domain", Paths: []string{"internal/domain"}},
			{Name: "infra", Paths: []string{"internal/infrastructure"}},
		},
		Rules: []Rule{
			{
				ID:       "R-CHECK-BAD",
				From:     "domain",
				To:       []string{"infra"},
				Check:    CheckExpr{Raw: "count(deps(domain, infra)) > 3"},
				Type:     RuleTypeCannot,
				Severity: SeverityError,
			},
		},
	}
	err := config.Validate()
	if err == nil {
		t.Fatal("expected error for check rule with from field")
	}
	if !strings.Contains(err.Error(), "cannot have 'from' field") {
		t.Errorf("error should mention 'from' field, got: %v", err)
	}
}

func TestConfig_Validate_CheckRuleWithToRejected(t *testing.T) {
	config := Config{
		Version: "1.0.0",
		Layers: []Layer{
			{Name: "domain", Paths: []string{"internal/domain"}},
			{Name: "infra", Paths: []string{"internal/infrastructure"}},
		},
		Rules: []Rule{
			{
				ID:       "R-CHECK-BAD",
				Check:    CheckExpr{Raw: "count(deps(domain, infra)) > 3"},
				To:       []string{"infra"},
				Severity: SeverityError,
			},
		},
	}
	err := config.Validate()
	if err == nil {
		t.Fatal("expected error for check rule with to field")
	}
	if !strings.Contains(err.Error(), "cannot have 'to' field") {
		t.Errorf("error should mention 'to' field, got: %v", err)
	}
}

func TestConfig_Validate_CheckRuleWithTemplateRejected(t *testing.T) {
	config := Config{
		Version: "1.0.0",
		Layers: []Layer{
			{Name: "domain", Paths: []string{"internal/domain"}},
		},
		Rules: []Rule{
			{
				ID:       "R-CHECK-BAD",
				Check:    CheckExpr{Raw: "count(deps(domain, infra)) > 3"},
				Template: "max-deps",
				Severity: SeverityError,
				Params:   map[string]interface{}{"from": "domain", "to": []interface{}{"infra"}, "max": 3},
			},
		},
	}
	err := config.Validate()
	if err == nil {
		t.Fatal("expected error for check rule with template field")
	}
	if !strings.Contains(err.Error(), "cannot have 'template' field") {
		t.Errorf("error should mention 'template' field, got: %v", err)
	}
}

func TestEvaluateRules_CheckExpression(t *testing.T) {
	deps := []Dependency{
		{SourceFile: "domain/a.go", ResolvedLayer: "infra"},
		{SourceFile: "domain/b.go", ResolvedLayer: "infra"},
		{SourceFile: "domain/c.go", ResolvedLayer: "infra"},
		{SourceFile: "domain/d.go", ResolvedLayer: "infra"},
	}
	layers := []Layer{
		{Name: "domain", Paths: []string{"domain/"}},
		{Name: "infra", Paths: []string{"infra/"}},
	}
	rules := []Rule{
		{
			ID:       "R-CHECK",
			Check:    CheckExpr{Raw: "count(deps(domain, infra)) > 3"},
			Severity: SeverityError,
		},
	}

	// Compile expressions
	for i := range rules {
		if err := rules[i].Validate(); err != nil {
			t.Fatalf("rule validation error: %v", err)
		}
	}

	violations := EvaluateRules(deps, rules, layers)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].RuleID != "R-CHECK" {
		t.Errorf("expected rule ID R-CHECK, got %s", violations[0].RuleID)
	}
}

func TestEvaluateRules_CheckExpressionNoViolation(t *testing.T) {
	deps := []Dependency{
		{SourceFile: "domain/a.go", ResolvedLayer: "infra"},
	}
	layers := []Layer{
		{Name: "domain", Paths: []string{"domain/"}},
		{Name: "infra", Paths: []string{"infra/"}},
	}
	rules := []Rule{
		{
			ID:       "R-CHECK",
			Check:    CheckExpr{Raw: "count(deps(domain, infra)) > 3"},
			Severity: SeverityError,
		},
	}

	for i := range rules {
		if err := rules[i].Validate(); err != nil {
			t.Fatalf("rule validation error: %v", err)
		}
	}

	violations := EvaluateRules(deps, rules, layers)
	if len(violations) != 0 {
		t.Fatalf("expected 0 violations, got %d", len(violations))
	}
}

func TestEvaluateRules_MixedTraditionalAndExpression(t *testing.T) {
	deps := []Dependency{
		{SourceFile: "domain/a.go", ResolvedLayer: "infra"},
		{SourceFile: "domain/b.go", ResolvedLayer: "infra"},
		{SourceFile: "domain/c.go", ResolvedLayer: "infra"},
		{SourceFile: "domain/d.go", ResolvedLayer: "infra"},
		{SourceFile: "app/x.go", ResolvedLayer: "db"},
	}
	layers := []Layer{
		{Name: "domain", Paths: []string{"domain/"}},
		{Name: "app", Paths: []string{"app/"}},
		{Name: "infra", Paths: []string{"infra/"}},
		{Name: "db", Paths: []string{"db/"}},
	}
	rules := []Rule{
		{
			ID:       "R-TRAD",
			From:     "app",
			To:       []string{"db"},
			Type:     RuleTypeCannot,
			Severity: SeverityError,
		},
		{
			ID:       "R-CHECK",
			Check:    CheckExpr{Raw: "count(deps(domain, infra)) > 3"},
			Severity: SeverityWarning,
		},
	}

	for i := range rules {
		if err := rules[i].Validate(); err != nil {
			t.Fatalf("rule validation error: %v", err)
		}
	}

	violations := EvaluateRules(deps, rules, layers)
	if len(violations) != 2 {
		t.Fatalf("expected 2 violations, got %d", len(violations))
	}

	var tradFound, exprFound bool
	for _, v := range violations {
		if v.RuleID == "R-TRAD" {
			tradFound = true
		}
		if v.RuleID == "R-CHECK" {
			exprFound = true
		}
	}
	if !tradFound {
		t.Error("expected violation for traditional rule R-TRAD")
	}
	if !exprFound {
		t.Error("expected violation for expression rule R-CHECK")
	}
}

func TestEvaluateRules_CheckExpressionWithExplanation(t *testing.T) {
	deps := []Dependency{
		{SourceFile: "domain/a.go", ResolvedLayer: "infra"},
		{SourceFile: "domain/b.go", ResolvedLayer: "infra"},
		{SourceFile: "domain/c.go", ResolvedLayer: "infra"},
		{SourceFile: "domain/d.go", ResolvedLayer: "infra"},
	}
	layers := []Layer{
		{Name: "domain", Paths: []string{"domain/"}},
		{Name: "infra", Paths: []string{"infra/"}},
	}
	rules := []Rule{
		{
			ID:          "R-CHECK",
			Check:       CheckExpr{Raw: "count(deps(domain, infra)) > 3"},
			Severity:    SeverityError,
			Explanation: "Too many domain->infra dependencies",
		},
	}

	for i := range rules {
		if err := rules[i].Validate(); err != nil {
			t.Fatalf("rule validation error: %v", err)
		}
	}

	violations := EvaluateRules(deps, rules, layers)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if !strings.Contains(violations[0].Message, "Too many domain->infra dependencies") {
		t.Errorf("expected explanation in message, got: %s", violations[0].Message)
	}
}

func TestEval_UnknownFunction(t *testing.T) {
	expr := &FuncCallExpr{Name: "unknown", Args: []Expr{}}
	_, err := expr.Eval(EvalContext{})
	if err == nil {
		t.Fatal("expected error for unknown function")
	}
	if !strings.Contains(err.Error(), "unknown function") {
		t.Errorf("error should mention 'unknown function', got: %v", err)
	}
}

func TestEval_CountWrongArgs(t *testing.T) {
	expr := &FuncCallExpr{Name: "count", Args: []Expr{}}
	_, err := expr.Eval(EvalContext{})
	if err == nil {
		t.Fatal("expected error for wrong arg count")
	}
}

func TestEval_DepsWrongArgs(t *testing.T) {
	expr := &FuncCallExpr{Name: "deps", Args: []Expr{&NumberLiteral{Value: 1}}}
	_, err := expr.Eval(EvalContext{})
	if err == nil {
		t.Fatal("expected error for wrong arg count")
	}
}

func TestTokenize_WhitespaceInsensitive(t *testing.T) {
	input1 := "count(deps(domain,infra))>3"
	input2 := "count( deps( domain , infra ) ) > 3"

	tokens1, err := tokenize(input1)
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}
	tokens2, err := tokenize(input2)
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}

	// Both should produce same token types (ignoring positions)
	if len(tokens1) != len(tokens2) {
		t.Fatalf("expected same token count: %d vs %d", len(tokens1), len(tokens2))
	}
	for i := range tokens1 {
		if tokens1[i].Type != tokens2[i].Type {
			t.Errorf("token[%d]: type mismatch %v vs %v", i, tokens1[i].Type, tokens2[i].Type)
		}
	}
}

// ─── Extended Expression Function Tests ──────────────────────────────────────

func TestEval_BuiltinFiles(t *testing.T) {
	ctx := EvalContext{
		Layers: []Layer{
			{Name: "domain", Paths: []string{"domain/"}},
			{Name: "infra", Paths: []string{"infra/"}},
		},
		LayerFiles: map[string][]string{
			"domain": {"domain/a.go", "domain/b.go", "domain/c.go"},
			"infra":  {"infra/x.go"},
		},
	}

	expr := &FuncCallExpr{
		Name: "files",
		Args: []Expr{&StringLiteral{Value: "domain"}},
	}
	val, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if val.Kind != ValueInt || val.Int != 3 {
		t.Errorf("expected files(domain)=3, got %v", val)
	}

	// Empty layer
	expr2 := &FuncCallExpr{
		Name: "files",
		Args: []Expr{&StringLiteral{Value: "unknown"}},
	}
	val2, err := expr2.Eval(ctx)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if val2.Kind != ValueInt || val2.Int != 0 {
		t.Errorf("expected files(unknown)=0, got %v", val2)
	}
}

func TestEval_BuiltinFilesWrongArgs(t *testing.T) {
	expr := &FuncCallExpr{Name: "files", Args: []Expr{}}
	_, err := expr.Eval(EvalContext{})
	if err == nil {
		t.Fatal("expected error for wrong arg count")
	}
}

func TestEval_BuiltinRatio(t *testing.T) {
	// ratio(4, 8) = 0 (integer division)
	expr := &FuncCallExpr{
		Name: "ratio",
		Args: []Expr{&NumberLiteral{Value: 4}, &NumberLiteral{Value: 8}},
	}
	val, err := expr.Eval(EvalContext{})
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if val.Kind != ValueInt || val.Int != 0 {
		t.Errorf("expected ratio(4,8)=0, got %v", val)
	}

	// ratio(8, 4) = 2
	expr2 := &FuncCallExpr{
		Name: "ratio",
		Args: []Expr{&NumberLiteral{Value: 8}, &NumberLiteral{Value: 4}},
	}
	val2, err := expr2.Eval(EvalContext{})
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if val2.Kind != ValueInt || val2.Int != 2 {
		t.Errorf("expected ratio(8,4)=2, got %v", val2)
	}

	// ratio(5, 0) = 0 (division by zero protection)
	expr3 := &FuncCallExpr{
		Name: "ratio",
		Args: []Expr{&NumberLiteral{Value: 5}, &NumberLiteral{Value: 0}},
	}
	val3, err := expr3.Eval(EvalContext{})
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if val3.Kind != ValueInt || val3.Int != 0 {
		t.Errorf("expected ratio(5,0)=0, got %v", val3)
	}
}

func TestEval_BuiltinRatioWrongArgs(t *testing.T) {
	expr := &FuncCallExpr{Name: "ratio", Args: []Expr{&NumberLiteral{Value: 1}}}
	_, err := expr.Eval(EvalContext{})
	if err == nil {
		t.Fatal("expected error for wrong arg count")
	}
}

func TestEval_BuiltinViolations(t *testing.T) {
	ctx := EvalContext{
		Violations: []Violation{
			{RuleID: "domain-no-import-infra"},
			{RuleID: "domain-no-import-infra"},
			{RuleID: "domain-no-import-infra"},
			{RuleID: "other-rule"},
		},
	}

	expr := &FuncCallExpr{
		Name: "violations",
		Args: []Expr{&StringLiteral{Value: "domain-no-import-infra"}},
	}
	val, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if val.Kind != ValueInt || val.Int != 3 {
		t.Errorf("expected violations(domain-no-import-infra)=3, got %v", val)
	}

	// Unknown rule
	expr2 := &FuncCallExpr{
		Name: "violations",
		Args: []Expr{&StringLiteral{Value: "missing-rule"}},
	}
	val2, err := expr2.Eval(ctx)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if val2.Kind != ValueInt || val2.Int != 0 {
		t.Errorf("expected violations(missing-rule)=0, got %v", val2)
	}
}

func TestEval_BuiltinViolationsWrongArgs(t *testing.T) {
	expr := &FuncCallExpr{Name: "violations", Args: []Expr{}}
	_, err := expr.Eval(EvalContext{})
	if err == nil {
		t.Fatal("expected error for wrong arg count")
	}
}

func TestEval_BuiltinThreshold(t *testing.T) {
	// threshold(3, 0, 5) = true
	expr := &FuncCallExpr{
		Name: "threshold",
		Args: []Expr{&NumberLiteral{Value: 3}, &NumberLiteral{Value: 0}, &NumberLiteral{Value: 5}},
	}
	val, err := expr.Eval(EvalContext{})
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if val.Kind != ValueBool || !val.Bool {
		t.Errorf("expected threshold(3,0,5)=true, got %v", val)
	}

	// threshold(5, 0, 5) = true (inclusive)
	expr2 := &FuncCallExpr{
		Name: "threshold",
		Args: []Expr{&NumberLiteral{Value: 5}, &NumberLiteral{Value: 0}, &NumberLiteral{Value: 5}},
	}
	val2, err := expr2.Eval(EvalContext{})
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if val.Kind != ValueBool || !val2.Bool {
		t.Errorf("expected threshold(5,0,5)=true, got %v", val2)
	}

	// threshold(6, 0, 5) = false
	expr3 := &FuncCallExpr{
		Name: "threshold",
		Args: []Expr{&NumberLiteral{Value: 6}, &NumberLiteral{Value: 0}, &NumberLiteral{Value: 5}},
	}
	val3, err := expr3.Eval(EvalContext{})
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if val.Kind != ValueBool || val3.Bool {
		t.Errorf("expected threshold(6,0,5)=false, got %v", val3)
	}

	// threshold(-1, 0, 5) = false
	expr4 := &FuncCallExpr{
		Name: "threshold",
		Args: []Expr{&NumberLiteral{Value: -1}, &NumberLiteral{Value: 0}, &NumberLiteral{Value: 5}},
	}
	val4, err := expr4.Eval(EvalContext{})
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if val.Kind != ValueBool || val4.Bool {
		t.Errorf("expected threshold(-1,0,5)=false, got %v", val4)
	}
}

func TestEval_BuiltinThresholdWrongArgs(t *testing.T) {
	expr := &FuncCallExpr{Name: "threshold", Args: []Expr{&NumberLiteral{Value: 1}}}
	_, err := expr.Eval(EvalContext{})
	if err == nil {
		t.Fatal("expected error for wrong arg count")
	}
}

func TestEval_ExtendedFunctionsInExpression(t *testing.T) {
	// threshold(count(deps(domain, infra)), 0, 5)
	deps := []Dependency{
		{SourceFile: "domain/a.go", ResolvedLayer: "infra"},
		{SourceFile: "domain/b.go", ResolvedLayer: "infra"},
		{SourceFile: "domain/c.go", ResolvedLayer: "infra"},
	}
	layers := []Layer{
		{Name: "domain", Paths: []string{"domain/"}},
		{Name: "infra", Paths: []string{"infra/"}},
	}
	ctx := EvalContext{Deps: deps, Layers: layers}

	expr, err := Parse("threshold(count(deps(domain, infra)), 0, 5)")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	val, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if !val.Bool {
		t.Errorf("expected true (3 in [0,5]), got %v", val)
	}
}

func TestEval_FilesWithLayerFiles(t *testing.T) {
	// files(domain) with LayerFiles populated
	ctx := EvalContext{
		Layers: []Layer{
			{Name: "domain", Paths: []string{"domain/"}},
		},
		LayerFiles: map[string][]string{
			"domain": {
				"domain/a.go",
				"domain/b.go",
				"domain/c.go",
				"domain/d.go",
				"domain/e.go",
				"domain/f.go",
				"domain/g.go",
				"domain/h.go",
				"domain/i.go",
				"domain/j.go",
				"domain/k.go",
				"domain/l.go",
			},
		},
	}

	expr, err := Parse("files(domain)")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	val, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if val.Kind != ValueInt || val.Int != 12 {
		t.Errorf("expected files(domain)=12, got %v", val)
	}
}

func TestEval_RatioInExpression(t *testing.T) {
	// ratio(8, 4) == 2
	expr, err := Parse("ratio(8, 4) == 2")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	val, err := expr.Eval(EvalContext{})
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if !val.Bool {
		t.Errorf("expected true (8/4 == 2), got %v", val)
	}
}

func TestEval_ViolationsInExpression(t *testing.T) {
	ctx := EvalContext{
		Violations: []Violation{
			{RuleID: "R1"},
			{RuleID: "R1"},
			{RuleID: "R2"},
		},
	}

	expr, err := Parse("violations(R1) == 2")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	val, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if !val.Bool {
		t.Errorf("expected true (violations(R1)=2), got %v", val)
	}
}

// ─── all() / any() Builtin Tests ─────────────────────────────────────────────

func TestEval_BuiltinAll_WithDeps(t *testing.T) {
	deps := []Dependency{
		{SourceFile: "domain/a.go", ResolvedLayer: "infra"},
		{SourceFile: "domain/b.go", ResolvedLayer: "infra"},
	}
	layers := []Layer{
		{Name: "domain", Paths: []string{"domain/"}},
		{Name: "infra", Paths: []string{"infra/"}},
	}
	ctx := EvalContext{Deps: deps, Layers: layers}

	expr := &FuncCallExpr{
		Name: "all",
		Args: []Expr{
			&FuncCallExpr{
				Name: "deps",
				Args: []Expr{
					&StringLiteral{Value: "domain"},
					&StringLiteral{Value: "infra"},
				},
			},
		},
	}
	val, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("all(deps(domain, infra)) eval error: %v", err)
	}
	if val.Kind != ValueBool || !val.Bool {
		t.Errorf("expected true, got %v", val)
	}
}

func TestEval_BuiltinAll_EmptyDeps(t *testing.T) {
	// No deps from domain to db
	deps := []Dependency{
		{SourceFile: "domain/a.go", ResolvedLayer: "infra"},
	}
	layers := []Layer{
		{Name: "domain", Paths: []string{"domain/"}},
		{Name: "db", Paths: []string{"db/"}},
		{Name: "infra", Paths: []string{"infra/"}},
	}
	ctx := EvalContext{Deps: deps, Layers: layers}

	expr := &FuncCallExpr{
		Name: "all",
		Args: []Expr{
			&FuncCallExpr{
				Name: "deps",
				Args: []Expr{
					&StringLiteral{Value: "domain"},
					&StringLiteral{Value: "db"},
				},
			},
		},
	}
	val, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("all(deps(domain, db)) eval error: %v", err)
	}
	if val.Kind != ValueBool || val.Bool {
		t.Errorf("expected false for empty deps, got %v", val)
	}
}

func TestEval_BuiltinAll_WrongArgCount(t *testing.T) {
	// Zero args
	expr := &FuncCallExpr{Name: "all", Args: []Expr{}}
	_, err := expr.Eval(EvalContext{})
	if err == nil {
		t.Fatal("expected error for all() with no args")
	}
	if !strings.Contains(err.Error(), "expects exactly 1 argument") {
		t.Errorf("error should mention arg count, got: %v", err)
	}

	// Two args
	expr2 := &FuncCallExpr{
		Name: "all",
		Args: []Expr{&NumberLiteral{Value: 1}, &NumberLiteral{Value: 2}},
	}
	_, err = expr2.Eval(EvalContext{})
	if err == nil {
		t.Fatal("expected error for all() with two args")
	}
	if !strings.Contains(err.Error(), "expects exactly 1 argument") {
		t.Errorf("error should mention arg count, got: %v", err)
	}
}

func TestEval_BuiltinAll_WrongArgType(t *testing.T) {
	// Passing a number instead of deps()
	expr := &FuncCallExpr{
		Name: "all",
		Args: []Expr{&NumberLiteral{Value: 42}},
	}
	_, err := expr.Eval(EvalContext{})
	if err == nil {
		t.Fatal("expected error for all() with non-deps arg")
	}
	if !strings.Contains(err.Error(), "expects a deps() call") {
		t.Errorf("error should mention deps() call, got: %v", err)
	}
}

func TestEval_BuiltinAny_WithDeps(t *testing.T) {
	deps := []Dependency{
		{SourceFile: "domain/a.go", ResolvedLayer: "infra"},
	}
	layers := []Layer{
		{Name: "domain", Paths: []string{"domain/"}},
		{Name: "infra", Paths: []string{"infra/"}},
	}
	ctx := EvalContext{Deps: deps, Layers: layers}

	expr := &FuncCallExpr{
		Name: "any",
		Args: []Expr{
			&FuncCallExpr{
				Name: "deps",
				Args: []Expr{
					&StringLiteral{Value: "domain"},
					&StringLiteral{Value: "infra"},
				},
			},
		},
	}
	val, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("any(deps(domain, infra)) eval error: %v", err)
	}
	if val.Kind != ValueBool || !val.Bool {
		t.Errorf("expected true, got %v", val)
	}
}

func TestEval_BuiltinAny_EmptyDeps(t *testing.T) {
	deps := []Dependency{
		{SourceFile: "domain/a.go", ResolvedLayer: "infra"},
	}
	layers := []Layer{
		{Name: "domain", Paths: []string{"domain/"}},
		{Name: "db", Paths: []string{"db/"}},
		{Name: "infra", Paths: []string{"infra/"}},
	}
	ctx := EvalContext{Deps: deps, Layers: layers}

	// No deps from domain to db
	expr := &FuncCallExpr{
		Name: "any",
		Args: []Expr{
			&FuncCallExpr{
				Name: "deps",
				Args: []Expr{
					&StringLiteral{Value: "domain"},
					&StringLiteral{Value: "db"},
				},
			},
		},
	}
	val, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("any(deps(domain, db)) eval error: %v", err)
	}
	if val.Kind != ValueBool || val.Bool {
		t.Errorf("expected false for empty deps, got %v", val)
	}
}

func TestEval_BuiltinAny_WrongArgCount(t *testing.T) {
	// Zero args
	expr := &FuncCallExpr{Name: "any", Args: []Expr{}}
	_, err := expr.Eval(EvalContext{})
	if err == nil {
		t.Fatal("expected error for any() with no args")
	}
	if !strings.Contains(err.Error(), "expects exactly 1 argument") {
		t.Errorf("error should mention arg count, got: %v", err)
	}
}

func TestEval_BuiltinAny_WrongArgType(t *testing.T) {
	expr := &FuncCallExpr{
		Name: "any",
		Args: []Expr{&StringLiteral{Value: "not_a_deps_call"}},
	}
	_, err := expr.Eval(EvalContext{})
	if err == nil {
		t.Fatal("expected error for any() with non-deps arg")
	}
	if !strings.Contains(err.Error(), "expects a deps() call") {
		t.Errorf("error should mention deps() call, got: %v", err)
	}
}

func TestEval_AllInExpression(t *testing.T) {
	// Parse and eval all(deps(domain, infra))
	deps := []Dependency{
		{SourceFile: "domain/a.go", ResolvedLayer: "infra"},
		{SourceFile: "domain/b.go", ResolvedLayer: "infra"},
	}
	layers := []Layer{
		{Name: "domain", Paths: []string{"domain/"}},
		{Name: "infra", Paths: []string{"infra/"}},
	}
	ctx := EvalContext{Deps: deps, Layers: layers}

	expr, err := Parse("all(deps(domain, infra))")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	val, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if val.Kind != ValueBool || !val.Bool {
		t.Errorf("expected true for all(deps(domain, infra)), got %v", val)
	}
}

func TestEval_AnyInExpression(t *testing.T) {
	// Parse and eval any(deps(domain, infra))
	deps := []Dependency{
		{SourceFile: "domain/a.go", ResolvedLayer: "infra"},
	}
	layers := []Layer{
		{Name: "domain", Paths: []string{"domain/"}},
		{Name: "infra", Paths: []string{"infra/"}},
	}
	ctx := EvalContext{Deps: deps, Layers: layers}

	expr, err := Parse("any(deps(domain, infra))")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	val, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if val.Kind != ValueBool || !val.Bool {
		t.Errorf("expected true for any(deps(domain, infra)), got %v", val)
	}
}

func TestEval_AllInExpression_EmptyDeps(t *testing.T) {
	// Parse and eval all(deps(domain, db)) — no deps exist
	deps := []Dependency{
		{SourceFile: "domain/a.go", ResolvedLayer: "infra"},
	}
	layers := []Layer{
		{Name: "domain", Paths: []string{"domain/"}},
		{Name: "db", Paths: []string{"db/"}},
		{Name: "infra", Paths: []string{"infra/"}},
	}
	ctx := EvalContext{Deps: deps, Layers: layers}

	expr, err := Parse("all(deps(domain, db))")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	val, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if val.Kind != ValueBool || val.Bool {
		t.Errorf("expected false for all(deps(domain, db)) with no deps, got %v", val)
	}
}

// ─── CheckExpr Tests ─────────────────────────────────────────────────────────

func TestCheckExpr_UnmarshalYAML_SingleString(t *testing.T) {
	// Backward compat: single string works exactly as before
	input := `check: "count(deps(domain, infra)) > 3"`
	var cfg struct {
		Check CheckExpr `yaml:"check"`
	}
	err := yaml.Unmarshal([]byte(input), &cfg)
	if err != nil {
		t.Fatalf("UnmarshalYAML error: %v", err)
	}
	if cfg.Check.Raw != "count(deps(domain, infra)) > 3" {
		t.Errorf("Raw = %q, want %q", cfg.Check.Raw, "count(deps(domain, infra)) > 3")
	}
	if cfg.Check.items != nil {
		t.Error("items should be nil for single string")
	}
	// Validate compiles successfully
	if err := cfg.Check.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if cfg.Check.Expr == nil {
		t.Fatal("Expr should be set after Validate()")
	}
}

func TestCheckExpr_UnmarshalYAML_List(t *testing.T) {
	input := `check: ["deps(a, b)", "violations(c) == 0"]`
	var cfg struct {
		Check CheckExpr `yaml:"check"`
	}
	err := yaml.Unmarshal([]byte(input), &cfg)
	if err != nil {
		t.Fatalf("UnmarshalYAML error: %v", err)
	}
	if cfg.Check.Raw != "deps(a, b) && violations(c) == 0" {
		t.Errorf("Raw = %q, want %q", cfg.Check.Raw, "deps(a, b) && violations(c) == 0")
	}
	if len(cfg.Check.items) != 2 {
		t.Fatalf("items length = %d, want 2", len(cfg.Check.items))
	}
	if cfg.Check.items[0] != "deps(a, b)" || cfg.Check.items[1] != "violations(c) == 0" {
		t.Errorf("items = %v, want [deps(a, b) violations(c) == 0]", cfg.Check.items)
	}
	// Validate compiles successfully
	if err := cfg.Check.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if cfg.Check.Expr == nil {
		t.Fatal("Expr should be set after Validate()")
	}
	// Verify it's an AND tree
	bin, ok := cfg.Check.Expr.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected *BinaryExpr for AND tree, got %T", cfg.Check.Expr)
	}
	if bin.Op != TokenAnd {
		t.Errorf("expected AND op, got %v", bin.Op)
	}
}

func TestCheckExpr_Validate_List_AllTruthy(t *testing.T) {
	ce := &CheckExpr{
		Raw: "layers() > 0 && !has_circular()",
	}
	// Validate triggers compilation
	if err := ce.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if ce.Expr == nil {
		t.Fatal("Expr is nil after Validate()")
	}
	// Eval with a context that makes both true
	ctx := EvalContext{
		Layers: []Layer{{Name: "a"}, {Name: "b"}},
	}
	val, err := ce.Expr.Eval(ctx)
	if err != nil {
		t.Fatalf("Eval() error: %v", err)
	}
	if !val.IsTruthy() {
		t.Error("expected truthy (layers > 0 && !circular)")
	}
}

func TestCheckExpr_Validate_List_AnyFalse(t *testing.T) {
	// layes() > 0 is true, but has_circular() is also true (no deps = no circular = false)
	// Actually !has_circular() when there IS no circular deps should be true
	// Let's use a different expression: layers() > 0 && layers() == 0 — always false
	ce := &CheckExpr{
		Raw: "layers() > 0 && layers() == 0",
	}
	if err := ce.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	ctx := EvalContext{
		Layers: []Layer{{Name: "a"}},
	}
	val, err := ce.Expr.Eval(ctx)
	if err != nil {
		t.Fatalf("Eval() error: %v", err)
	}
	if val.IsTruthy() {
		t.Error("expected falsy (layers > 0 && layers == 0)")
	}
}

func TestCheckExpr_Validate_List_CompileError(t *testing.T) {
	// List with one valid and one broken expression
	ce := &CheckExpr{
		Raw:   "layers() > 0 && broken((",
		items: []string{"layers() > 0", "broken(("},
	}
	err := ce.Validate()
	if err == nil {
		t.Fatal("expected error for broken expression in list")
	}
	if !strings.Contains(err.Error(), "broken((") {
		t.Errorf("error should mention the broken item, got: %v", err)
	}
}

func TestCheckExpr_UnmarshalYAML_InvalidType(t *testing.T) {
	// A mapping is not accepted — not a string or list of strings
	input := `check: {bad: type}`
	var cfg struct {
		Check CheckExpr `yaml:"check"`
	}
	err := yaml.Unmarshal([]byte(input), &cfg)
	if err == nil {
		t.Fatal("expected error for mapping check, got nil")
	}
	if !strings.Contains(err.Error(), "must be a string or list of strings") {
		t.Errorf("error should mention valid types, got: %v", err)
	}
}

func TestCheckExpr_UnmarshalYAML_EmptyList(t *testing.T) {
	input := `check: []`
	var cfg struct {
		Check CheckExpr `yaml:"check"`
	}
	err := yaml.Unmarshal([]byte(input), &cfg)
	if err == nil {
		t.Fatal("expected error for empty list, got nil")
	}
	if !strings.Contains(err.Error(), "must not be empty") {
		t.Errorf("error should mention empty list, got: %v", err)
	}
}

func TestCheckExpr_Rule_Validate_ListForm(t *testing.T) {
	// Integration test: Rule with list-form Check validates and evaluates correctly
	rule := Rule{
		ID:    "R-LIST",
		Check: CheckExpr{Raw: "layers() > 0 && layers() < 10"},
		Severity: SeverityError,
	}
	if err := rule.Validate(); err != nil {
		t.Fatalf("Rule.Validate() error: %v", err)
	}
	if rule.compiledExpr == nil {
		t.Fatal("compiledExpr should be set after Validate()")
	}
	// CheckExpressionIsStandalone returns true
	if !rule.CheckExpressionIsStandalone() {
		t.Error("CheckExpressionIsStandalone should be true for check-based rule")
	}
}

func TestCheckExpr_Empty_Raw(t *testing.T) {
	// Empty CheckExpr — no check expression
	ce := &CheckExpr{Raw: ""}
	if err := ce.Validate(); err != nil {
		t.Fatalf("Validate() error for empty: %v", err)
	}
	if ce.Expr != nil {
		t.Error("Expr should be nil for empty check")
	}
}

func TestCheckExpr_String(t *testing.T) {
	ce := CheckExpr{Raw: "deps(a, b) > 0"}
	if ce.String() != "deps(a, b) > 0" {
		t.Errorf("String() = %q, want %q", ce.String(), "deps(a, b) > 0")
	}
}

// ─── User-Defined Function Tests ──────────────────────────────────────────────

func TestIsValidIdentifier(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid simple", "my_func", true},
		{"valid with digits", "fn123", true},
		{"valid underscore prefix", "_private", true},
		{"empty", "", false},
		{"starts with digit", "123bad", false},
		{"contains special", "bad-func", false},
		{"contains space", "my func", false},
		{"single letter", "f", true},
		{"single underscore", "_", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidIdentifier(tt.input); got != tt.want {
				t.Errorf("IsValidIdentifier(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsBuiltinName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"count", true},
		{"deps", true},
		{"layers", true},
		{"has_circular", true},
		{"files", true},
		{"ratio", true},
		{"violations", true},
		{"threshold", true},
		{"all", true},
		{"any", true},
		{"my_func", false},
		{"unknown", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBuiltinName(tt.name); got != tt.want {
				t.Errorf("IsBuiltinName(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestCollectFuncCalls(t *testing.T) {
	expr, err := Parse("a(b(c(), d())) && e()")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	calls := CollectFuncCalls(expr)
	expected := []string{"a", "b", "c", "d", "e"}
	if len(calls) != len(expected) {
		t.Fatalf("got %d calls, want %d: %v", len(calls), len(expected), calls)
	}
	for i, want := range expected {
		if calls[i] != want {
			t.Errorf("call[%d] = %q, want %q", i, calls[i], want)
		}
	}
}

func TestCollectFuncCalls_Nil(t *testing.T) {
	// Should not panic
	names := CollectFuncCalls(nil)
	if names != nil {
		t.Errorf("expected nil, got %v", names)
	}
}

// ─── Config-level user function validation ──────────────────────────────────

func TestConfig_Validate_ValidUserFunctions(t *testing.T) {
	config := Config{
		Version: "1.0.0",
		Layers: []Layer{
			{Name: "domain", Paths: []string{"domain/"}},
			{Name: "infra", Paths: []string{"infra/"}},
		},
		Functions: map[string]string{
			"no_forbidden": "violations(forbidden_dep) == 0",
		},
	}
	err := config.Validate()
	if err != nil {
		t.Fatalf("Config.Validate() error: %v", err)
	}
	if config.UserFunctions() == nil {
		t.Fatal("expected UserFunctions to be non-nil after Validate()")
	}
	if _, ok := config.UserFunctions()["no_forbidden"]; !ok {
		t.Error("expected no_forbidden to be compiled")
	}
}

func TestConfig_Validate_UserFunctionCallsBuiltin(t *testing.T) {
	// User function body that uses builtins and comparisons
	config := Config{
		Version: "1.0.0",
		Layers: []Layer{
			{Name: "domain", Paths: []string{"domain/"}},
			{Name: "infra", Paths: []string{"infra/"}},
		},
		Functions: map[string]string{
			"count_deps": "count(deps(domain, infra))",
		},
	}
	err := config.Validate()
	if err != nil {
		t.Fatalf("Config.Validate() error: %v", err)
	}
}

func TestConfig_Validate_UserFunctionCrossReference(t *testing.T) {
	// Function A calls function B — no cycle, should validate
	config := Config{
		Version: "1.0.0",
		Layers: []Layer{
			{Name: "domain", Paths: []string{"domain/"}},
			{Name: "infra", Paths: []string{"infra/"}},
		},
		Functions: map[string]string{
			"no_cycle": "all(deps(a, b))",
			"strict":   "no_cycle() && violations(c) == 0",
		},
	}
	err := config.Validate()
	if err != nil {
		t.Fatalf("Config.Validate() error: %v", err)
	}
}

func TestConfig_Validate_UserFunctionCircularReference(t *testing.T) {
	config := Config{
		Version: "1.0.0",
		Layers: []Layer{
			{Name: "domain", Paths: []string{"domain/"}},
		},
		Functions: map[string]string{
			"a": "b()",
			"b": "a()",
		},
	}
	err := config.Validate()
	if err == nil {
		t.Fatal("expected error for circular reference, got nil")
	}
	if !strings.Contains(err.Error(), "circular reference") {
		t.Errorf("error should contain 'circular reference', got: %v", err)
	}
}

func TestConfig_Validate_UserFunctionIndirectCycle(t *testing.T) {
	config := Config{
		Version: "1.0.0",
		Layers: []Layer{
			{Name: "domain", Paths: []string{"domain/"}},
		},
		Functions: map[string]string{
			"a": "b()",
			"b": "c()",
			"c": "a()",
		},
	}
	err := config.Validate()
	if err == nil {
		t.Fatal("expected error for indirect cycle, got nil")
	}
	if !strings.Contains(err.Error(), "circular reference") {
		t.Errorf("error should contain 'circular reference', got: %v", err)
	}
}

func TestConfig_Validate_UserFunctionSelfReference(t *testing.T) {
	config := Config{
		Version: "1.0.0",
		Layers: []Layer{
			{Name: "domain", Paths: []string{"domain/"}},
		},
		Functions: map[string]string{
			"f": "f()",
		},
	}
	err := config.Validate()
	if err == nil {
		t.Fatal("expected error for self-reference, got nil")
	}
	if !strings.Contains(err.Error(), "circular reference") {
		t.Errorf("error should contain 'circular reference', got: %v", err)
	}
}

func TestConfig_Validate_UserFunctionBuiltinShadowing(t *testing.T) {
	tests := []string{"deps", "count", "layers", "has_circular", "files", "ratio", "violations", "threshold", "all", "any"}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			config := Config{
				Version: "1.0.0",
				Layers: []Layer{
					{Name: "domain", Paths: []string{"domain/"}},
				},
				Functions: map[string]string{
					name: "true",
				},
			}
			err := config.Validate()
			if err == nil {
				t.Fatalf("expected error for shadowing builtin %q, got nil", name)
			}
			if !strings.Contains(err.Error(), "cannot shadow builtin") {
				t.Errorf("error should contain 'cannot shadow builtin', got: %v", err)
			}
		})
	}
}

func TestConfig_Validate_UserFunctionInvalidIdentifier(t *testing.T) {
	config := Config{
		Version: "1.0.0",
		Layers: []Layer{
			{Name: "domain", Paths: []string{"domain/"}},
		},
		Functions: map[string]string{
			"123bad_name": "true",
		},
	}
	err := config.Validate()
	if err == nil {
		t.Fatal("expected error for invalid identifier, got nil")
	}
	if !strings.Contains(err.Error(), "invalid identifier") {
		t.Errorf("error should contain 'invalid identifier', got: %v", err)
	}
}

func TestConfig_Validate_UserFunctionParseError(t *testing.T) {
	config := Config{
		Version: "1.0.0",
		Layers: []Layer{
			{Name: "domain", Paths: []string{"domain/"}},
		},
		Functions: map[string]string{
			"f": "deps(",
		},
	}
	err := config.Validate()
	if err == nil {
		t.Fatal("expected error for parse error in function body, got nil")
	}
}

// ─── User function evaluation tests ─────────────────────────────────────────

func TestEval_UserFunctionCallsBuiltin(t *testing.T) {
	// Simulate: user function "no_forbidden" = "violations(forbidden_dep) == 0"
	userBody, err := Parse("violations(forbidden_dep) == 0")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	userFuncs := map[string]Expr{
		"no_forbidden": userBody,
	}

	ctx := EvalContext{
		Violations: []Violation{
			{RuleID: "forbidden_dep"},
		},
		UserFunctions: userFuncs,
	}

	// Evaluate no_forbidden() — should be false because violations > 0
	expr := &FuncCallExpr{Name: "no_forbidden", Args: []Expr{}}
	val, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if val.IsTruthy() {
		t.Error("expected false (violations == 1, not 0)")
	}
}

func TestEval_UserFunctionAllBuiltins(t *testing.T) {
	// User function calls all() with a deps() call
	userBody, err := Parse("all(deps(domain, infra))")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	userFuncs := map[string]Expr{
		"check_deps": userBody,
	}

	deps := []Dependency{
		{SourceFile: "domain/a.go", ResolvedLayer: "infra"},
		{SourceFile: "domain/b.go", ResolvedLayer: "infra"},
	}
	layers := []Layer{
		{Name: "domain", Paths: []string{"domain/"}},
		{Name: "infra", Paths: []string{"infra/"}},
	}
	ctx := EvalContext{
		Deps:          deps,
		Layers:        layers,
		UserFunctions: userFuncs,
	}

	expr := &FuncCallExpr{Name: "check_deps", Args: []Expr{}}
	val, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if !val.IsTruthy() {
		t.Error("expected true (deps exist from domain to infra)")
	}
}

func TestEval_UserFunctionCrossCall(t *testing.T) {
	// Function B calls function A: a="count(deps(domain, infra))", b="a() > 0"
	fnA, err := Parse("count(deps(domain, infra))")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	fnB, err := Parse("a() > 0")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	userFuncs := map[string]Expr{
		"a": fnA,
		"b": fnB,
	}

	deps := []Dependency{
		{SourceFile: "domain/a.go", ResolvedLayer: "infra"},
	}
	layers := []Layer{
		{Name: "domain", Paths: []string{"domain/"}},
		{Name: "infra", Paths: []string{"infra/"}},
	}
	ctx := EvalContext{
		Deps:          deps,
		Layers:        layers,
		UserFunctions: userFuncs,
	}

	// Calling b() should evaluate to true (count > 0)
	expr := &FuncCallExpr{Name: "b", Args: []Expr{}}
	val, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if !val.IsTruthy() {
		t.Error("expected true (count(deps) = 1 > 0)")
	}
}

func TestEval_UserFunctionChain(t *testing.T) {
	// Three-level chain: a() -> b() -> c()
	// a = "b()", b = "c()", c = "layers() > 0"
	fnA, _ := Parse("b()")
	fnB, _ := Parse("c()")
	fnC, _ := Parse("layers() > 0")
	userFuncs := map[string]Expr{
		"a": fnA,
		"b": fnB,
		"c": fnC,
	}
	ctx := EvalContext{
		Layers:        []Layer{{Name: "x"}},
		UserFunctions: userFuncs,
	}
	expr := &FuncCallExpr{Name: "a", Args: []Expr{}}
	val, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if !val.IsTruthy() {
		t.Error("expected true (layers() > 0)")
	}
}

func TestEval_UserFunctionUnknownName(t *testing.T) {
	// Calling a function that is not in UserFunctions and not a builtin
	ctx := EvalContext{
		UserFunctions: map[string]Expr{
			"existing": &NumberLiteral{Value: 1},
		},
	}
	expr := &FuncCallExpr{Name: "nonexistent", Args: []Expr{}}
	_, err := expr.Eval(ctx)
	if err == nil {
		t.Fatal("expected error for unknown function, got nil")
	}
	if !strings.Contains(err.Error(), "unknown function") {
		t.Errorf("error should mention 'unknown function', got: %v", err)
	}
}

func TestEval_UserFunctionNilMap(t *testing.T) {
	// When UserFunctions is nil, should fall through to builtins
	deps := []Dependency{
		{SourceFile: "domain/a.go", ResolvedLayer: "infra"},
	}
	layers := []Layer{
		{Name: "domain", Paths: []string{"domain/"}},
		{Name: "infra", Paths: []string{"infra/"}},
	}
	ctx := EvalContext{
		Deps:          deps,
		Layers:        layers,
		UserFunctions: nil,
	}
	// Builtin should still work
	expr := &FuncCallExpr{Name: "layers", Args: []Expr{}}
	val, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if val.Kind != ValueInt || val.Int != 2 {
		t.Errorf("expected layers=2, got %v", val)
	}
}

// ─── Integration: user function in rule-check pipeline ──────────────────────

func TestEvaluateRules_UserFunction(t *testing.T) {
	deps := []Dependency{
		{SourceFile: "domain/a.go", ResolvedLayer: "infra"},
		{SourceFile: "domain/b.go", ResolvedLayer: "infra"},
		{SourceFile: "domain/c.go", ResolvedLayer: "infra"},
		{SourceFile: "domain/d.go", ResolvedLayer: "infra"},
	}
	layers := []Layer{
		{Name: "domain", Paths: []string{"domain/"}},
		{Name: "infra", Paths: []string{"infra/"}},
	}

	// Define a user function and a rule that calls it
	config := Config{
		Version: "1.0.0",
		Layers:  layers,
		Functions: map[string]string{
			"too_many_deps": "count(deps(domain, infra)) > 3",
		},
		Rules: []Rule{
			{
				ID:       "R-USERFN",
				Check:    CheckExpr{Raw: "too_many_deps()"},
				Severity: SeverityError,
			},
		},
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("Config.Validate() error: %v", err)
	}

	// Compile rules
	for i := range config.Rules {
		if err := config.Rules[i].Validate(); err != nil {
			t.Fatalf("rule validation error: %v", err)
		}
	}

	violations := EvaluateRules(deps, config.Rules, layers, config.UserFunctions())
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].RuleID != "R-USERFN" {
		t.Errorf("expected rule ID R-USERFN, got %s", violations[0].RuleID)
	}
}

func TestEvaluateRules_UserFunctionNoViolation(t *testing.T) {
	deps := []Dependency{
		{SourceFile: "domain/a.go", ResolvedLayer: "infra"},
	}
	layers := []Layer{
		{Name: "domain", Paths: []string{"domain/"}},
		{Name: "infra", Paths: []string{"infra/"}},
	}

	config := Config{
		Version: "1.0.0",
		Layers:  layers,
		Functions: map[string]string{
			"too_many_deps": "count(deps(domain, infra)) > 3",
		},
		Rules: []Rule{
			{
				ID:       "R-USERFN",
				Check:    CheckExpr{Raw: "too_many_deps()"},
				Severity: SeverityError,
			},
		},
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("Config.Validate() error: %v", err)
	}
	for i := range config.Rules {
		if err := config.Rules[i].Validate(); err != nil {
			t.Fatalf("rule validation error: %v", err)
		}
	}

	violations := EvaluateRules(deps, config.Rules, layers, config.UserFunctions())
	if len(violations) != 0 {
		t.Fatalf("expected 0 violations, got %d", len(violations))
	}
}

func TestEvaluateRules_UserFunctionAllBuiltin(t *testing.T) {
	// Integration: user function uses all() builtin
	deps := []Dependency{
		{SourceFile: "domain/a.go", ResolvedLayer: "infra"},
	}
	layers := []Layer{
		{Name: "domain", Paths: []string{"domain/"}},
		{Name: "infra", Paths: []string{"infra/"}},
	}

	config := Config{
		Version: "1.0.0",
		Layers:  layers,
		Functions: map[string]string{
			"has_deps": "all(deps(domain, infra))",
		},
		Rules: []Rule{
			{
				ID:       "R-ALLFN",
				Check:    CheckExpr{Raw: "!has_deps()"},
				Severity: SeverityError,
			},
		},
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("config validate: %v", err)
	}
	for i := range config.Rules {
		if err := config.Rules[i].Validate(); err != nil {
			t.Fatalf("rule validate: %v", err)
		}
	}

	// has_deps() returns true (deps exist), so !has_deps() is false → no violation
	violations := EvaluateRules(deps, config.Rules, layers, config.UserFunctions())
	if len(violations) != 0 {
		t.Fatalf("expected 0 violations (deps exist, !has_deps=false), got %d", len(violations))
	}
}
