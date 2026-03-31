package config

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

type testConfig struct {
	StringVal   string        `flag:"string-val,s" env:"STRING_VAL" default:"default-string" desc:"a string value"`
	IntVal      int           `flag:"int-val,i" env:"INT_VAL" default:"42" desc:"an int value"`
	BoolVal     bool          `flag:"bool-val,b" env:"BOOL_VAL" default:"true" desc:"a bool value"`
	DurationVal time.Duration `flag:"duration-val,d" env:"DURATION_VAL" default:"30s" desc:"a duration value"`
	NoEnv       string        `flag:"no-env" default:"no-env-default" desc:"field without env var"`
	NoDefault   string        `flag:"no-default" env:"NO_DEFAULT" desc:"field without default"`
	Skipped     string        // no flag tag, should be skipped
}

func newTestCommand() *cobra.Command {
	return &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
}

func TestRegisterFlags(t *testing.T) {
	cmd := newTestCommand()
	cfg := &testConfig{}

	if err := RegisterFlags(cmd, cfg); err != nil {
		t.Fatalf("RegisterFlags failed: %v", err)
	}

	// Verify flags were registered
	tests := []struct {
		name      string
		shorthand string
	}{
		{"string-val", "s"},
		{"int-val", "i"},
		{"bool-val", "b"},
		{"duration-val", "d"},
		{"no-env", ""},
		{"no-default", ""},
	}

	for _, tt := range tests {
		flag := cmd.Flags().Lookup(tt.name)
		if flag == nil {
			t.Errorf("flag %q was not registered", tt.name)
			continue
		}
		if tt.shorthand != "" && flag.Shorthand != tt.shorthand {
			t.Errorf("flag %q shorthand = %q, want %q", tt.name, flag.Shorthand, tt.shorthand)
		}
	}
}

func TestRegisterFlags_InvalidInput(t *testing.T) {
	cmd := newTestCommand()

	// Non-pointer should fail
	if err := RegisterFlags(cmd, testConfig{}); err == nil {
		t.Error("RegisterFlags should fail for non-pointer")
	}

	// Pointer to non-struct should fail
	str := "test"
	if err := RegisterFlags(cmd, &str); err == nil {
		t.Error("RegisterFlags should fail for pointer to non-struct")
	}
}

func TestRegisterFlags_UnsupportedType(t *testing.T) {
	type badConfig struct {
		BadField []string `flag:"bad-field" desc:"slice type not supported"`
	}

	cmd := newTestCommand()
	cfg := &badConfig{}

	if err := RegisterFlags(cmd, cfg); err == nil {
		t.Error("RegisterFlags should fail for unsupported field type")
	}
}

func TestLoad_Defaults(t *testing.T) {
	cmd := newTestCommand()
	cfg := &testConfig{}

	if err := RegisterFlags(cmd, cfg); err != nil {
		t.Fatalf("RegisterFlags failed: %v", err)
	}

	// Execute command to initialize flags
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute failed: %v", err)
	}

	if err := Load(cmd, cfg); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.StringVal != "default-string" {
		t.Errorf("StringVal = %q, want %q", cfg.StringVal, "default-string")
	}
	if cfg.IntVal != 42 {
		t.Errorf("IntVal = %d, want %d", cfg.IntVal, 42)
	}
	if cfg.BoolVal != true {
		t.Errorf("BoolVal = %v, want %v", cfg.BoolVal, true)
	}
	if cfg.DurationVal != 30*time.Second {
		t.Errorf("DurationVal = %v, want %v", cfg.DurationVal, 30*time.Second)
	}
	if cfg.NoEnv != "no-env-default" {
		t.Errorf("NoEnv = %q, want %q", cfg.NoEnv, "no-env-default")
	}
	if cfg.NoDefault != "" {
		t.Errorf("NoDefault = %q, want %q", cfg.NoDefault, "")
	}
}

func TestLoad_FlagOverride(t *testing.T) {
	cmd := newTestCommand()
	cfg := &testConfig{}

	if err := RegisterFlags(cmd, cfg); err != nil {
		t.Fatalf("RegisterFlags failed: %v", err)
	}

	cmd.SetArgs([]string{
		"--string-val=flag-override",
		"--int-val=99",
		"--bool-val=false",
		"--duration-val=5m",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute failed: %v", err)
	}

	if err := Load(cmd, cfg); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.StringVal != "flag-override" {
		t.Errorf("StringVal = %q, want %q", cfg.StringVal, "flag-override")
	}
	if cfg.IntVal != 99 {
		t.Errorf("IntVal = %d, want %d", cfg.IntVal, 99)
	}
	if cfg.BoolVal != false {
		t.Errorf("BoolVal = %v, want %v", cfg.BoolVal, false)
	}
	if cfg.DurationVal != 5*time.Minute {
		t.Errorf("DurationVal = %v, want %v", cfg.DurationVal, 5*time.Minute)
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	// Set environment variables
	if err := os.Setenv("STRING_VAL", "env-override"); err != nil {
		t.Fatalf("os.Setenv: %v", err)
	}
	if err := os.Setenv("INT_VAL", "123"); err != nil {
		t.Fatalf("os.Setenv: %v", err)
	}
	if err := os.Setenv("BOOL_VAL", "false"); err != nil {
		t.Fatalf("os.Setenv: %v", err)
	}
	if err := os.Setenv("DURATION_VAL", "2h"); err != nil {
		t.Fatalf("os.Setenv: %v", err)
	}
	defer func() {
		os.Unsetenv("STRING_VAL")
		os.Unsetenv("INT_VAL")
		os.Unsetenv("BOOL_VAL")
		os.Unsetenv("DURATION_VAL")
	}()

	cmd := newTestCommand()
	cfg := &testConfig{}

	if err := RegisterFlags(cmd, cfg); err != nil {
		t.Fatalf("RegisterFlags failed: %v", err)
	}

	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute failed: %v", err)
	}

	if err := Load(cmd, cfg); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.StringVal != "env-override" {
		t.Errorf("StringVal = %q, want %q", cfg.StringVal, "env-override")
	}
	if cfg.IntVal != 123 {
		t.Errorf("IntVal = %d, want %d", cfg.IntVal, 123)
	}
	if cfg.BoolVal != false {
		t.Errorf("BoolVal = %v, want %v", cfg.BoolVal, false)
	}
	if cfg.DurationVal != 2*time.Hour {
		t.Errorf("DurationVal = %v, want %v", cfg.DurationVal, 2*time.Hour)
	}
}

func TestLoad_FlagTakesPrecedenceOverEnv(t *testing.T) {
	// Set environment variable
	if err := os.Setenv("STRING_VAL", "env-value"); err != nil {
		t.Fatalf("os.Setenv: %v", err)
	}
	defer os.Unsetenv("STRING_VAL")

	cmd := newTestCommand()
	cfg := &testConfig{}

	if err := RegisterFlags(cmd, cfg); err != nil {
		t.Fatalf("RegisterFlags failed: %v", err)
	}

	// Flag should override env
	cmd.SetArgs([]string{"--string-val=flag-value"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute failed: %v", err)
	}

	if err := Load(cmd, cfg); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.StringVal != "flag-value" {
		t.Errorf("StringVal = %q, want %q (flag should override env)", cfg.StringVal, "flag-value")
	}
}

func TestLoad_InvalidEnvValue(t *testing.T) {
	tests := []struct {
		name   string
		envKey string
		envVal string
	}{
		{"invalid int", "INT_VAL", "not-a-number"},
		{"invalid bool", "BOOL_VAL", "not-a-bool"},
		{"invalid duration", "DURATION_VAL", "not-a-duration"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.Setenv(tt.envKey, tt.envVal); err != nil {
				t.Fatalf("os.Setenv: %v", err)
			}
			defer os.Unsetenv(tt.envKey)

			cmd := newTestCommand()
			cfg := &testConfig{}

			if err := RegisterFlags(cmd, cfg); err != nil {
				t.Fatalf("RegisterFlags failed: %v", err)
			}

			cmd.SetArgs([]string{})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("cmd.Execute failed: %v", err)
			}

			if err := Load(cmd, cfg); err == nil {
				t.Errorf("Load should fail for invalid env value %s=%s", tt.envKey, tt.envVal)
			}
		})
	}
}

func TestLoad_InvalidInput(t *testing.T) {
	cmd := newTestCommand()

	// Non-pointer should fail
	if err := Load(cmd, testConfig{}); err == nil {
		t.Error("Load should fail for non-pointer")
	}

	// Pointer to non-struct should fail
	str := "test"
	if err := Load(cmd, &str); err == nil {
		t.Error("Load should fail for pointer to non-struct")
	}
}

func TestShorthandFlags(t *testing.T) {
	cmd := newTestCommand()
	cfg := &testConfig{}

	if err := RegisterFlags(cmd, cfg); err != nil {
		t.Fatalf("RegisterFlags failed: %v", err)
	}

	// Use shorthand flags
	cmd.SetArgs([]string{"-s", "short-string", "-i", "77"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute failed: %v", err)
	}

	if err := Load(cmd, cfg); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.StringVal != "short-string" {
		t.Errorf("StringVal = %q, want %q", cfg.StringVal, "short-string")
	}
	if cfg.IntVal != 77 {
		t.Errorf("IntVal = %d, want %d", cfg.IntVal, 77)
	}
}

func TestInvalidDefaultValues(t *testing.T) {
	t.Run("invalid int default", func(t *testing.T) {
		type badIntDefault struct {
			Val int `flag:"val" default:"not-an-int"`
		}
		cmd := newTestCommand()
		if err := RegisterFlags(cmd, &badIntDefault{}); err == nil {
			t.Error("RegisterFlags should fail for invalid int default")
		}
	})

	t.Run("invalid bool default", func(t *testing.T) {
		type badBoolDefault struct {
			Val bool `flag:"val" default:"not-a-bool"`
		}
		cmd := newTestCommand()
		if err := RegisterFlags(cmd, &badBoolDefault{}); err == nil {
			t.Error("RegisterFlags should fail for invalid bool default")
		}
	})

	t.Run("invalid duration default", func(t *testing.T) {
		type badDurationDefault struct {
			Val time.Duration `flag:"val" default:"not-a-duration"`
		}
		cmd := newTestCommand()
		if err := RegisterFlags(cmd, &badDurationDefault{}); err == nil {
			t.Error("RegisterFlags should fail for invalid duration default")
		}
	})
}
