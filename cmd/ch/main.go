package main

import (
	"docker-fs/docker"
	"fmt"
	"log"
	"os"
)

// usage: containerID
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: <container_id>\n")
		os.Exit(2)
	}
	addr := "/var/run/docker.sock"
	mng := docker.NewMng(addr)
	changes, err := mng.FetchFsChanges(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", changes)
}
