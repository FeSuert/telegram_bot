package alarm

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Client struct {
	base string
	http *http.Client
}

func New(base string) *Client { return &Client{base: base, http: &http.Client{}} }

func (c *Client) Arm() error    { return c.post("/arm") }
func (c *Client) Disarm() error { return c.post("/disarm") }

func (c *Client) Status() (string, error) {
	var r struct {
		State string `json:"state"`
	}
	if err := c.get("/status", &r); err != nil {
		return "", err
	}
	return r.State, nil
}

func (c *Client) post(path string) error {
	res, err := c.http.Post(c.base+path, "application/json", nil)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Pi returned %s", res.Status)
	}
	return nil
}

func (c *Client) get(path string, v any) error {
	res, err := c.http.Get(c.base + path)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return json.NewDecoder(res.Body).Decode(v)
}