package dockerfs

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

var _ = (fs.NodeGetattrer)((*LittleDir)(nil))
var _ = (fs.NodeLookuper)((*LittleDir)(nil))

// var _ = (fs.NodeReaddirer)((*LittleDir)(nil))

type LittleDir struct {
	fs.Inode
	mng *Mng

	fullpath string
}

func (r *LittleDir) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = 0755
	return 0
}

func (d *LittleDir) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	log.Printf("[DEBUG] (%s) Lookup(%s)...", d.fullpath, name)
	path := filepath.Join(d.fullpath, name)

	attrs, err := d.mng.getRawAttrs(path)
	if err == ErrorNotFound {
		return nil, syscall.ENOENT
	}
	if err != nil {
		log.Printf("Failed to get raw attrs: %v", err)
		return nil, syscall.EIO
	}
	mode := os.FileMode(uint32(attrs["mode"].(int64)))
	if mode.IsDir() {
		return d.NewPersistentInode(ctx, &LittleDir{mng: d.mng, fullpath: path}, fs.StableAttr{Mode: fuse.S_IFDIR}), 0
	}

	return d.NewPersistentInode(ctx, &File{mng: d.mng, fullpath: path}, fs.StableAttr{}), 0
}
