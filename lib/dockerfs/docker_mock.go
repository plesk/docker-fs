package dockerfs

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

type dockerMngMock struct {
	//
	root string
}

var _ = (dockerMng)((*dockerMngMock)(nil))

func newDockerMngMock() *dockerMngMock {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic(fmt.Errorf("Cannot get caller file path."))
	}
	root := filepath.Join(filepath.Dir(file), "testdata/root")
	return &dockerMngMock{
		root: root,
	}
}

func (d *dockerMngMock) ContainerExport() (io.ReadCloser, error) {
	buffer := &bytes.Buffer{}
	tw := tar.NewWriter(buffer)
	err := filepath.Walk(d.root, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		name := file[len(d.root):]
		if name == "" {
			return nil
		}
		name = "." + name
		log.Printf("dockerMngMock: Add file to archive: %q", name)
		// generate tar header
		header, err := tar.FileInfoHeader(fi, name)
		if err != nil {
			return err
		}

		// must provide real name
		// (see https://golang.org/src/archive/tar/common.go?#L626)
		header.Name = filepath.ToSlash(name)

		// write header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		// if not a dir, write file content
		if !fi.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return err
			}
			if _, err := io.Copy(tw, data); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return ioutil.NopCloser(buffer), nil
}

func (d *dockerMngMock) GetPathAttrs(path string) (*ContainerPathStat, error) {
	fi, err := os.Lstat(filepath.Join(d.root, path))
	if os.IsNotExist(err) {
		return nil, ErrorNotFound{}
	}
	if err != nil {
		return nil, err
	}
	return &ContainerPathStat{
		Name:  fi.Name(),
		Size:  fi.Size(),
		Mode:  fi.Mode(),
		Mtime: fi.ModTime(),
		// TODO handle link target
	}, nil
}

func (d *dockerMngMock) GetFsChanges() (FsChanges, error) {
	// TODO
	return nil, nil
}

// Get plain file content
func (d *dockerMngMock) GetFile(path string) (io.ReadCloser, error) {
	f, err := os.Open(filepath.Join(d.root, path))
	if os.IsNotExist(err) {
		return nil, ErrorNotFound{}
	}
	return f, err
}

// Save file
func (d *dockerMngMock) SaveFile(path string, data []byte, stat *ContainerPathStat) (err error) {
	// Only modification is supported at the moment
	f, err := os.OpenFile("notes.txt", os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	return err
}
