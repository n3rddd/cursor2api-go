// Copyright (c) 2025-2026 libaxuan
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	envContent := `PORT=9000
DEBUG=true
API_KEY=test-key
MODELS=claude-sonnet-4.6
SYSTEM_PROMPT_INJECT=Test prompt
TIMEOUT=60
MAX_INPUT_LENGTH=10000
USER_AGENT=Test Agent
SCRIPT_URL=https://test.com/script.js`

	tmpDir := t.TempDir()
	envPath := tmpDir + "/.env"
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create test .env file: %v", err)
	}

	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			os.Setenv(parts[0], parts[1])
			t.Cleanup(func() { os.Unsetenv(parts[0]) })
		}
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	assertEqual(t, cfg.Port, 9000, "Port")
	assertEqual(t, cfg.Debug, true, "Debug")
	assertEqual(t, cfg.APIKey, "test-key", "APIKey")
	assertEqual(t, cfg.SystemPromptInject, "Test prompt", "SystemPromptInject")
	assertEqual(t, cfg.Timeout, 60, "Timeout")
	assertEqual(t, cfg.MaxInputLength, 10000, "MaxInputLength")
	assertEqual(t, cfg.FP.UserAgent, "Test Agent", "UserAgent")
	assertEqual(t, cfg.ScriptURL, "https://test.com/script.js", "ScriptURL")
}

func TestGetModels(t *testing.T) {
	cfg := &Config{Models: "claude-sonnet-4.6"}
	models := cfg.GetModels()
	expected := []string{"claude-sonnet-4.6", "claude-sonnet-4.6-thinking"}

	if len(models) != len(expected) {
		t.Errorf("GetModels() length = %d, want %d", len(models), len(expected))
	}
	for i, m := range models {
		if i < len(expected) && m != expected[i] {
			t.Errorf("GetModels()[%d] = %q, want %q", i, m, expected[i])
		}
	}
}

func TestIsValidModel(t *testing.T) {
	cfg := &Config{Models: "claude-sonnet-4.6"}

	tests := []struct {
		name  string
		model string
		want  bool
	}{
		{"valid base model", "claude-sonnet-4.6", true},
		{"valid thinking model", "claude-sonnet-4.6-thinking", true},
		{"invalid model", "unknown-model", false},
		{"empty model", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cfg.IsValidModel(tt.model); got != tt.want {
				t.Errorf("IsValidModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{"valid config", &Config{Port: 8000, APIKey: "k", Timeout: 30, MaxInputLength: 1000}, false},
		{"invalid port - zero", &Config{Port: 0, APIKey: "k", Timeout: 30, MaxInputLength: 1000}, true},
		{"invalid port - too high", &Config{Port: 70000, APIKey: "k", Timeout: 30, MaxInputLength: 1000}, true},
		{"missing API key", &Config{Port: 8000, APIKey: "", Timeout: 30, MaxInputLength: 1000}, true},
		{"invalid timeout", &Config{Port: 8000, APIKey: "k", Timeout: 0, MaxInputLength: 1000}, true},
		{"invalid max input length", &Config{Port: 8000, APIKey: "k", Timeout: 30, MaxInputLength: 0}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMaskedAPIKey(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"abcdef1234", "abcd****"},
		{"abc", "****"},
		{"", "****"},
	}
	for _, tt := range tests {
		cfg := &Config{APIKey: tt.key}
		if got := cfg.MaskedAPIKey(); got != tt.want {
			t.Errorf("MaskedAPIKey(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func assertEqual(t *testing.T, got, want interface{}, name string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}
