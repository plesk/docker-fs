package dockerfs

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
)

type httpClient interface {
	Get(string) (*http.Response, error)
	Head(string) (*http.Response, error)
}

var _ = httpClient((*client)(nil))

type client struct {
	addr string
	cl   *http.Client
}

func NewClient(addr string) (*client, error) {
	unixPrefix := "unix:"
	if strings.HasPrefix(addr, unixPrefix) {
		addr = addr[len(unixPrefix):]
		log.Printf("httpClient: using unix socket %q", addr)
		return &client{
			addr: "http://unix",
			cl: &http.Client{
				Transport: &http.Transport{
					DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
						dialer := net.Dialer{}
						return dialer.DialContext(ctx, "unix", addr)
					},
				},
			},
		}, nil
	}
	return nil, fmt.Errorf("Unsupported protocol for address: %q", addr)
}

func (c *client) Get(url string) (*http.Response, error) {
	resp, err := c.cl.Get(c.addr + url)
	return checkResponse(http.MethodGet, url, resp, err)
}

func (c *client) Head(url string) (*http.Response, error) {
	resp, err := c.cl.Head(c.addr + url)
	return checkResponse(http.MethodHead, url, resp, err)
}

func checkResponse(method, url string, resp *http.Response, err error) (*http.Response, error) {
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, ErrorNotFound{}
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("Unexpected status code on %v %q (expected 200 OK): %v", method, url, http.StatusText(resp.StatusCode))
	}
	return resp, nil
}

type ErrorNotFound struct {
}

func (e ErrorNotFound) Error() string {
	return "Not found"
}
