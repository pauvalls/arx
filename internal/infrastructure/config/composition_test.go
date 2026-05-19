package config

import (
	"strings"
	"testing"
)

func TestDeepMerge_SimpleOverride(t *testing.T) {
	base := []byte("version: \"1.0\"\nlayers:\n  - name: domain\n    paths: [\"./domain\"]\n")
	override := []byte("version: \"2.0\"\n")

	result, err := DeepMerge(base, override)
	if err != nil {
		t.Fatalf("DeepMerge() error = %v", err)
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, "version: \"2.0\"") {
		t.Errorf("DeepMerge() should use override version.\n got: %s", resultStr)
	}
	if !strings.Contains(resultStr, "name: domain") {
		t.Errorf("DeepMerge() should keep base layers.\n got: %s", resultStr)
	}
}

func TestDeepMerge_NestedMerge(t *testing.T) {
	base := []byte("logging:\n  level: debug\n  format: json\n")
	override := []byte("logging:\n  level: error\n")

	result, err := DeepMerge(base, override)
	if err != nil {
		t.Fatalf("DeepMerge() error = %v", err)
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, "level: error") {
		t.Errorf("DeepMerge() nested: should use override level.\n got: %s", resultStr)
	}
	if !strings.Contains(resultStr, "format: json") {
		t.Errorf("DeepMerge() nested: should keep base format.\n got: %s", resultStr)
	}
}

func TestDeepMerge_ArrayReplacement(t *testing.T) {
	base := []byte("rules:\n  - id: rule1\n  - id: rule2\n")
	override := []byte("rules:\n  - id: rule3\n")

	result, err := DeepMerge(base, override)
	if err != nil {
		t.Fatalf("DeepMerge() error = %v", err)
	}

	resultStr := string(result)
	if strings.Contains(resultStr, "rule1") || strings.Contains(resultStr, "rule2") {
		t.Errorf("DeepMerge() array should be entirely replaced, base values remain.\n got: %s", resultStr)
	}
	if !strings.Contains(resultStr, "rule3") {
		t.Errorf("DeepMerge() array should have override value.\n got: %s", resultStr)
	}
}

func TestDeepMerge_NoOverride(t *testing.T) {
	base := []byte("version: \"1.0\"\nlayers:\n  - name: domain\n")

	result, err := DeepMerge(base, nil)
	if err != nil {
		t.Fatalf("DeepMerge() nil override error = %v", err)
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, "version: \"1.0\"") {
		t.Errorf("DeepMerge() nil override should keep base.\n got: %s", resultStr)
	}
	if !strings.Contains(resultStr, "name: domain") {
		t.Errorf("DeepMerge() nil override should keep base layers.\n got: %s", resultStr)
	}
}

func TestDeepMerge_EmptyOverride(t *testing.T) {
	base := []byte("version: \"1.0\"")
	override := []byte("{}\n")

	result, err := DeepMerge(base, override)
	if err != nil {
		t.Fatalf("DeepMerge() empty override error = %v", err)
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, "version:") {
		t.Errorf("DeepMerge() empty override should keep base.\n got: %s", resultStr)
	}
}

func TestDeepMerge_MultipleKeys(t *testing.T) {
	base := []byte("a: 1\nb: 2\nc: 3\n")
	override := []byte("b: 20\nd: 4\n")

	result, err := DeepMerge(base, override)
	if err != nil {
		t.Fatalf("DeepMerge() error = %v", err)
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, "a: 1") {
		t.Errorf("DeepMerge() should keep base key 'a'.\n got: %s", resultStr)
	}
	if !strings.Contains(resultStr, "b: 20") {
		t.Errorf("DeepMerge() should have override value for 'b'.\n got: %s", resultStr)
	}
	if !strings.Contains(resultStr, "c: 3") {
		t.Errorf("DeepMerge() should keep base key 'c'.\n got: %s", resultStr)
	}
	if !strings.Contains(resultStr, "d: 4") {
		t.Errorf("DeepMerge() should include new key 'd' from override.\n got: %s", resultStr)
	}
	// Check that old 'b: 2' was replaced (b: 2 should not appear as its own line)
	if strings.Contains(resultStr, "\nb: 2\n") || strings.HasPrefix(resultStr, "b: 2\n") {
		t.Errorf("DeepMerge() should override 'b'.\n got: %s", resultStr)
	}
}

func TestDeepMerge_DeeplyNested(t *testing.T) {
	base := []byte("a:\n  b:\n    c: 1\n    d: 2\n  e: 3\n")
	override := []byte("a:\n  b:\n    d: 20\n  f: 4\n")

	result, err := DeepMerge(base, override)
	if err != nil {
		t.Fatalf("DeepMerge() error = %v", err)
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, "c: 1") {
		t.Errorf("DeepMerge() deep nested: should keep 'a.b.c'.\n got: %s", resultStr)
	}
	if !strings.Contains(resultStr, "d: 20") {
		t.Errorf("DeepMerge() deep nested: should override 'a.b.d'.\n got: %s", resultStr)
	}
	if strings.Contains(resultStr, "\nd: 2\n") || strings.HasPrefix(resultStr, "d: 2\n") {
		t.Errorf("DeepMerge() deep nested: 'a.b.d' should be overridden.\n got: %s", resultStr)
	}
	if !strings.Contains(resultStr, "e: 3") {
		t.Errorf("DeepMerge() deep nested: should keep 'a.e'.\n got: %s", resultStr)
	}
	if !strings.Contains(resultStr, "f: 4") {
		t.Errorf("DeepMerge() deep nested: should include new key 'a.f'.\n got: %s", resultStr)
	}
}
