package docker

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type dockerDir struct {
	fs.Inode

	containerId string
	fullname    string

	mng *Mng

	entries []fuse.DirEntry
}

func (dd *dockerDir) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	changes, err := dd.mng.ChangesInDir(dd.containerId, dd.fullname)
	if err != nil {
		log.Printf("[ERR] Changes in Dir %q failed: %v", dd.fullname, err)
		return nil, syscall.EIO
	}

	var res []fuse.DirEntry
	// Check for removed files
ENTRY:
	for _, entry := range dd.entries {
		for _, change := range changes {
			if change.Kind == FileRemoved && filepath.Base(change.Path) == entry.Name {
				continue ENTRY
			}
		}
		res = append(res, entry)
	}
	//check for added files
	for _, change := range changes {
		if change.Kind != FileAdded {
			continue
		}
		res = append(res, fuse.DirEntry{
			Mode: change.mode,
			Name: filepath.Base(change.Path),
		})
	}
	return fs.NewListDirStream(res), 0
}

func (dd *dockerDir) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*Inode, syscall.Errno) {
	data, err := dd.mng.getRawAttrs(dd.containerId, filepath.Join(dd.fullpath, name))
	if err != nil {
		log.Printf("Get raw attrs failed: %v", err)
		return nil, syscall.ENOENT
	}
	mode := uint32(data["mode"].(int64))
	if os.FileMode(mode).IsDir() {

	}
}
