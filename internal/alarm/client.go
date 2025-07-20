package alarm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
    base string
    http *http.Client
}

func New(base string) *Client {
    return &Client{base: strings.TrimRight(base, "/"), http: &http.Client{}}
}

/* state-changing commands use GET */

func (c *Client) Arm() error    { return c.simpleGet("/arm") }
func (c *Client) Disarm() error { return c.simpleGet("/disarm") }

func (c *Client) simpleGet(path string) error {
    res, err := c.http.Get(c.base + path)
    if err != nil {
        return err
    }
    if res.StatusCode != http.StatusOK {
        return fmt.Errorf("alarm returned %s", res.Status)
    }
    return nil
}

/* status endpoint */

func (c *Client) Status() (string, error) {
    res, err := c.http.Get(c.base + "/status")
    if err != nil {
        return "", err
    }
    defer res.Body.Close()

    // read the whole payload once
    data, err := io.ReadAll(res.Body)
    if err != nil {
        return "", err
    }

    // first try JSON: {"state":"ARMED"} or {"state":"DISARMED"}
    var r struct {
        State string `json:"state"`
    }
    if err := json.Unmarshal(data, &r); err == nil && r.State != "" {
        return strings.ToUpper(r.State), nil
    }

    // fall back to plain text
    return strings.ToUpper(strings.TrimSpace(string(data))), nil
}

func (c *Client) ChangePIN(pin string) error {
    u := fmt.Sprintf("%s/change_pin?pin=%s", c.base, url.QueryEscape(pin))
    res, err := c.http.Get(u)
    if err != nil {
        return err
    }
    if res.StatusCode != http.StatusOK {
        return fmt.Errorf("alarm returned %s", res.Status)
    }
    return nil
}