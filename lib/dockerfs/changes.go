package dockerfs

import "fmt"

type fsChanges []fsChange

func (c fsChanges) WasRemoved(path string) bool {
	for _, ch := range c {
		if ch.Path == path && ch.Kind == FileRemoved {
			return true
		}
	}
	return false
}

type fsChange struct {
	Path string       `json:"Path"`
	Kind FsChangeKind `json:"Kind"`

	mode uint32
}

type FsChangeKind int

const (
	FileModified FsChangeKind = 0
	FileAdded    FsChangeKind = 1
	FileRemoved  FsChangeKind = 2
)

func (k FsChangeKind) String() string {
	switch k {
	case FileModified:
		return "Modified"
	case FileAdded:
		return "Added"
	case FileRemoved:
		return "Removed"
	default:
		panic(fmt.Errorf("Unknown FsChangeKind: %d", k))
	}
}
