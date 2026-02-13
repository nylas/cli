package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestSetFieldValue(t *testing.T) {
	tests := []struct {
		name      string
		fieldType reflect.Kind
		value     string
		setupFunc func() reflect.Value     // Function to create the field
		checkFunc func(reflect.Value) bool // Function to verify the field value
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "set string field",
			fieldType: reflect.String,
			value:     "test_value",
			setupFunc: func() reflect.Value {
				var s string
				return reflect.ValueOf(&s).Elem()
			},
			checkFunc: func(v reflect.Value) bool {
				return v.String() == "test_value"
			},
		},
		{
			name:      "set int field",
			fieldType: reflect.Int,
			value:     "42",
			setupFunc: func() reflect.Value {
				var i int
				return reflect.ValueOf(&i).Elem()
			},
			checkFunc: func(v reflect.Value) bool {
				return v.Int() == 42
			},
		},
		{
			name:      "set int64 field",
			fieldType: reflect.Int64,
			value:     "9223372036854775807",
			setupFunc: func() reflect.Value {
				var i int64
				return reflect.ValueOf(&i).Elem()
			},
			checkFunc: func(v reflect.Value) bool {
				return v.Int() == 9223372036854775807
			},
		},
		{
			name:      "set bool field true",
			fieldType: reflect.Bool,
			value:     "true",
			setupFunc: func() reflect.Value {
				var b bool
				return reflect.ValueOf(&b).Elem()
			},
			checkFunc: func(v reflect.Value) bool {
				return v.Bool() == true
			},
		},
		{
			name:      "set bool field false",
			fieldType: reflect.Bool,
			value:     "false",
			setupFunc: func() reflect.Value {
				var b bool
				return reflect.ValueOf(&b).Elem()
			},
			checkFunc: func(v reflect.Value) bool {
				return v.Bool() == false
			},
		},
		{
			name:      "invalid int value",
			fieldType: reflect.Int,
			value:     "not_a_number",
			setupFunc: func() reflect.Value {
				var i int
				return reflect.ValueOf(&i).Elem()
			},
			wantErr: true,
			errMsg:  "invalid integer value",
		},
		{
			name:      "invalid bool value",
			fieldType: reflect.Bool,
			value:     "not_a_bool",
			setupFunc: func() reflect.Value {
				var b bool
				return reflect.ValueOf(&b).Elem()
			},
			wantErr: true,
			errMsg:  "invalid boolean value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := tt.setupFunc()
			err := setFieldValue(field, tt.value)

			if tt.wantErr {
				if err == nil {
					t.Errorf("setFieldValue() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("setFieldValue() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("setFieldValue() unexpected error: %v", err)
				return
			}

			if tt.checkFunc != nil && !tt.checkFunc(field) {
				t.Errorf("setFieldValue() did not set correct value, got %v", field.Interface())
			}
		})
	}
}

func TestSetConfigValue(t *testing.T) {
	type APIConfig struct {
		Timeout string
		BaseUrl string
		Port    int
	}

	type OutputConfig struct {
		Format string
		Color  string
		Pretty bool
	}

	type TestConfig struct {
		Region       string
		DefaultGrant string
		API          *APIConfig
		Output       OutputConfig
	}

	tests := []struct {
		name      string
		cfg       any
		key       string
		value     string
		checkFunc func(*TestConfig) bool
		wantErr   bool
		errMsg    string
	}{
		{
			name:  "set top-level string field",
			cfg:   &TestConfig{},
			key:   "region",
			value: "eu",
			checkFunc: func(c *TestConfig) bool {
				return c.Region == "eu"
			},
		},
		{
			name:  "set snake_case field",
			cfg:   &TestConfig{},
			key:   "default_grant",
			value: "grant_abc123",
			checkFunc: func(c *TestConfig) bool {
				return c.DefaultGrant == "grant_abc123"
			},
		},
		{
			name:  "set nested field in nil pointer (should initialize)",
			cfg:   &TestConfig{},
			key:   "api.timeout",
			value: "120s",
			checkFunc: func(c *TestConfig) bool {
				return c.API != nil && c.API.Timeout == "120s"
			},
		},
		{
			name: "set nested field in existing pointer",
			cfg: &TestConfig{
				API: &APIConfig{
					BaseUrl: "existing",
				},
			},
			key:   "api.timeout",
			value: "90s",
			checkFunc: func(c *TestConfig) bool {
				return c.API.Timeout == "90s" && c.API.BaseUrl == "existing"
			},
		},
		{
			name:  "set nested int field",
			cfg:   &TestConfig{},
			key:   "api.port",
			value: "8080",
			checkFunc: func(c *TestConfig) bool {
				return c.API != nil && c.API.Port == 8080
			},
		},
		{
			name:  "set nested bool field",
			cfg:   &TestConfig{},
			key:   "output.pretty",
			value: "true",
			checkFunc: func(c *TestConfig) bool {
				return c.Output.Pretty == true
			},
		},
		{
			name:  "set nested field in non-pointer struct",
			cfg:   &TestConfig{},
			key:   "output.format",
			value: "json",
			checkFunc: func(c *TestConfig) bool {
				return c.Output.Format == "json"
			},
		},
		{
			name:    "unknown field returns error",
			cfg:     &TestConfig{},
			key:     "unknown_field",
			value:   "value",
			wantErr: true,
			errMsg:  "unknown config key",
		},
		{
			name:    "invalid nested key returns error",
			cfg:     &TestConfig{Region: "us"},
			key:     "region.nested",
			value:   "value",
			wantErr: true,
			errMsg:  "cannot access field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := setConfigValue(tt.cfg, tt.key, tt.value)

			if tt.wantErr {
				if err == nil {
					t.Errorf("setConfigValue() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("setConfigValue() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("setConfigValue() unexpected error: %v", err)
				return
			}

			if tt.checkFunc != nil {
				testCfg, ok := tt.cfg.(*TestConfig)
				if !ok {
					t.Fatal("test config is not *TestConfig")
				}
				if !tt.checkFunc(testCfg) {
					t.Errorf("setConfigValue() did not set expected value")
				}
			}
		})
	}
}

func TestSetConfigValue_RoundTrip(t *testing.T) {
	// Test that we can set and then get the same value
	type Config struct {
		Field string
	}

	cfg := &Config{}
	key := "field"
	value := "test_value"

	// Set the value
	err := setConfigValue(cfg, key, value)
	if err != nil {
		t.Fatalf("setConfigValue() error: %v", err)
	}

	// Get the value
	got, err := getConfigValue(cfg, key)
	if err != nil {
		t.Fatalf("getConfigValue() error: %v", err)
	}

	if got != value {
		t.Errorf("round trip failed: set %q, got %q", value, got)
	}
}
