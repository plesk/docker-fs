package dockerfs

import "sync"

type Ino struct {
	inodes map[string]uint64
	mutex  sync.Mutex
}

func NewIno() *Ino {
	return &Ino{
		inodes: make(map[string]uint64),
	}
}

func (i *Ino) Inode(path string) uint64 {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	if value, ok := i.inodes[path]; ok {
		return value
	}

	// generate inode starting from 2
	n := uint64(len(i.inodes)) + 2
	i.inodes[path] = n
	return n
}
