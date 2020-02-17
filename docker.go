package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
)

var unixc = http.Client{
	Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			dialer := net.Dialer{}
			return dialer.DialContext(ctx, "unix", dockerSocketAddr)
		},
	},
}

// Fetch container archive and return path to tar-file.
func FetchContainerArchive(id string) (path string, err error) {
	resp, err := unixc.Get("http://unix/containers/" + id + "/export")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Unexpected status code (expected 200 OK): %v", http.StatusText(resp.StatusCode))
	}
	output, err := prepareOutputFile(id)
	defer output.Close()

	if err != nil {
		return "", err
	}
	if _, err := io.Copy(output, resp.Body); err != nil {
		return "", err
	}
	return output.Name(), nil
}

func prepareOutputFile(id string) (*os.File, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".cache/dockerfs", id)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(filepath.Join(dir, "content.tar"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
	return file, err
}
