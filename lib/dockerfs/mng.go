package dockerfs

import (
	"archive/tar"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/hanwen/go-fuse/v2/fs"
)

type Mng struct {
	dockerAddr string
	unixc      *http.Client

	id string

	staticFiles map[string]os.FileMode
}

func NewMng(containerId string) *Mng {
	return &Mng{
		id:         containerId,
		dockerAddr: "/var/run/docker.sock",
	}
}

func (m *Mng) Init() error {
	m.unixc = &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				dialer := net.Dialer{}
				return dialer.DialContext(ctx, "unix", m.dockerAddr)
			},
		},
	}

	return nil
}

func (m *Mng) Root() fs.InodeEmbedder {
	return &LittleDir{
		mng:      m,
		fullpath: "/",
	}
}

// Fetch container archive and return path to tar-file.
func (m *Mng) fetchContainerArchive() (path string, err error) {
	resp, err := m.unixc.Get("http://unix/containers/" + m.id + "/export")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Unexpected status code (expected 200 OK): %v", http.StatusText(resp.StatusCode))
	}
	output, err := prepareOutputFile(m.id)
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
	dir := filepath.Join(home, ".cache/dockerfs")
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(filepath.Join(dir, fmt.Sprintf("content_%s.tar", id)), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
	return file, err
}

func parseContainterContent(file string) (map[string]os.FileMode, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tr := tar.NewReader(f)

	result := make(map[string]os.FileMode)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			// end of tar archive
			break
		}
		if err != nil {
			log.Printf("Add: %v", err)
			// XXX handle error
			break
		}
		if hdr.Typeflag == 'L' {
			// Don't know what to do with it
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeReg, tar.TypeRegA:
			result[filepath.Clean(hdr.Name)] = os.FileMode(uint32(hdr.Mode))
		default:
			log.Printf("Don't know how to handle file of type %v: %q. Skipping.", hdr.Typeflag, hdr.Name)
		}
	}
	return result, nil
}

var ErrorNotFound = errors.New("Not found")

func (m *Mng) getRawAttrs(path string) (map[string]interface{}, error) {
	url := "http://unix/containers/" + m.id + "/archive?path=" + path
	resp, err := m.unixc.Head(url)
	if err != nil {
		return nil, fmt.Errorf("Head request to %q failed: %w", url, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrorNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unexpected status code on GET %q (expected 200 OK): %v", url, http.StatusText(resp.StatusCode))
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

func (m *Mng) getFileArchive(path string) (io.ReadCloser, error) {
	url := "http://unix/containers/" + m.id + "/archive?path=" + path
	resp, err := m.unixc.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Head request to %q failed: %w", url, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrorNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unexpected status code on GET %q (expected 200 OK): %v", url, http.StatusText(resp.StatusCode))
	}
	return resp.Body, nil
}
