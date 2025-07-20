package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"home-alarm-bot/internal/alarm"
	"home-alarm-bot/internal/state"
	"home-alarm-bot/internal/telegram"
)

/* ----------------------------------------------------------------------
   Helpers ---------------------------------------------------------------- */

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
    return f(r)
}

// startTestServer boots a real HTTP listener on a random port using the real
// Listen method, but all outbound HTTP from the Telegram client is stubbed so
// no traffic leaves the test process. It returns the base URL of the server as
// well as references to the store and bot so the caller can make assertions.
func startTestServer(t *testing.T) (base string, st *state.Store) {
    t.Helper()

    // Preserve the real transport so we can delegate localhost calls to it.
    realTransport := http.DefaultTransport

    // Stub all outbound HTTP *except* those that target our local test server.
    http.DefaultTransport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
        host := r.URL.Hostname()
        if host == "127.0.0.1" || host == "localhost" { // local server → real handler
            return realTransport.RoundTrip(r)
        }
        // Everything else (Telegram API, alarm client) is stubbed out.
        return &http.Response{
            StatusCode: http.StatusOK,
            Header:     make(http.Header),
            Body:       io.NopCloser(strings.NewReader(`{"ok":true,"result":[]}`)),
        }, nil
    })
    t.Cleanup(func() { http.DefaultTransport = realTransport })

    // Dependencies for telegram.Bot.
    st = state.New()
    tgAPI := telegram.NewAPI("TESTTOKEN")
    dummyAlarm := alarm.New("http://dummy")
    bot := telegram.NewBot(tgAPI, st, dummyAlarm)

    srv := New(st, bot)

    // Bind an available port first so we know the chosen address.
    l, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil {
        t.Fatalf("net.Listen: %v", err)
    }
    addr := l.Addr().String()
    l.Close()

    go func() { _ = srv.Listen(addr) }()

    // Wait until the /status endpoint responds (server ready).
    for i := 0; i < 50; i++ {
        if _, err := http.Get("http://" + addr + "/status"); err == nil {
            break
        }
        time.Sleep(10 * time.Millisecond)
    }

    return "http://" + addr, st
}

/* ----------------------------------------------------------------------
   Endpoint tests --------------------------------------------------------- */

func TestArmDisarmStatus(t *testing.T) {
    base, st := startTestServer(t)

    // 1 — Arm -----------------------------------------------------------------
    res, err := http.Get(base + "/arm")
    if err != nil || res.StatusCode != http.StatusOK {
        t.Fatalf("/arm failed: %v, status %d", err, res.StatusCode)
    }
    if got := st.Get(); got != state.Armed {
        t.Fatalf("store not Armed after /arm: %v", got)
    }

    // 2 — Disarm --------------------------------------------------------------
    res, err = http.Get(base + "/disarm")
    if err != nil || res.StatusCode != http.StatusOK {
        t.Fatalf("/disarm failed: %v, status %d", err, res.StatusCode)
    }
    if got := st.Get(); got != state.Disarmed {
        t.Fatalf("store not Disarmed after /disarm: %v", got)
    }

    // 3 — Status --------------------------------------------------------------
    res, err = http.Get(base + "/status")
    if err != nil {
        t.Fatalf("/status request error: %v", err)
    }
    defer res.Body.Close()
    var payload struct {
        State state.AlarmState `json:"state"`
    }
    if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
        t.Fatalf("decode json: %v", err)
    }
    if payload.State != state.Disarmed {
        t.Fatalf("/status state mismatch: got %v, want %v", payload.State, state.Disarmed)
    }
}

func TestSimpleBroadcastEndpoints(t *testing.T) {
    base, _ := startTestServer(t)
    paths := []string{"/alarm", "/success"}
    for _, p := range paths {
        res, err := http.Get(base + p)
        if err != nil || res.StatusCode != http.StatusOK {
            t.Fatalf("GET %s: err=%v status=%d", p, err, res.StatusCode)
        }
    }
}

/* ----------------------------------------------------------------------
   /video handler --------------------------------------------------------- */

func TestVideoHandler_Success(t *testing.T) {
    base, _ := startTestServer(t)

    body := &bytes.Buffer{}
    mw := multipart.NewWriter(body)
    fw, _ := mw.CreateFormFile("file", "clip.mp4")
    fw.Write([]byte("dummydata"))
    mw.Close()

    req, _ := http.NewRequest(http.MethodPost, base+"/video", body)
    req.Header.Set("Content-Type", mw.FormDataContentType())

    res, err := http.DefaultClient.Do(req)
    if err != nil {
        t.Fatalf("/video post: %v", err)
    }
    if res.StatusCode != http.StatusOK {
        t.Fatalf("/video status = %d, want 200", res.StatusCode)
    }
}

func TestVideoHandler_Errors(t *testing.T) {
    base, _ := startTestServer(t)

    // Case 1: invalid multipart form ----------------------------------------
    res, err := http.Post(base+"/video", "text/plain", strings.NewReader("oops"))
    if err != nil {
        t.Fatalf("invalid‑form POST: %v", err)
    }
    if res.StatusCode != http.StatusBadRequest {
        t.Fatalf("invalid‑form status = %d, want 400", res.StatusCode)
    }

    // Case 2: missing file field --------------------------------------------
    body := &bytes.Buffer{}
    mw := multipart.NewWriter(body)
    // create unrelated field so it *is* multipart but lacks "file"
    mw.WriteField("foo", "bar")
    mw.Close()

    req, _ := http.NewRequest(http.MethodPost, base+"/video", body)
    req.Header.Set("Content-Type", mw.FormDataContentType())

    res, err = http.DefaultClient.Do(req)
    if err != nil {
        t.Fatalf("missing‑file POST: %v", err)
    }
    if res.StatusCode != http.StatusBadRequest {
        t.Fatalf("missing‑file status = %d, want 400", res.StatusCode)
    }
}