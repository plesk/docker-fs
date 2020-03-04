package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	logdir      string
	contentFile string
)

func init() {
	flag.StringVar(&logdir, "logdir", "/tmp", "log dir")
	flag.StringVar(&contentFile, "content-file", "content.txt", "content file")
}

func main() {
	flag.Parse()
	if err := os.MkdirAll(logdir, 0755); err != nil {
		log.Fatal(err)
	}
	for {
		now := time.Now()
		name := now.Format("15-04")
		d := filepath.Join(logdir, name)
		if err := os.MkdirAll(d, 0755); err != nil {
			log.Print(err)
			time.Sleep(10 * time.Second)
			continue
		}
		logfile := filepath.Join(d, fmt.Sprintf("log.%v.txt", name))
		content := readContent(contentFile)
		func() {
			f, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Print(err)
				return
			}
			defer f.Close()
			fmt.Printf("%v : %v\n", now, content)
			fmt.Fprintf(f, "%v : %v\n", now, content)
		}()
		time.Sleep(7 * time.Second)
	}
}

func readContent(path string) string {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("Error: %v", err)
		return ""
	}
	return strings.TrimSpace(string(content))
}
