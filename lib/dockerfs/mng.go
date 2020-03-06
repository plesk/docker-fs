package dockerfs

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/plesk/docker-fs/lib/log"

	"github.com/hanwen/go-fuse/v2/fs"
)

type Mng struct {
	dockerAddr string
	docker     dockerMng

	id string

	inodes *Ino

	staticFiles map[string]os.FileMode

	changes               FsChanges
	changesUpdated        time.Time
	changesUpdateInterval time.Duration
	// TODO replace with RWMutex
	changesMutex sync.Mutex

	// current user uid, gid
	uid, gid uint32
}

func NewMng(containerId string) *Mng {
	return &Mng{
		id:                    containerId,
		dockerAddr:            "unix:/var/run/docker.sock",
		changesUpdateInterval: 1 * time.Second,
		inodes:                NewIno(),
		uid:                   uint32(os.Getuid()),
		gid:                   uint32(os.Getgid()),
	}
}

func (m *Mng) Init() (err error) {
	if m.docker == nil {
		httpc, err := NewClient(m.dockerAddr)
		if err != nil {
			return err
		}
		m.docker = NewDockerMng(httpc, m.id)
	}

	log.Printf("[debug] fetching container content...")
	archPath, err := m.fetchContainerArchive()
	if err != nil {
		return err
	}
	defer os.Remove(archPath)
	log.Printf("[debug] parse container content...")
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
	respBody, err := m.docker.ContainerExport()
	if err != nil {
		return "", err
	}
	defer respBody.Close()

	output, err := prepareOutputFile(m.id)
	defer output.Close()

	if err != nil {
		return "", err
	}
	if _, err := io.Copy(output, respBody); err != nil {
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
			log.Printf("[debug] Add: %v", err)
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

func (m *Mng) ChangesInDir(dir string) (result FsChanges, err error) {
	m.changesMutex.Lock()
	defer m.changesMutex.Unlock()
	if m.changes == nil || time.Now().After(m.changesUpdated.Add(m.changesUpdateInterval)) {
		changes, err := m.docker.GetFsChanges()
		if err != nil {
			return nil, err
		}
		m.changes = changes
		m.changesUpdated = time.Now()
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
		stat, err := m.docker.GetPathAttrs(change.Path)
		if err != nil {
			if !errors.As(err, &ErrorNotFound{}) {
				log.Printf("[error] Failed to get raw attrs of %q: %v", change.Path, err)
			}
			continue
		}
		change.mode = uint32(stat.Mode)
		result = append(result, change)
	}
	return FsChanges(result), nil
}
