package main

import (
	"log"
	"net/http"
)

var __default_transport http.RoundTripper

func enableDebugTransport() {
	__default_transport = http.DefaultTransport

	http.DefaultTransport = &DebugTransport{
		Transport: http.DefaultTransport,
	}
}

func disableDebugTransport() {
	http.DefaultTransport = __default_transport
}

type DebugTransport struct {
	Transport http.RoundTripper
}

func (t *DebugTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	log.Printf("Send    ==>: %s %s", req.Method, req.URL)
	log.Printf("%v", req.Header)

	resp, err = t.Transport.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	log.Printf("Receive <==: %d %s (size=%d)", resp.StatusCode, resp.Request.URL, resp.ContentLength)

	return resp, err
}
