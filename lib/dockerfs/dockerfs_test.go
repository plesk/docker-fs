package dockerfs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

var (
	server     *fuse.Server
	mountPoint string
)

func setup() {
	dir, err := ioutil.TempDir("", "dockerfs_test_")
	if err != nil {
		panic(fmt.Errorf("Cannot create test mount point: %v", err))
	}
	mountPoint = dir

	mng := NewMng("0001")
	mng.docker = newDockerMngMock()
	if err := mng.Init(); err != nil {
		panic(fmt.Errorf("mng.Init() failed: %v", err))
	}
	root := mng.Root()
	server, err = fs.Mount(mountPoint, root, &fs.Options{})
	if err != nil {
		panic(fmt.Errorf("fs.Mount(...) failed: %v", err))
	}
}

func shutdown() {
	if err := server.Unmount(); err != nil {
		panic(fmt.Errorf("Unmount() failed: %v", err))
	}
	if err := os.RemoveAll(mountPoint); err != nil {
		panic(fmt.Errorf("os.RemoveAll(%q) failed: %v", mountPoint, err))
	}
}

func TestFileList(t *testing.T) {
	expFiles := map[string]bool{
		"/file1.txt":      false,
		"/dir2/file2.txt": false,
		"/file3.txt":      false,
		"/dir2/file4.txt": false,
		"/dir3/file5.txt": false,
	}
	expDirs := map[string]bool{
		// Root dir
		"":      false,
		"/dir2": false,
		"/dir3": false,
	}

	err := filepath.Walk(mountPoint, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			t.Errorf("Error accessing file %q: %v", file, err)
			return nil
		}
		file = file[len(mountPoint):]
		t.Logf("Walk: %q", file)
		if !fi.IsDir() {
			// Regular file
			if _, ok := expFiles[file]; !ok {
				t.Errorf("Unexpected regular file found: %q", file)
			} else {
				expFiles[file] = true
			}
		} else {
			// Directory
			if _, ok := expDirs[file]; !ok {
				t.Errorf("Unexpected directory found: %q", file)
			} else {
				expDirs[file] = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("filepath.Walk(%q, ...) failed: %v", mountPoint, err)
	}
	for file, found := range expFiles {
		if !found {
			t.Errorf("File not found: %q", file)
		}
	}
	for dir, found := range expDirs {
		if !found {
			t.Errorf("Directory not found: %q", dir)
		}
	}
}

func TestReadRegularFile(t *testing.T) {
	testdata := []struct {
		path, content string
	}{
		{"file1.txt", "file1\n"},
		{"dir2/file2.txt", "file2\n"},
		{"file3.txt", "file3\n"},
		{"dir2/file4.txt", "file4\n"},
		{"dir3/file5.txt", "file5\n"},
	}
	for _, test := range testdata {
		t.Run(test.path, func(t *testing.T) {
			file := filepath.Join(mountPoint, test.path)
			content, err := ioutil.ReadFile(file)
			if err != nil {
				t.Fatalf("ReadFile(%q) failed: %v", file, err)
			}
			if act, exp := string(content), test.content; act != exp {
				t.Errorf("Incorrect file content: expected %q, actual %q", exp, act)
			}
		})
	}
}
