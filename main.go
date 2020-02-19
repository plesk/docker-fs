package main

import (
	"docker-fs/lib/dockerfs"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

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
)

func init() {
	flag.StringVar(&containerId, "id", "", "Docker containter ID (or name)")
	flag.StringVar(&mountPoint, "mount", "", "Mount point for containter FS")
	// TODO make http support
	flag.StringVar(&dockerSocketAddr, "docker-socket", "/var/run/docker.sock", "Docker socket")
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

	log.Printf("Check if mount directory exists (%v)...", mountPoint)
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		log.Fatal(err)
	}

	log.Printf("Fetching content of container %v...", containerId)
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

	log.Printf("Setting up signal handler...")
	osSignalChannel := make(chan os.Signal, 1)
	signal.Notify(osSignalChannel, syscall.SIGTERM, syscall.SIGINT)
	go shutdown(server, osSignalChannel)

	log.Printf("OK!")
	server.Wait()
	log.Printf("Server finished.")
}

func shutdown(server *fuse.Server, signals <-chan os.Signal) {
	<-signals
	if err := server.Unmount(); err != nil {
		log.Printf("[WARN] server unmount failed: %v", err)
		os.Exit(1)
	}

	log.Printf("Unmount successful.")
	os.Exit(0)
}
