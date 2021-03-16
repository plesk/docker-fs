package dockerfs

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

const (
	suffixAdded   = ".added"
	suffixRemoved = ".removed"
)

func (d *dockerMngMock) ContainerExport() (io.ReadCloser, error) {
	buffer := &bytes.Buffer{}
	tw := tar.NewWriter(buffer)
	err := filepath.Walk(d.root, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(file, suffixAdded) {
			// skip "added" files
			return nil
		}

		// strip ".removed" suffix if present
		file = strings.TrimSuffix(file, suffixRemoved)

		name := file[len(d.root):]
		if name == "" {
			// skip root
			return nil
		}
		name = "." + name
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

func (d *dockerMngMock) GetPathAttrs(path string) (st *ContainerPathStat, err error) {
	// Check if file was added
	fullpath := filepath.Join(d.root, path)
	if _, err := os.Lstat(fullpath + suffixAdded); err == nil {
		fullpath += suffixAdded
	}

	// Check if file was added along with the directory
	dr, fl := filepath.Split(fullpath)
	dr = dr[:len(dr)-1]
	fp := filepath.Join(dr+suffixAdded, fl+suffixAdded)
	if _, err := os.Lstat(fp); err == nil {
		fullpath = fp
	}

	fi, err := os.Lstat(fullpath)
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

func (d *dockerMngMock) GetFsChanges() (changes FsChanges, err error) {
	err = filepath.Walk(d.root, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		file = file[len(d.root):]
		if file == "" {
			return nil
		}
		if strings.HasSuffix(file, suffixAdded) {
			dr, fl := filepath.Split(file)
			file := filepath.Join(strings.TrimSuffix(dr, suffixAdded+"/"), strings.TrimSuffix(fl, suffixAdded))
			changes = append(changes, FsChange{
				Path: file,
				Kind: FileAdded,
			})
		} else if strings.HasSuffix(file, suffixRemoved) {
			file = strings.TrimSuffix(file, suffixRemoved)
			changes = append(changes, FsChange{
				Path: file,
				Kind: FileRemoved,
			})
		}
		return nil
	})
	return changes, err
}

// Get plain file content
func (d *dockerMngMock) GetFile(path string) (io.ReadCloser, error) {
	// Check if file was added
	fullpath := filepath.Join(d.root, path)
	if _, err := os.Lstat(fullpath + suffixAdded); err == nil {
		fullpath += suffixAdded
	}

	// Check if file was added along with the directory
	dr, fl := filepath.Split(fullpath)
	dr = dr[:len(dr)-1]
	fp := filepath.Join(dr+suffixAdded, fl+suffixAdded)
	if _, err := os.Lstat(fp); err == nil {
		fullpath = fp
	}

	f, err := os.Open(fullpath)
	if os.IsNotExist(err) {
		return nil, ErrorNotFound{}
	}
	return f, err
}

// Save file
func (d *dockerMngMock) SaveFile(path string, data []byte, stat *ContainerPathStat) (err error) {
	// Only modification is supported at the moment
	fullpath := filepath.Join(d.root, path)
	f, err := os.OpenFile(fullpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	return err
}

func (d *dockerMngMock) ContainersList() ([]Container, error) {
	return nil, nil
}
