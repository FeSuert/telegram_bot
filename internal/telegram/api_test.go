package telegram

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func jsonResp(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": {"application/json"}},
	}
}

func TestAPI_SendMessage(t *testing.T) {
	var gotPath string
	var gotVals url.Values

	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		gotPath = r.URL.Path
		gotVals = r.PostForm
		return jsonResp(`{"ok":true}`), nil
	})

	api := NewAPI("DUMMY")
	api.client = &http.Client{Transport: rt}

	if err := api.SendMessage(42, "hello"); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	if !strings.HasSuffix(gotPath, "/sendMessage") {
		t.Fatalf("wrong endpoint: %s", gotPath)
	}
	if gotVals.Get("chat_id") != "42" || gotVals.Get("text") != "hello" {
		t.Fatalf("wrong parameters: %v", gotVals)
	}
}

func TestAPI_GetUpdates(t *testing.T) {
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if !strings.Contains(r.URL.Path, "/getUpdates") {
			t.Fatalf("wrong path %s", r.URL.Path)
		}
		return jsonResp(`{"ok":true,"result":[{"update_id":1}]}`), nil
	})

	api := NewAPI("DUMMY")
	api.client = &http.Client{Transport: rt}

	upd, err := api.GetUpdates(0)
	if err != nil {
		t.Fatalf("GetUpdates: %v", err)
	}
	if len(upd) != 1 || upd[0].UpdateID != 1 {
		t.Fatalf("unexpected result: %#v", upd)
	}
}

func TestAPI_SendVideo(t *testing.T) {
	// SendVideo uses http.DefaultClient, so we have to swap it.
	orig := http.DefaultClient
	defer func() { http.DefaultClient = orig }()

	called := false
	http.DefaultClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		called = true
		if !strings.HasSuffix(r.URL.Path, "/sendVideo") {
			t.Fatalf("wrong endpoint: %s", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
	})}

	api := NewAPI("TOKEN")
	if err := api.SendVideo(99, []byte("data"), "cap"); err != nil {
		t.Fatalf("SendVideo: %v", err)
	}
	if !called {
		t.Fatal("SendVideo never hit stub transport")
	}
}
