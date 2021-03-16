package manager

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/plesk/docker-fs/lib/log"

	"github.com/plesk/docker-fs/lib/dockerfs"

	"github.com/hanwen/go-fuse/v2/fs"

	daemon "github.com/sevlyar/go-daemon"
)

type Manager struct {
	statusPath string
}

func New() *Manager {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("[warning] Cannot detect user home directory. Use /tmp.")
		home = "/tmp"
	}
	return &Manager{
		statusPath: filepath.Join(home, ".dockerfs.status.json"),
	}
}

func (m *Manager) ListContainers() ([]Container, error) {
	httpc, err := dockerfs.NewClient("unix:/var/run/docker.sock")
	if err != nil {
		return nil, err
	}
	dmng := dockerfs.NewDockerMng(httpc, "")
	list, err := dmng.ContainersList()
	if err != nil {
		return nil, err
	}
	status, err := m.readStatus()
	if err != nil {
		return nil, err
	}
	var result []Container
	for _, c := range list {
		ct := FromContainer(&c)
		if s, ok := status[ct.Id]; ok {
			ct.MountPoint = s
			ct.Mounted = ct.MountPoint != ""
		}
		result = append(result, ct)
	}
	return result, nil
}

func (m *Manager) MountContainer(containerId, mountPoint string) error {
	log.Printf("[info] Check if mount directory exists (%v)...", mountPoint)
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return err
	}
	log.Printf("[info] Fetching content of container %v...", containerId)
	dockerMng := dockerfs.NewMng(containerId)
	if err := dockerMng.Init(); err != nil {
		return fmt.Errorf("dockerMng.Init() failed: %w", err)
	}

	root := dockerMng.Root()

	log.Printf("Mounting FS to %v...", mountPoint)
	server, err := fs.Mount(mountPoint, root, &fs.Options{})
	if err != nil {
		return fmt.Errorf("Mount failed: %w", err)
	}

	if err := m.writeStatus(containerId, mountPoint); err != nil {
		return err
	}

	// daemonize
	ctx := daemon.Context{
		LogFileName: fmt.Sprintf("/tmp/container-%v.log", containerId),
	}
	log.Printf("[warning] writing log to %v", ctx.LogFileName)
	child, err := ctx.Reborn()
	if err != nil {
		return fmt.Errorf("Daemonization failed: %w", err)
	}
	if child != nil {
		// parent process
		return nil
	}

	fmt.Println("OK!")
	server.Wait()
	fmt.Println("[info] Server finished.")

	return nil
}

func (m *Manager) UnmountContainer(id, path string) error {
	cmd := exec.Command("umount", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
	return m.writeStatus(id, "")
}

func (m *Manager) writeStatus(id, path string) error {
	fmt.Printf("write status: %q = %q\n", id, path)
	status, err := m.readStatus()
	if err != nil {
		return err
	}
	if path != "" {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		status[id] = absPath
	} else {
		delete(status, id)
	}
	data, err := json.Marshal(status)
	if err != nil {
		return err
	}
	fmt.Printf("status => %s\n", data)
	return ioutil.WriteFile(m.statusPath, data, 0644)
}

func (m *Manager) readStatus() (map[string]string, error) {
	data, err := ioutil.ReadFile(m.statusPath)
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	status := map[string]string{}
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, err
	}
	return status, nil
}
