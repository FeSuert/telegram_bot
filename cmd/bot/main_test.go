package main

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

/* ----------------------------------------------------------------------
   Tests for mustEnv ------------------------------------------------------ */

// TestMustEnvValue verifies that mustEnv returns the correct value when the
// environment variable is present.
func TestMustEnvValue(t *testing.T) {
    const key, val = "TEST_ENV_VALUE", "hello-world"

    t.Setenv(key, val)
    if got := mustEnv(key); got != val {
        t.Fatalf("mustEnv(%q) = %q, want %q", key, got, val)
    }
}

// TestMustEnvMissing checks that mustEnv terminates the process (log.Fatalf →
// os.Exit(1)) when the variable is missing. We fork a helper process to observe
// the exit status without killing the test runner itself.
func TestMustEnvMissing(t *testing.T) {
    // When this env var is set, we are running inside the helper subprocess.
    if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
        mustEnv("SOME_UNSET_ENV")
        return // unreachable — mustEnv should have exited.
    }

    cmd := exec.Command(os.Args[0], "-test.run=TestMustEnvMissing")
    cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")

    err := cmd.Run()
    if err == nil {
        t.Fatalf("expected process to exit with a non‑zero status")
    }

    // Confirm we really observed a non‑zero exit code.
    if ee, ok := err.(*exec.ExitError); ok {
        if code := ee.ExitCode(); code == 0 {
            t.Fatalf("unexpected zero exit code from helper process")
        }
    } else {
        t.Fatalf("unexpected error type: %v", err)
    }
}

/* ----------------------------------------------------------------------
   Integration‑lite test for main() -------------------------------------- */

// roundTripperFunc is an adapter to let ordinary functions be used as HTTP
// transports.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
    return f(r)
}

// TestMainFunction executes the real main() in a controlled environment to
// cover the setup logic. We stub out all outbound HTTP so no network traffic
// occurs and we abort the long‑poll loop after a couple of calls via panic &
// recover. This way we execute the majority of the code paths without hanging
// the test.
func TestMainFunction(t *testing.T) {
    // Provide the env vars that main() expects.
    t.Setenv("SERVER_BASE_URL", "http://alarm.local")
    t.Setenv("BOT_TOKEN", "TESTTOKEN")

    // Replace the global default transport so every outbound request is served
    // by our stub (both Telegram and alarm client inherit it via http.Client{}).
    var calls int32
    stub := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
        // After a couple of GetUpdates calls, trigger a panic to unwind the
        // goroutine running main().
        if strings.Contains(r.URL.Path, "getUpdates") {
            if atomic.AddInt32(&calls, 1) >= 3 {
                panic("done")
            }
        }
        return &http.Response{
            StatusCode: http.StatusOK,
            Header:     make(http.Header),
            Body:       io.NopCloser(strings.NewReader(`{"ok":true,"result":[]}`)),
        }, nil
    })

    oldTransport := http.DefaultTransport
    http.DefaultTransport = stub
    t.Cleanup(func() { http.DefaultTransport = oldTransport })

    // Run main() in its own goroutine so we can bail out once enough code has
    // been executed.
    done := make(chan struct{})
    go func() {
        defer func() { _ = recover(); close(done) }()
        main()
    }()

    // Give the goroutine a brief moment to execute the startup sequence and a
    // few loop iterations. This keeps the test quick while still exercising
    // nearly every statement in main.go.
    time.Sleep(100 * time.Millisecond)
    <-done // ensure the goroutine terminated via our panic
}
