// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package router

import (
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"
)

// Resume / forwarding timeouts and their environment overrides.
//
// Resume-on-demand of a suspended actor performs a cold gVisor restore, which
// for a large snapshot (tens of MiB) can take tens of seconds. The default
// route / ext_proc / background-resume timeouts are sized for steady-state
// traffic and can cancel an in-flight restore, surfacing as a 504. These knobs
// let an operator running heavy actors raise the ceilings without a code change.
//
// Defaults are unchanged from prior behavior, so this is purely additive.
//
// Longer term, suspend-safe actor networking (agent-substrate/substrate#465)
// should remove most of the need to tune these.
const (
	resumeTimeoutEnv  = "ATE_RESUME_TIMEOUT_SECONDS"
	routeTimeoutEnv   = "ATE_ROUTE_TIMEOUT_SECONDS"
	extProcTimeoutEnv = "ATE_EXTPROC_TIMEOUT_SECONDS"

	defaultResumeTimeout  = 15 * time.Second
	defaultRouteTimeout   = 10 * time.Second
	defaultExtProcTimeout = 5 * time.Second
)

// timeoutFromEnv returns the duration from a whole-seconds env var, falling back
// to def when the var is unset or not a positive integer.
//
// A timeout of zero is not accepted: for these three knobs it would mean "no
// deadline at all" on a route, an ext_proc round-trip, or a background resume,
// which turns a stuck actor into a leaked goroutine or a hung connection rather
// than a 504. Unparseable and non-positive values are rejected with a warning
// and the default is used, so a typo degrades to prior behavior instead of
// silently removing a bound.
func timeoutFromEnv(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	secs, err := strconv.Atoi(v)
	if err != nil || secs <= 0 {
		slog.Warn("Ignoring invalid timeout override; expected a positive whole number of seconds",
			slog.String("env", key),
			slog.String("value", v),
			slog.Duration("using", def))
		return def
	}
	return time.Duration(secs) * time.Second
}

// The env vars are read once per process: they cannot change under a running
// container, xds.go re-reads the route / ext_proc timeouts on every xDS rebuild,
// and memoizing keeps an invalid-value warning to a single line.

// resumeTimeout bounds a detached, background resume operation.
var resumeTimeout = sync.OnceValue(func() time.Duration {
	return timeoutFromEnv(resumeTimeoutEnv, defaultResumeTimeout)
})

// routeTimeout bounds a forwarded upstream request (e.g. a long LLM turn).
var routeTimeout = sync.OnceValue(func() time.Duration {
	return timeoutFromEnv(routeTimeoutEnv, defaultRouteTimeout)
})

// extProcTimeout bounds the ext_proc round-trip, which can drive a cold restore.
var extProcTimeout = sync.OnceValue(func() time.Duration {
	return timeoutFromEnv(extProcTimeoutEnv, defaultExtProcTimeout)
})
