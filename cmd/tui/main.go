package main

import (
	"github.com/plesk/docker-fs/lib/log"

	"github.com/plesk/docker-fs/lib/manager"
	"github.com/plesk/docker-fs/lib/tui"
)

func main() {
	mng := manager.New()
	ui := tui.NewTui(mng)

	if err := ui.Run(tui.List); err != nil {
		log.Fatal(err)
	}
}
