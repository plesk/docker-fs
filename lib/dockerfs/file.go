package dockerfs

import (
	"context"
	"errors"
	"io/ioutil"
	"syscall"

	"github.com/plesk/docker-fs/lib/log"

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

func (f *File) Open(ctx context.Context, flags uint32) (fh fs.FileHandle, mode uint32, syserr syscall.Errno) {
	defer log.Printf("[debug] File (%s) Open(%o): %v", f.fullpath, flags, syserr)
	// Fetch file content
	reader, err := f.mng.docker.GetFile(f.fullpath)
	if errors.As(err, &ErrorNotFound{}) {
		return nil, 0, syscall.ENOENT
	}
	if err != nil {
		log.Printf("[error] Failed to get file archive for %q: %v", f.fullpath, err)
		return nil, 0, syscall.EIO
	}
	defer reader.Close()
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Printf("[error] Failed to read file from tar archive for %q: %v", f.fullpath, err)
		return nil, 0, syscall.EIO
	}
	f.data = data

	// check flags
	if (flags&syscall.O_RDONLY) == syscall.O_RDONLY || (flags&syscall.O_RDWR) == syscall.O_RDWR {
		log.Printf("[trace] File (%s) read", f.fullpath)
		f.read = true
	}
	if (flags&syscall.O_WRONLY) == syscall.O_WRONLY || (flags&syscall.O_RDWR) == syscall.O_RDWR {
		log.Printf("[trace] File (%s) write", f.fullpath)
		f.write = true
	}
	if (flags & syscall.O_APPEND) == syscall.O_APPEND {
		log.Printf("[trace] File (%s) append", f.fullpath)
		f.pos = int64(len(f.data))
	}
	if (flags & syscall.O_TRUNC) == syscall.O_TRUNC {
		log.Printf("[trace] File (%s) truncate", f.fullpath)
		f.data = f.data[:0]
	}
	return nil, 0, 0
}

// Read simply returns the data that was already unpacked in the Open call
func (f *File) Read(ctx context.Context, fh fs.FileHandle, dest []byte, off int64) (result fuse.ReadResult, syserr syscall.Errno) {
	defer log.Printf("[debug] File (%s) Read(%d bytes, offset = %d): %v, %v", f.fullpath, len(dest), off, result, syserr)
	end := int(off) + len(dest)
	if end > len(f.data) {
		end = len(f.data)
	}
	return fuse.ReadResultData(f.data[off:end]), 0
}

func (f *File) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) (syserr syscall.Errno) {
	defer log.Printf("[debug] File (%s) Getattr(): %v", f.fullpath, syserr)
	attrs, err := f.mng.docker.GetPathAttrs(f.fullpath)
	if errors.As(err, &ErrorNotFound{}) {
		return syscall.ENOENT
	}
	if err != nil {
		log.Printf("[error] File(%s) Getting raw attrs failed: %v (%T)", f.fullpath, err, err)
		return syscall.EIO
	}
	out.Mode = uint32(attrs.Mode) & 07777
	out.Nlink = 1
	out.Size = uint64(attrs.Size)
	out.SetTimes(nil, &attrs.Mtime, nil)

	out.Owner.Uid, out.Owner.Gid = f.mng.uid, f.mng.gid
	return 0
}

func (f *File) Write(ctx context.Context, fh fs.FileHandle, data []byte, off int64) (n uint32, syserr syscall.Errno) {
	defer log.Printf("[debug] File (%s) Write(%d bytes, offset = %d): %d, %v", f.fullpath, len(data), off, n, syserr)
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

// On closing file
func (f *File) Flush(ctx context.Context, fh fs.FileHandle) (res syscall.Errno) {
	defer log.Printf("[debug] File (%v) Flush() = %v", f.fullpath, res)
	if !f.write {
		return 0
	}
	if err := f.mng.docker.SaveFile(f.fullpath, f.data, nil); err != nil {
		log.Printf("[error] Failed to save file: %v", err)
		return syscall.EIO
	}
	// reset/free memory
	f.data = nil
	f.read, f.write = false, false
	return 0
}

func (f *File) Fsync(ctx context.Context, fh fs.FileHandle, flags uint32) (res syscall.Errno) {
	defer log.Printf("[debug] File (%v) Fsync() = %v", f.fullpath, res)
	if !f.write {
		return 0
	}
	if err := f.mng.docker.SaveFile(f.fullpath, f.data, nil); err != nil {
		log.Printf("[error] Failed to save file: %v", err)
		return syscall.EIO
	}
	return 0
}
