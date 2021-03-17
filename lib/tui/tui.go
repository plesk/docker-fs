package tui

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/manifoldco/promptui"
	"github.com/plesk/docker-fs/lib/manager"
)

type State int

const (
	ChooseAction State = iota
	List
)

type Tui struct {
	state State
	mng   *manager.Manager
}

func NewTui(mng *manager.Manager) *Tui {
	return &Tui{
		mng: mng,
	}
}

func (t *Tui) Run(state State) error {
	t.state = state
	for {
		if err := t.list(); err != nil {
			return err
		}
	}
}

func (t *Tui) list() error {
	cts, err := t.mng.ListContainers()
	if err != nil {
		return err
	}

	sel := promptui.Select{
		Label:     "Select container to mount",
		Items:     cts,
		Templates: listTemplates,
	}
	i, _, err := sel.Run()
	if err != nil {
		return err
	}
	ct := cts[i]
	if ct.Mounted {
		// ask to unmount
		sel := promptui.Select{
			Label: fmt.Sprintf("Unmount container %v from %v?", ct.ShortId, ct.MountPoint),
			Items: []string{
				"Yes",
				"No",
			},
		}
		i, _, err := sel.Run()
		if err != nil {
			return err
		}
		if i == 1 {
			return nil
		}
		// unmounting
		if err := t.mng.UnmountContainer(ct.Id, ct.MountPoint); err != nil {
			return err
		}
	} else {
		// Mounting
		promptPath := promptui.Prompt{
			Label:     "Choose path to mount docker container",
			Default:   fmt.Sprintf("./mount-%v", cts[i].Name),
			AllowEdit: true,
		}

		mountPoint, err := promptPath.Run()
		if err != nil {
			log.Fatal(err)
		}

		executable, err := os.Executable()
		if err != nil {
			return fmt.Errorf("Cannot detect executable path: %w", err)
		}

		cmd := exec.Command(executable, "-id", cts[i].Id, "-mount", mountPoint, "-daemonize")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("Mount command failed: %w", err)
		}
	}
	return nil
}
