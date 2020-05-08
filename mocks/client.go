package mocks

import "net/http"

// Client is the mock client struct
type Client struct {
    DoFunc func(req *http.Request) (*http.Response, error)
}

var (
    // GetDoFunc fetches the mock client's `Do` func
    GetDoFunc func(req *http.Request) (*http.Response, error)
)

func (c *Client) Do(req *http.Request) (*http.Response, error) {
    return GetDoFunc(req)
}