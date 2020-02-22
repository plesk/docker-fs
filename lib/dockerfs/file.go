package dockerfs

import (
	"archive/tar"
	"context"
	"errors"
	"io/ioutil"
	"log"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

var _ = (fs.NodeOpener)((*File)(nil))
var _ = (fs.NodeReader)((*File)(nil))
var _ = (fs.NodeWriter)((*File)(nil))
var _ = (fs.NodeGetattrer)((*File)(nil))
var _ = (fs.NodeFlusher)((*File)(nil))
var _ = (fs.NodeFsyncer)((*File)(nil))

type File struct {
	fs.Inode
	mng *Mng

	fullpath    string
	data        []byte
	read, write bool
	pos         int64
}

func (f *File) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	// Fetch file content
	reader, err := f.mng.getFileArchive(f.fullpath)
	if errors.As(err, &ErrorNotFound{}) {
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

	// check flags
	if (flags&syscall.O_RDONLY) == syscall.O_RDONLY || (flags&syscall.O_RDWR) == syscall.O_RDWR {
		f.read = true
	}
	if (flags&syscall.O_WRONLY) == syscall.O_WRONLY || (flags&syscall.O_RDWR) == syscall.O_RDWR {
		f.write = true
	}
	if (flags & syscall.O_APPEND) == syscall.O_APPEND {
		f.pos = int64(len(f.data))
	}
	if (flags & syscall.O_TRUNC) == syscall.O_TRUNC {
		f.data = f.data[:0]
	}
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
	if errors.As(err, &ErrorNotFound{}) {
		return syscall.ENOENT
	}
	if err != nil {
		log.Printf("get raw attrs on %q failed: %v (%T)", f.fullpath, err, err)
		return syscall.EIO
	}
	out.Mode = uint32(attrs.Mode) & 07777
	out.Nlink = 1
	out.Size = uint64(attrs.Size)
	out.SetTimes(nil, &attrs.Mtime, nil)

	out.Owner.Uid, out.Owner.Gid = f.mng.uid, f.mng.gid
	return 0
}

func parseAttrTime(str string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, str)
}

func (f *File) Write(ctx context.Context, fh fs.FileHandle, data []byte, off int64) (uint32, syscall.Errno) {
	if !f.write {
		return 0, syscall.EBADF
	}

	off += f.pos

	// f.mu.Lock()
	// defer f.mu.Unlock()
	end := int64(len(data)) + off
	if int64(len(f.data)) < end {
		n := make([]byte, end)
		copy(n, f.data)
		f.data = n
	}

	copy(f.data[off:off+int64(len(data))], data)

	return uint32(len(data)), 0
}

func (f *File) Flush(ctx context.Context, fh fs.FileHandle) (res syscall.Errno) {
	defer log.Printf("[DEBUG] (%v) Flush() = %v", f.fullpath, res)
	if err := f.mng.saveFile(f.fullpath, f.data); err != nil {
		log.Printf("Failed to save file: %v", err)
		return syscall.EIO
	}
	return 0
}

func (f *File) Fsync(ctx context.Context, fh fs.FileHandle, flags uint32) (res syscall.Errno) {
	defer log.Printf("[DEBUG] (%v) Fsync() = %v", f.fullpath, res)
	if err := f.mng.saveFile(f.fullpath, f.data); err != nil {
		log.Printf("Failed to save file: %v", err)
		return syscall.EIO
	}
	// Maybe reset read/write flags?
	return 0
}
