package domain

import (
	"strings"
	"testing"
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
		Check:    "count(deps(domain, infra)) > 3",
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
		Check:    "count(invalid_syntax",
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
				Check:    "count(deps(domain, infra)) > 3",
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
				Check:    "count(deps(domain, infra)) > 3",
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
				Check:    "count(deps(domain, infra)) > 3",
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
				Check:    "count(deps(domain, infra)) > 3",
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
			Check:    "count(deps(domain, infra)) > 3",
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
			Check:    "count(deps(domain, infra)) > 3",
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
			Check:    "count(deps(domain, infra)) > 3",
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
			Check:       "count(deps(domain, infra)) > 3",
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
