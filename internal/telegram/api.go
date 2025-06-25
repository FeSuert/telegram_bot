package telegram

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type API struct {
	token  string
	client *http.Client
}

func NewAPI(token string) *API {
	return &API{
		token:  token,
		client: &http.Client{},
	}
}

func (t *API) endpoint(method string) string {
	return fmt.Sprintf("https://api.telegram.org/bot%s/%s", t.token, method)
}

// GET https://api.telegram.org/bot<TOKEN>/getUpdates?offset=…
func (t *API) GetUpdates(offset int) ([]Update, error) {
	v := url.Values{}
	v.Set("offset", fmt.Sprint(offset))
	v.Set("timeout", "60")
	resp, err := t.client.Get(t.endpoint("getUpdates") + "?" + v.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var r struct {
		Ok     bool     `json:"ok"`
		Result []Update `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	return r.Result, nil
}

// POST https://api.telegram.org/bot<TOKEN>/sendMessage
func (t *API) SendMessage(chatID int64, text string) error {
	payload := url.Values{}
	payload.Set("chat_id", fmt.Sprint(chatID))
	payload.Set("text", text)
	resp, err := t.client.PostForm(t.endpoint("sendMessage"), payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var pr struct {
		Ok bool `json:"ok"`
	}
	return json.NewDecoder(resp.Body).Decode(&pr)
}