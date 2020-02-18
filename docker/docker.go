package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Mng struct {
	unixc *http.Client

	changes               *FsChanges
	changesUpdated        time.Time
	changesUpdateInterval time.Duration

	// TODO replace with RWMutex
	mutex sync.Mutex
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
	return &Mng{
		unixc:                 unixc,
		changesUpdateInterval: 30 * time.Second,
	}
}

func (m *Mng) get(url string) (io.ReadCloser, error) {
	resp, err := m.unixc.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unexpected status code on GET %q (expected 200 OK): %v", url, http.StatusText(resp.StatusCode))
	}
	return resp.Body, nil
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

func (f *FsChanges) changesInDir(dir string) (result []fsChange) {
	dir = filepath.Clean(dir)
	for _, change := range f.Changes {
		// let's skip modified files for now
		if change.Kind == FileModified {
			continue
		}
		if filepath.Clean(filepath.Dir(change.Path)) == dir {
			result = append(result, change)
		}
	}
	return result
}

type fsChange struct {
	Path string
	Kind FsChangeKind

	mode uint32
}

type FsChangeKind int

const (
	FileModified FsChangeKind = 0
	FileAdded    FsChangeKind = 1
	FileRemoved  FsChangeKind = 2
)

func (k FsChangeKind) String() string {
	switch k {
	case FileModified:
		return "Modified"
	case FileAdded:
		return "Added"
	case FileRemoved:
		return "Removed"
	default:
		panic(fmt.Errorf("Unknown FsChangeKind: %d", k))
	}
}

func (m *Mng) fetchFsChanges(id string) error {
	resp, err := m.unixc.Get("http://unix/containers/" + id + "/changes")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Unexpected status code (expected 200 OK): %v", http.StatusText(resp.StatusCode))
	}

	changes := &FsChanges{}
	if err := json.NewDecoder(resp.Body).Decode(&(changes.Changes)); err != nil {
		return err
	}
	m.changes = changes
	m.changesUpdated = time.Now()
	return nil
}

func (m *Mng) ChangesInDir(id, dir string) (result []fsChange, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.changes == nil || time.Now().After(m.changesUpdated.Add(m.changesUpdateInterval)) {
		err = m.fetchFsChanges(id)
		if err != nil {
			return nil, err
		}
	}

	dir = filepath.Clean(dir)
	for _, change := range m.changes.Changes {
		// let's skip modified files for now
		if change.Kind == FileModified {
			continue
		}
		if filepath.Clean(filepath.Dir(change.Path)) != dir {
			// Not a direct child
			continue
		}
		data, err := m.getRawAttrs(id, change.Path)
		if err != nil {
			log.Printf("[ERR] Failed to get raw attrs of %q: %v", change.Path, err)
			continue
		}
		change.mode = uint32(data["mode"].(int64))
		result = append(result, change)
	}
	return result, nil
}

func (m *Mng) getRawAttrs(id, path string) (map[string]interface{}, error) {
	url := "http://unix/containers/" + id + "/archive?path=" + path
	resp, err := m.unixc.Head(url)
	if err != nil {
		return nil, fmt.Errorf("Head request to %q failed: %w", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unexpected status code on GET %q (expected 200 OK): %w", url, http.StatusText(resp.StatusCode))
	}
	stat := resp.Header.Get("X-Docker-Container-Path-Stat")
	if stat == "" {
		return nil, fmt.Errorf("X-Docker-Container-Path-Stat header not found")
	}
	data := make(map[string]interface{})
	err = json.NewDecoder(base64.NewDecoder(base64.StdEncoding, strings.NewReader(stat))).Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("Decoding failed: %w, %v", stat, err)
	}
	return data, nil
}
