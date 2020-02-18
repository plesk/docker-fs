package dockerfs

import (
	"archive/tar"
	"context"
	"io/ioutil"
	"log"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

var _ = (fs.NodeOpener)((*File)(nil))
var _ = (fs.NodeReader)((*File)(nil))
var _ = (fs.NodeGetattrer)((*File)(nil))

type File struct {
	fs.Inode
	mng *Mng

	fullpath string
	data     []byte
}

func (f *File) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	reader, err := f.mng.getFileArchive(f.fullpath)
	if err == ErrorNotFound {
		return nil, 0, syscall.ENOENT
	}
	if err != nil {
		log.Printf("Failed to get file archive: %v", err)
		return nil, 0, syscall.EIO
	}
	defer reader.Close()
	tr := tar.NewReader(reader)
	if _, err := tr.Next(); err != nil {
		log.Printf("Failed to find file in tar archive: %v", err)
		return nil, 0, syscall.EIO
	}
	data, err := ioutil.ReadAll(tr)
	if err != nil {
		log.Printf("Failed to read file from tar archive: %v", err)
		return nil, 0, syscall.EIO
	}
	f.data = data
	return nil, 0, 0
}

// Read simply returns the data that was already unpacked in the Open call
func (f *File) Read(ctx context.Context, fh fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	end := int(off) + len(dest)
	if end > len(f.data) {
		end = len(f.data)
	}
	return fuse.ReadResultData(f.data[off:end]), 0
}

func (f *File) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	attrs, err := f.mng.getRawAttrs(f.fullpath)
	if err == ErrorNotFound {
		return syscall.ENOENT
	}
	if err != nil {
		log.Printf("get raw attrs on %q failed: %v", f.fullpath, err)
		return syscall.EIO
	}
	out.Size = uint64(attrs["size"].(int64))
	return 0
}
