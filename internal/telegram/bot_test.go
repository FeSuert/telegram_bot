package telegram

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	alarmPkg "home-alarm-bot/internal/alarm"
	"home-alarm-bot/internal/state"
)

/* ---------- helpers ----------------------------------------------------- */

type fakeRoundTripper struct {
    mu   sync.Mutex
    reqs []*http.Request
}

func (f *fakeRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
    f.mu.Lock()
    f.reqs = append(f.reqs, r)
    f.mu.Unlock()

    return &http.Response{
        StatusCode: 200,
        Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
        Header:     http.Header{"Content-Type": {"application/json"}},
    }, nil
}

// newInstrumentedBot wires a Bot with a stub Telegram transport and a stub
// alarm server. It returns the bot, the transport (to inspect requests), a
// counter for /arm calls, and the store.
func newInstrumentedBot(t *testing.T) (*Bot, *fakeRoundTripper, *int32, *state.Store) {
    t.Helper()

    // Stub Telegram -------------------------------------------------------
    rt := &fakeRoundTripper{}
    api := NewAPI("TOKEN")
    api.client = &http.Client{Transport: rt}

    // Stub alarm server ---------------------------------------------------
    var armCalls int32
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        switch r.URL.Path {
        case "/arm":
            atomic.AddInt32(&armCalls, 1)
            w.WriteHeader(http.StatusOK)
        case "/disarm":
            w.WriteHeader(http.StatusOK)
        case "/status":
            w.Write([]byte(`{"state":"ARMED"}`))
        default:
            w.WriteHeader(http.StatusOK)
        }
    }))
    t.Cleanup(srv.Close)

    alarm := alarmPkg.New(srv.URL)

    st := state.New()
    bot := NewBot(api, st, alarm)

    // give the bot one chat
    bot.mu.Lock()
    bot.chats[1] = struct{}{}
    bot.mu.Unlock()

    return bot, rt, &armCalls, st
}

/* ------------------- API.SendVideo error path --------------------------- */

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAPI_SendVideoError(t *testing.T) {
    // Swap default client
    orig := http.DefaultClient
    defer func() { http.DefaultClient = orig }()

    http.DefaultClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
        return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader("fail"))}, nil
    })}

    api := NewAPI("BADTOKEN")
    if err := api.SendVideo(123, []byte("data"), "cap"); err == nil {
        t.Fatal("expected error on 500 response, got nil")
    }
}

/* ------------------- Bot.BroadcastVideo --------------------------------- */

type countingRT struct{ count int32 }

func (c *countingRT) RoundTrip(r *http.Request) (*http.Response, error) {
    if !strings.HasSuffix(r.URL.Path, "/sendVideo") {
        return nil, fmt.Errorf("unexpected path %s", r.URL.Path)
    }
    atomic.AddInt32(&c.count, 1)
    return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
}

func TestBroadcastVideo_MultiChats(t *testing.T) {
    orig := http.DefaultClient
    ctr := &countingRT{}
    http.DefaultClient = &http.Client{Transport: ctr}
    defer func() { http.DefaultClient = orig }()

    bot, _, _, _ := newInstrumentedBot(t)
    // add two more chats
    bot.mu.Lock()
    bot.chats[2], bot.chats[3] = struct{}{}, struct{}{}
    bot.mu.Unlock()

    if err := bot.BroadcastVideo(strings.NewReader("clip"), "cap"); err != nil {
        t.Fatalf("BroadcastVideo error: %v", err)
    }
    if got := atomic.LoadInt32(&ctr.count); got != 3 {
        t.Fatalf("expected 3 sendVideo calls, got %d", got)
    }
}

/* ------------------- Bot.Handle branches -------------------------------- */

func TestBot_HandleDisarm(t *testing.T) {
    bot, rt, _, st := newInstrumentedBot(t)

    bot.Handle(Update{Message: &Message{Text: "/disarm", Chat: Chat{ID: 1}}})

    if st.Get() != state.Disarmed {
        t.Fatalf("state = %s, want DISARMED", st.Get())
    }
    if len(rt.reqs) != 1 {
        t.Fatalf("expected 1 Telegram call, got %d", len(rt.reqs))
    }
}

func TestBot_HandleStatus(t *testing.T) {
    bot, rt, _, _ := newInstrumentedBot(t)

    bot.Handle(Update{Message: &Message{Text: "/status", Chat: Chat{ID: 1}}})

    if len(rt.reqs) != 1 {
        t.Fatalf("expected 1 Telegram call, got %d", len(rt.reqs))
    }
    raw, _ := io.ReadAll(rt.reqs[0].Body)
    vals, _ := url.ParseQuery(string(raw))
    txt := vals.Get("text")
    if !strings.Contains(strings.ToLower(txt), "armed") {
        t.Fatalf("status reply should mention Armed, got %q", txt)
    }
}
