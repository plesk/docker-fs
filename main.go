package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/hanwen/go-fuse/v2/fs"
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
		os.Exit(2)
	}

	if mountPoint == "" {
		fmt.Fprintf(os.Stderr, "Mount point is not specified.\n")
		os.Exit(2)
	}

	log.Printf("Fetching content of container %v...", containerId)
	file, err := FetchContainerArchive(containerId)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Creating FS tree from archive (%v)...", file)
	root, err := NewTarTree(file)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Mounting FS to %v...", mountPoint)
	server, err := fs.Mount(mountPoint, root, &fs.Options{})
	if err != nil {
		log.Fatalf("Mount failed: %v", err)
	}

	log.Printf("OK!")
	server.Wait()
}
