package handler

import (
	"io"
	"net/http"
)

// Transport implements http.RoundTripper
type Transport struct{}

// RoundTrip makes roundtrip using http-wasm host
func (r *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	return Send(req)
}

// Send sends an HTTP request and return the HTTP response.
func Send(req *http.Request) (*http.Response, error) {
	return send(req)
}

func NewClient() *http.Client {
	return &http.Client{
		Transport: &Transport{},
	}
}

func send(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	return Host.HTTPRequest(req.Method, req.URL.String(), string(body))
}
