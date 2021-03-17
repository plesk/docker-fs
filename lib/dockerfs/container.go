package dockerfs

import (
	"fmt"
	"strings"
)

type Container struct {
	Id      string
	Names   []string
	Image   string
	Command string
}

func (c *Container) String() string {
	return fmt.Sprintf("%v %v (from %v): %v", c.Id[:8], strings.Join(c.Names, ", "), c.Image, c.Command)
}
