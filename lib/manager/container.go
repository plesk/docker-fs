package manager

import (
	"fmt"
	"strings"

	"github.com/plesk/docker-fs/lib/dockerfs"
)

type Container struct {
	Id      string
	Names   []string
	Image   string
	Command string

	//
	MountPoint string
	Mounted    bool
	ShortId    string
	Name       string
}

func FromContainer(c *dockerfs.Container) Container {
	return Container{
		Id:      c.Id,
		Names:   c.Names,
		Image:   c.Image,
		Command: c.Command,
		ShortId: c.Id[:8],
		Name:    strings.TrimLeft(c.Names[0], "/"),
	}
}

func (c *Container) String() string {
	return fmt.Sprintf("%v %v (from %v): %v", c.Id[:8], strings.Join(c.Names, ", "), c.Image, c.Command)
}
