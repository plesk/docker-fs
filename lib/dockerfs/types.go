package dockerfs

import (
	"os"
	"time"
)

// This type is taken from github.com/docker/docker
type ContainerPathStat struct {
	Name       string      `json:"name"`
	Size       int64       `json:"size"`
	Mode       os.FileMode `json:"mode"`
	Mtime      time.Time   `json:"mtime"`
	LinkTarget string      `json:"linkTarget"`
}
