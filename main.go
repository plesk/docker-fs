package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/plesk/docker-fs/lib/log"

	"github.com/plesk/docker-fs/lib/dockerfs"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

var (
	// Docker container ID (or name)
	containerId string

	// Directory to mount container FS
	mountPoint string

	//
	dockerSocketAddr string

	logLevel       string
	verbose, quiet bool
)

func init() {
	flag.StringVar(&containerId, "id", "", "Docker containter ID (or name)")
	flag.StringVar(&containerId, "i", "", "Docker containter ID (or name)")

	flag.StringVar(&mountPoint, "mount", "", "Mount point for containter FS")
	flag.StringVar(&mountPoint, "m", "", "Mount point for containter FS")

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

	if containerId == "" {
		fmt.Fprintf(os.Stderr, "Container id is not specified.\n")
		flag.Usage()
		os.Exit(2)
	}

	if mountPoint == "" {
		fmt.Fprintf(os.Stderr, "Mount point is not specified.\n")
		flag.Usage()
		os.Exit(2)
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

	log.Printf("[info] Check if mount directory exists (%v)...", mountPoint)
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		log.Fatal(err)
	}

	log.Printf("[info] Fetching content of container %v...", containerId)
	dockerMng := dockerfs.NewMng(containerId)
	if err := dockerMng.Init(); err != nil {
		log.Fatalf("dockerMng.Init() failed: %v", err)
	}

	root := dockerMng.Root()

	log.Printf("Mounting FS to %v...", mountPoint)
	server, err := fs.Mount(mountPoint, root, &fs.Options{})
	if err != nil {
		log.Fatalf("Mount failed: %v", err)
	}

	log.Printf("[info] Setting up signal handler...")
	osSignalChannel := make(chan os.Signal, 1)
	signal.Notify(osSignalChannel, syscall.SIGTERM, syscall.SIGINT)
	go shutdown(server, osSignalChannel)

	log.Printf("OK!")
	log.Printf("Press CTRL-C to unmount docker FS")
	server.Wait()
	log.Printf("[info] Server finished.")
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
