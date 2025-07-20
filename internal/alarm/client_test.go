package alarm

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

/* ----------------------------------------------------------------------
Status() ---------------------------------------------------------------------- */

func TestStatus_JsonAndPlain(t *testing.T) {
    cases := []struct {
        name string
        body string
        want string
    }{
        {"json armed", `{"state":"ARMED"}`, "ARMED"},
        {"json disarmed lower", `{"state":"disarmed"}`, "DISARMED"},
        {"plain upper", "ArMed", "ARMED"},
    }

    for _, tc := range cases {
        tc := tc
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()

            srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                // Status() always asks for /status
                if r.URL.Path != "/status" {
                    t.Fatalf("unexpected path: %s", r.URL.Path)
                }
                w.Write([]byte(tc.body))
            }))
            defer srv.Close()

            c := New(srv.URL) // default http.Client is fine
            got, err := c.Status()
            if err != nil {
                t.Fatalf("Status() error = %v", err)
            }
            if got != tc.want {
                t.Fatalf("Status() = %s, want %s", got, tc.want)
            }
        })
    }
}

// Ensure that completely invalid JSON gracefully falls back to plain text.
func TestStatus_InvalidJSONFallback(t *testing.T) {
    const body = `[not json` // makes json.Unmarshal fail → fallback branch

    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
        w.Write([]byte(body))
    }))
    defer srv.Close()

    c := New(srv.URL)
    got, err := c.Status()
    if err != nil {
        t.Fatalf("Status() error = %v", err)
    }
    if want := "[NOT JSON"; got != want {
        t.Fatalf("Status() = %q, want %q", got, want)
    }
}

/* ----------------------------------------------------------------------
Arm / Disarm (simpleGet success path) ------------------------------------ */

func TestArmAndDisarm_OK(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        switch r.URL.Path {
        case "/arm", "/disarm":
            w.WriteHeader(http.StatusOK)
        default:
            t.Fatalf("unexpected path: %s", r.URL.Path)
        }
    }))
    defer srv.Close()

    c := New(srv.URL)
    if err := c.Arm(); err != nil {
        t.Fatalf("Arm() error = %v", err)
    }
    if err := c.Disarm(); err != nil {
        t.Fatalf("Disarm() error = %v", err)
    }
}

/* ----------------------------------------------------------------------
ChangePIN ---------------------------------------------------------------- */

func TestChangePIN_SuccessAndEscaping(t *testing.T) {
    const pin = " 42&%$! " // contains characters that must be percent‑escaped

    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/change_pin" {
            t.Fatalf("unexpected path: %s", r.URL.Path)
        }
        // query is automatically unescaped by r.URL.Query()
        if got := r.URL.Query().Get("pin"); got != pin {
            t.Fatalf("pin not round‑tripped: got %q, want %q", got, pin)
        }
        w.WriteHeader(http.StatusOK)
    }))
    defer srv.Close()

    c := New(srv.URL)
    if err := c.ChangePIN(pin); err != nil {
        t.Fatalf("ChangePIN() error = %v", err)
    }
}

func TestChangePIN_ErrorFromServer(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
        w.WriteHeader(http.StatusInternalServerError)
    }))
    defer srv.Close()

    c := New(srv.URL)
    if err := c.ChangePIN("1234"); err == nil {
        t.Fatalf("expected error on 500 response")
    }
}

/* ----------------------------------------------------------------------
Error cases for simpleGet ------------------------------------------------- */

func TestSimpleGet_Non200(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
        w.WriteHeader(http.StatusTeapot) // 418 – definitely not OK
    }))
    defer srv.Close()

    c := New(srv.URL)
    if err := c.Arm(); err == nil {
        t.Fatalf("expected error when alarm returns non‑200")
    }
}

func TestSimpleGet_ClientError(t *testing.T) {
    c := New("http://example")

    // Inject an http.Client that always fails.
    c.http = &http.Client{Transport: roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
        return nil, fmt.Errorf("network kaboom")
    })}

    if err := c.Arm(); err == nil {
        t.Fatalf("expected error from underlying http.Client")
    }
}