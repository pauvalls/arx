package domain

import (
	"testing"
)

func TestSchemaVersion_Validate(t *testing.T) {
	tests := []struct {
		name    string
		v       SchemaVersion
		wantErr bool
	}{
		{name: "valid 1.0", v: SchemaVersion{Major: 1, Minor: 0}, wantErr: false},
		{name: "valid 2.0", v: SchemaVersion{Major: 2, Minor: 0}, wantErr: false},
		{name: "valid 1.5", v: SchemaVersion{Major: 1, Minor: 5}, wantErr: false},
		{name: "major is 0", v: SchemaVersion{Major: 0, Minor: 1}, wantErr: true},
		{name: "negative major", v: SchemaVersion{Major: -1, Minor: 0}, wantErr: true},
		{name: "negative minor", v: SchemaVersion{Major: 1, Minor: -1}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.v.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("SchemaVersion{%d,%d}.Validate() error = %v, wantErr %v", tt.v.Major, tt.v.Minor, err, tt.wantErr)
			}
		})
	}
}

func TestSchemaVersion_String(t *testing.T) {
	tests := []struct {
		v    SchemaVersion
		want string
	}{
		{SchemaVersion{Major: 1, Minor: 0}, "1.0"},
		{SchemaVersion{Major: 2, Minor: 0}, "2.0"},
		{SchemaVersion{Major: 1, Minor: 5}, "1.5"},
		{SchemaVersion{Major: 10, Minor: 99}, "10.99"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.v.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseSchemaVersion(t *testing.T) {
	tests := []struct {
		input   string
		want    SchemaVersion
		wantErr bool
	}{
		{input: "1.0", want: SchemaVersion{Major: 1, Minor: 0}, wantErr: false},
		{input: "2.0", want: SchemaVersion{Major: 2, Minor: 0}, wantErr: false},
		{input: "1.5", want: SchemaVersion{Major: 1, Minor: 5}, wantErr: false},
		{input: "10.99", want: SchemaVersion{Major: 10, Minor: 99}, wantErr: false},
		{input: "", wantErr: true},
		{input: "1", want: SchemaVersion{Major: 1, Minor: 0}, wantErr: false},
		{input: "1.0.0", wantErr: true},
		{input: "abc", wantErr: true},
		{input: "1.x", wantErr: true},
		{input: "1.", wantErr: true},
		{input: ".1", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseSchemaVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSchemaVersion(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseSchemaVersion(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSchemaVersion_Compare(t *testing.T) {
	tests := []struct {
		a, b SchemaVersion
		want int
	}{
		{SchemaVersion{1, 0}, SchemaVersion{1, 0}, 0},
		{SchemaVersion{2, 0}, SchemaVersion{1, 0}, 1},
		{SchemaVersion{1, 0}, SchemaVersion{2, 0}, -1},
		{SchemaVersion{1, 5}, SchemaVersion{1, 0}, 1},
		{SchemaVersion{1, 0}, SchemaVersion{1, 5}, -1},
		{SchemaVersion{1, 0}, SchemaVersion{1, 0}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.a.String()+"_vs_"+tt.b.String(), func(t *testing.T) {
			got := tt.a.Compare(tt.b)
			if got != tt.want {
				t.Errorf("%v.Compare(%v) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestRegistry_RegisterAndResolve(t *testing.T) {
	r := NewRegistry()

	v1 := SchemaVersion{Major: 1, Minor: 0}
	v2 := SchemaVersion{Major: 2, Minor: 0}

	// Register direct v1→v2
	err := r.Register(Migration{From: v1, To: v2, Func: nopMigration})
	if err != nil {
		t.Fatalf("Register(v1→v2) unexpected error: %v", err)
	}

	// Resolve direct
	chain, err := r.Resolve(v1, v2)
	if err != nil {
		t.Fatalf("Resolve(v1, v2) unexpected error: %v", err)
	}
	if len(chain) != 1 {
		t.Fatalf("Resolve(v1, v2) chain length = %d, want 1", len(chain))
	}
}

func TestRegistry_MultiStepChain(t *testing.T) {
	r := NewRegistry()

	v1 := SchemaVersion{Major: 1, Minor: 0}
	v15 := SchemaVersion{Major: 1, Minor: 5}
	v2 := SchemaVersion{Major: 2, Minor: 0}

	err := r.Register(Migration{From: v1, To: v15, Func: nopMigration})
	if err != nil {
		t.Fatalf("Register(v1→v1.5) error: %v", err)
	}
	err = r.Register(Migration{From: v15, To: v2, Func: nopMigration})
	if err != nil {
		t.Fatalf("Register(v1.5→v2) error: %v", err)
	}

	chain, err := r.Resolve(v1, v2)
	if err != nil {
		t.Fatalf("Resolve(v1, v2) unexpected error: %v", err)
	}
	if len(chain) != 2 {
		t.Fatalf("Resolve(v1, v2) chain length = %d, want 2", len(chain))
	}
}

func TestRegistry_AmbiguousRejectedAtRegister(t *testing.T) {
	r := NewRegistry()

	v1 := SchemaVersion{Major: 1, Minor: 0}
	v2 := SchemaVersion{Major: 2, Minor: 0}
	v3 := SchemaVersion{Major: 3, Minor: 0}

	// Register v1→v2
	err := r.Register(Migration{From: v1, To: v2, Func: nopMigration})
	if err != nil {
		t.Fatalf("First Register(v1→v2) should succeed: %v", err)
	}

	// Register v1→v3 — ambiguous since v1 now has two outgoing paths
	err = r.Register(Migration{From: v1, To: v3, Func: nopMigration})
	if err == nil {
		t.Fatal("Second Register(v1→v3) should fail: ambiguous from v1")
	}
}

func TestRegistry_NoReverseMigration(t *testing.T) {
	r := NewRegistry()

	v1 := SchemaVersion{Major: 1, Minor: 0}
	v2 := SchemaVersion{Major: 2, Minor: 0}

	err := r.Register(Migration{From: v1, To: v2, Func: nopMigration})
	if err != nil {
		t.Fatalf("Register(v1→v2) error: %v", err)
	}

	_, err = r.Resolve(v2, v1)
	if err == nil {
		t.Fatal("Resolve(v2, v1) should error: no reverse migration")
	}
}

func TestRegistry_SourceVersion(t *testing.T) {
	r := NewRegistry()

	v1 := SchemaVersion{Major: 1, Minor: 0}
	v2 := SchemaVersion{Major: 2, Minor: 0}
	v3 := SchemaVersion{Major: 3, Minor: 0}

	r.Register(Migration{From: v1, To: v2, Func: nopMigration})
	r.Register(Migration{From: v2, To: v3, Func: nopMigration})

	got := r.SourceVersion()
	if got != v1 {
		t.Errorf("SourceVersion() = %v, want %v", got, v1)
	}
}

// nopMigration is a no-op migration function for testing.
func nopMigration(_ []byte) ([]byte, error) {
	return nil, nil
}
