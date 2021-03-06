package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/plesk/docker-fs/lib/log"
	"github.com/plesk/docker-fs/lib/tui"

	"github.com/plesk/docker-fs/lib/manager"

	"github.com/hanwen/go-fuse/v2/fuse"
)

var (
	// Docker container ID (or name)
	containerId string

	// Directory to mount container FS
	mountPoint string

	//
	dockerSocketAddr string

	daemonize bool

	logLevel       string
	verbose, quiet bool
)

func init() {
	flag.StringVar(&containerId, "id", "", "Docker containter ID (or name)")
	flag.StringVar(&containerId, "i", "", "Docker containter ID (or name)")

	flag.StringVar(&mountPoint, "mount", "", "Mount point for containter FS")
	flag.StringVar(&mountPoint, "m", "", "Mount point for containter FS")

	flag.BoolVar(&daemonize, "daemonize", false, "Daemonize fuse process")
	flag.BoolVar(&daemonize, "d", false, "Daemonize fuse process")

	// TODO make http support
	flag.StringVar(&dockerSocketAddr, "docker-socket", "/var/run/docker.sock", "Docker socket")

	flag.StringVar(&logLevel, "log-level", "warning", "Logging level")
	flag.BoolVar(&verbose, "verbose", false, "Increase loggin level to 'debug'")
	flag.BoolVar(&verbose, "v", false, "Increase loggin level to 'debug'")
	flag.BoolVar(&quiet, "quiet", false, "Decrease loggin level to 'error'")
	flag.BoolVar(&quiet, "q", false, "Decrease loggin level to 'error'")
}

func main() {
	flag.Parse()

	if containerId != "" {
		if mountPoint == "" {
			fmt.Fprintf(os.Stderr, "Mount point is not specified.\n")
			flag.Usage()
			os.Exit(2)
		}
		mng := manager.New()
		if err := mng.MountContainer(containerId, mountPoint, daemonize); err != nil {
			log.Fatal(err)
		}
		return
	}

	if verbose && quiet {
		fmt.Fprintf(os.Stderr, "Cannot make it quite and verbose simultaneously\n")
		flag.Usage()
		os.Exit(2)
	}
	if verbose {
		logLevel = log.Debug.String()
	}
	if quiet {
		logLevel = log.Error.String()
	}
	if err := log.SetLevel(logLevel); err != nil {
		log.Printf("[warning] cannot set log level: %q (%v)", logLevel, err)
	}

	mng := manager.New()
	ui := tui.NewTui(mng)

	if err := ui.Run(tui.List); err != nil {
		log.Fatal(err)
	}
}

func shutdown(server *fuse.Server, signals <-chan os.Signal) {
	<-signals
	if err := server.Unmount(); err != nil {
		log.Printf("[warning] server unmount failed: %v", err)
		os.Exit(1)
	}

	log.Printf("[info] Unmount successful.")
	os.Exit(0)
}
