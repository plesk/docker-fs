package dockerfs

import (
	"archive/tar"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

type dockerMng interface {
	// returns read-closer to tar-archive fetched by /containers/{id}/export api method
	ContainerExport() (io.ReadCloser, error)

	GetPathAttrs(path string) (*ContainerPathStat, error)

	GetFsChanges() (FsChanges, error)

	// Get plain file content
	GetFile(path string) (io.ReadCloser, error)

	// Save file
	SaveFile(path string, data []byte, stat *ContainerPathStat) (err error)

	// List containers
	ContainersList() ([]Container, error)
}

var _ = (dockerMng)((*dockerMngImpl)(nil))

type dockerMngImpl struct {
	httpc httpClient
	id    string
}

func NewDockerMng(httpc httpClient, containerId string) dockerMng {
	return &dockerMngImpl{
		httpc: httpc,
		id:    containerId,
	}
}

func (d *dockerMngImpl) ContainerExport() (io.ReadCloser, error) {
	resp, err := d.httpc.Get("/containers/" + d.id + "/export")
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (d *dockerMngImpl) GetPathAttrs(path string) (*ContainerPathStat, error) {
	url := "/containers/" + d.id + "/archive?path=" + path
	resp, err := d.httpc.Head(url)
	if err != nil {
		return nil, fmt.Errorf("Head request to %q failed: %w", url, err)
	}
	stat := resp.Header.Get("X-Docker-Container-Path-Stat")
	if stat == "" {
		return nil, fmt.Errorf("X-Docker-Container-Path-Stat header not found")
	}
	data := new(ContainerPathStat)
	err = json.NewDecoder(base64.NewDecoder(base64.StdEncoding, strings.NewReader(stat))).Decode(data)
	if err != nil {
		return nil, fmt.Errorf("Decoding failed: %q, %w", stat, err)
	}
	return data, nil
}

func (d *dockerMngImpl) GetFsChanges() (FsChanges, error) {
	resp, err := d.httpc.Get("/containers/" + d.id + "/changes")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	changes := FsChanges([]FsChange{})
	if err := json.NewDecoder(resp.Body).Decode(&changes); err != nil {
		return nil, err
	}
	return changes, nil
}

func (d *dockerMngImpl) GetFile(path string) (io.ReadCloser, error) {
	url := "/containers/" + d.id + "/archive?path=" + path
	resp, err := d.httpc.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Head request to %q failed: %w", url, err)
	}
	tr := tar.NewReader(resp.Body)
	if _, err := tr.Next(); err != nil {
		return nil, fmt.Errorf("Failed to find file in tar archive: %w", err)
	}
	return &readCloser{
		reader: tr,
		close: func() error {
			return resp.Body.Close()
		},
	}, nil
}

func (d *dockerMngImpl) ContainersList() ([]Container, error) {
	url := "/containers/json"
	resp, err := d.httpc.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Get request to %q failed: %w", url, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var cts []Container
	if err := json.Unmarshal(data, &cts); err != nil {
		return nil, err
	}
	return cts, nil
}

type readCloser struct {
	reader io.Reader
	close  func() error
}

func (rc *readCloser) Read(p []byte) (n int, err error) {
	return rc.reader.Read(p)
}

func (rc *readCloser) Close() error {
	return rc.close()
}

// Save file content.
// Currently supports only modification of existing files.
func (d *dockerMngImpl) SaveFile(path string, data []byte, stat *ContainerPathStat) (err error) {
	if stat == nil {
		stat, err = d.GetPathAttrs(path)
		if err != nil {
			return err
		}
	}

	var buffer bytes.Buffer
	writer := tar.NewWriter(&buffer)
	dir, name := filepath.Split(path)
	hdr := &tar.Header{
		Name:    name,
		Size:    int64(len(data)),
		Mode:    int64(stat.Mode),
		ModTime: time.Now(),
	}
	if err := writer.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := writer.Write(data); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	url := "/containers/" + d.id + "/archive?path=" + dir
	_, err = d.httpc.Put(url, http.DetectContentType(buffer.Bytes()), &buffer)
	return err
}
