package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
)

type Mng struct {
	unixc *http.Client
}

func NewMng(dockerAddr string) *Mng {
	unixc := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				dialer := net.Dialer{}
				return dialer.DialContext(ctx, "unix", dockerAddr)
			},
		},
	}
	return &Mng{unixc: unixc}
}

// Fetch container archive and return path to tar-file.
func (m *Mng) FetchContainerArchive(id string) (path string, err error) {
	resp, err := m.unixc.Get("http://unix/containers/" + id + "/export")
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

type FsChanges struct {
	Changes []fsChange
}

type fsChange struct {
	Path string
	Kind FsChangeKind
}

type FsChangeKind int

const (
	FileModified FsChangeKind = 0
	FileAdded    FsChangeKind = 1
	FileRemoved  FsChangeKind = 2
)

func (m *Mng) FetchFsChanges(id string) (*FsChanges, error) {
	resp, err := m.unixc.Get("http://unix/containers/" + id + "/changes")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unexpected status code (expected 200 OK): %v", http.StatusText(resp.StatusCode))
	}

	changes := &FsChanges{}
	if err := json.NewDecoder(resp.Body).Decode(&(changes.Changes)); err != nil {
		return nil, err
	}
	return changes, nil
}
