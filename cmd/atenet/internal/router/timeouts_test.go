// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package router

import (
	"testing"
	"time"
)

// The exported accessors memoize with sync.OnceValue, so the parsing behavior is
// tested through timeoutFromEnv rather than through resumeTimeout et al.
func TestTimeoutFromEnv(t *testing.T) {
	const key = "ATE_TEST_TIMEOUT_SECONDS"
	const def = 15 * time.Second

	tests := []struct {
		name string
		env  string // "" means leave the var unset
		want time.Duration
	}{
		{name: "unset uses default", env: "", want: def},
		{name: "valid override", env: "300", want: 300 * time.Second},
		{name: "one second", env: "1", want: time.Second},
		{name: "zero rejected", env: "0", want: def},
		{name: "negative rejected", env: "-1", want: def},
		{name: "garbage rejected", env: "300s", want: def},
		{name: "float rejected", env: "1.5", want: def},
		{name: "whitespace rejected", env: " 300 ", want: def},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(key, tt.env)
			if got := timeoutFromEnv(key, def); got != tt.want {
				t.Errorf("timeoutFromEnv(%q=%q) = %v, want %v", key, tt.env, got, tt.want)
			}
		})
	}
}

// Guards against a knob being wired to the wrong env var or default.
func TestTimeoutDefaults(t *testing.T) {
	tests := []struct {
		name string
		key  string
		def  time.Duration
	}{
		{name: "resume", key: resumeTimeoutEnv, def: defaultResumeTimeout},
		{name: "route", key: routeTimeoutEnv, def: defaultRouteTimeout},
		{name: "extproc", key: extProcTimeoutEnv, def: defaultExtProcTimeout},
	}
	seen := map[string]bool{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if seen[tt.key] {
				t.Errorf("env var %q is used by more than one timeout", tt.key)
			}
			seen[tt.key] = true

			t.Setenv(tt.key, "")
			if got := timeoutFromEnv(tt.key, tt.def); got != tt.def {
				t.Errorf("timeoutFromEnv(%q) = %v, want %v", tt.key, got, tt.def)
			}
			t.Setenv(tt.key, "300")
			if got, want := timeoutFromEnv(tt.key, tt.def), 300*time.Second; got != want {
				t.Errorf("timeoutFromEnv(%q=300) = %v, want %v", tt.key, got, want)
			}
		})
	}
}
