package docker

import (
	"archive/tar"
	"context"
	"io/ioutil"
	"log"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type dockerFile struct {
	fs.Inode

	containerId string
	name        string

	mng *Mng

	data []byte
}

func (df *dockerFile) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	body, err := df.mng.get("http://unix/containers/" + df.containerId + "/archive?path=" + df.name)
	if err != nil {
		// TODO logging
		return nil, 0, syscall.EIO
	}
	untar := tar.NewReader(body)
	data, err := ioutil.ReadAll(untar)
	if err != nil {
		// TODO logging
		return nil, 0, syscall.EIO
	}
	df.data = data
	// OK
	return nil, 0, 0
}

func (df *dockerFile) ReadRead(ctx context.Context, f fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	end := int(off) + len(dest)
	if end > len(df.data) {
		end = len(df.data)
	}
	return fuse.ReadResultData(df.data[off:end]), 0
}

func (df *dockerFile) Getattr(ctx context.Context, f fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	data, err := df.mng.getRawAttrs(df.containerId, df.name)
	if err != nil {
		log.Printf("Getting raw attrs failed: %v", err)
		return syscall.EIO
	}

	out.Mode = uint32(data["mode"].(int64)) & 07777
	out.Nlink = 1
	out.Size = uint64(data["size"].(int64))

	return 0
}
