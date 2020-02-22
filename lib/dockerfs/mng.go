package dockerfs

import (
	"archive/tar"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
)

type Mng struct {
	dockerAddr string
	httpc      httpClient

	id string

	inodes *Ino

	staticFiles map[string]os.FileMode

	changes               fsChanges
	changesUpdated        time.Time
	changesUpdateInterval time.Duration
	// TODO replace with RWMutex
	changesMutex sync.Mutex
}

func NewMng(containerId string) *Mng {
	return &Mng{
		id:                    containerId,
		dockerAddr:            "unix:/var/run/docker.sock",
		changesUpdateInterval: 1 * time.Second,
		inodes:                NewIno(),
	}
}

func (m *Mng) Init() (err error) {
	if m.httpc == nil {
		m.httpc, err = NewClient(m.dockerAddr)
		if err != nil {
			return err
		}
	}

	log.Printf("[DEBUG] fetching container content...")
	archPath, err := m.fetchContainerArchive()
	if err != nil {
		return err
	}
	defer os.Remove(archPath)
	log.Printf("[DEBUG] parse container content...")
	m.staticFiles, err = parseContainterContent(archPath)
	return err
}

func (m *Mng) Root() fs.InodeEmbedder {
	return &Dir{
		mng:      m,
		fullpath: "/",
	}
}

// Fetch container archive and return path to tar-file.
func (m *Mng) fetchContainerArchive() (path string, err error) {
	resp, err := m.httpc.Get("/containers/" + m.id + "/export")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

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

		switch hdr.Typeflag {
		case tar.TypeReg, tar.TypeRegA, tar.TypeSymlink:
			result["/"+filepath.Clean(hdr.Name)] = os.FileMode(uint32(hdr.Mode))
		case tar.TypeDir:
			// skip empty dirs
		default:
			log.Printf("Don't know how to handle file of type %v: %q. Skipping.", hdr.Typeflag, hdr.Name)
		}
	}
	return result, nil
}

func (m *Mng) getRawAttrs(path string) (*ContainerPathStat, error) {
	url := "/containers/" + m.id + "/archive?path=" + path
	resp, err := m.httpc.Head(url)
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

func (m *Mng) getFileArchive(path string) (io.ReadCloser, error) {
	url := "/containers/" + m.id + "/archive?path=" + path
	resp, err := m.httpc.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Head request to %q failed: %w", url, err)
	}
	return resp.Body, nil
}

func (m *Mng) ChangesInDir(dir string) (result fsChanges, err error) {
	m.changesMutex.Lock()
	defer m.changesMutex.Unlock()
	if m.changes == nil || time.Now().After(m.changesUpdated.Add(m.changesUpdateInterval)) {
		err = m.fetchFsChanges()
		if err != nil {
			return nil, err
		}
	}

	dir = filepath.Clean(dir)
	for _, change := range m.changes {
		// let's skip modified files for now
		if change.Kind == FileModified {
			continue
		}
		if filepath.Clean(filepath.Dir(change.Path)) != dir {
			// Not a direct child
			continue
		}
		stat, err := m.getRawAttrs(change.Path)
		if err != nil {
			if !errors.As(err, &ErrorNotFound{}) {
				log.Printf("[ERR] Failed to get raw attrs of %q: %v", change.Path, err)
			}
			continue
		}
		change.mode = uint32(stat.Mode)
		result = append(result, change)
	}
	return fsChanges(result), nil
}

func (m *Mng) fetchFsChanges() error {
	resp, err := m.httpc.Get("/containers/" + m.id + "/changes")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	changes := fsChanges([]fsChange{})
	if err := json.NewDecoder(resp.Body).Decode(&changes); err != nil {
		return err
	}
	m.changes = changes
	m.changesUpdated = time.Now()
	return nil
}

func (m *Mng) saveFile(path string, data []byte) error {
	var buffer bytes.Buffer
	writer := tar.NewWriter(&buffer)
	dir, name := filepath.Split(path)
	hdr := &tar.Header{
		Name: name,
		Size: int64(len(data)),
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

	url := "/containers/" + m.id + "/archive?path=" + dir
	_, err := m.httpc.Put(url, http.DetectContentType(data), bytes.NewBuffer(data))
	return err
}
