package dockerfs

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

var _ = (fs.NodeGetattrer)((*Dir)(nil))
var _ = (fs.NodeLookuper)((*Dir)(nil))
var _ = (fs.NodeReaddirer)((*Dir)(nil))

type Dir struct {
	fs.Inode
	mng *Mng

	fullpath string
}

func (d *Dir) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = 0755
	return 0
}

func (d *Dir) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	log.Printf("[DEBUG] (%s) Lookup(%s)...", d.fullpath, name)
	path := filepath.Join(d.fullpath, name)

	attrs, err := d.mng.getRawAttrs(path)
	if errors.As(err, &ErrorNotFound{}) {
		return nil, syscall.ENOENT
	}
	if err != nil {
		log.Printf("Failed to get raw attrs: %v, (%T)", err, err)
		return nil, syscall.EIO
	}
	mode := attrs.Mode
	log.Printf("[DEBUG] (%s) Lookup(%s): mode = %o", d.fullpath, name, mode)
	inode := d.mng.inodes.Inode(filepath.Clean(path))
	if (mode & os.ModeSymlink) != 0 {
		linkTarget := attrs.LinkTarget
		return d.NewPersistentInode(ctx, &fs.MemSymlink{Data: []byte(linkTarget)}, fs.StableAttr{Mode: fuse.S_IFLNK, Ino: inode}), 0
	}

	if mode.IsDir() {
		return d.NewPersistentInode(ctx, &Dir{mng: d.mng, fullpath: path}, fs.StableAttr{Mode: fuse.S_IFDIR, Ino: inode}), 0
	}

	return d.NewPersistentInode(ctx, &File{mng: d.mng, fullpath: path}, fs.StableAttr{Ino: inode}), 0
}

func (d *Dir) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	log.Printf("[DEBUG] (%s) Readdir()...", d.fullpath)
	children := make(map[string]uint32)
	path := d.fullpath
	if path != "/" {
		path = path + "/"
	}

	changes, err := d.mng.ChangesInDir(d.fullpath)
	if err != nil {
		log.Printf("[ERR] Cannot retrieve FS changes: %v", err)
		return nil, syscall.EIO
	}

	// check static files and removed ones
	for name, mode := range d.mng.staticFiles {
		if !strings.HasPrefix(name, path) {
			continue
		}
		// Check if file is removed
		if changes.WasRemoved(name) {
			continue
		}
		sub := name[len(path):]
		pos := strings.Index(sub, "/")
		if pos > 0 {
			log.Printf("[DEBUG] Readdir (1): children[%v] = %o", sub[:pos], fuse.S_IFDIR)
			children[sub[:pos]] = fuse.S_IFDIR
		} else if pos < 0 {
			log.Printf("[DEBUG] Readdir (2): children[%v] = %o", sub, uint32(mode))
			if (mode & fuse.S_IFLNK) == fuse.S_IFLNK {
				children[sub] = fuse.S_IFLNK
			} else {
				children[sub] = fuse.S_IFREG
			}
		}
	}

	// check added files
	for _, ch := range changes {
		if ch.Kind != FileAdded {
			continue
		}
		log.Printf("[DEBUG] Readdir (3): childred[%v] = %o", filepath.Base(ch.Path), ch.mode)
		fuseMode := uint32(fuse.S_IFREG)
		if os.FileMode(ch.mode).IsDir() {
			fuseMode = fuse.S_IFDIR
		}
		children[filepath.Base(ch.Path)] = fuseMode
	}

	var list []fuse.DirEntry
	for child, mode := range children {
		inode := d.mng.inodes.Inode(filepath.Clean(filepath.Join(d.fullpath, child)))
		list = append(list, fuse.DirEntry{
			Mode: mode,
			Name: child,
			Ino:  inode,
		})
	}
	return fs.NewListDirStream(list), 0
}
