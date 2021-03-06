# Docker-fs.

![Go](https://github.com/plesk/docker-fs/workflows/Go/badge.svg)
[![Coverage Status](https://coveralls.io/repos/github/plesk/docker-fs/badge.svg?branch=master)](https://coveralls.io/github/plesk/docker-fs?branch=master)

Mounts your docker container FS into a local directory.

## Build

Build with go compiler >= 1.12.
```
$ go build
```

## Usage.

Find your running container id with `docker ps`:
```
$ docker ps
CONTAINER ID        IMAGE                       COMMAND                  CREATED             STATUS
a80d96fa4c91        web-installer_development   "./backend --log.for…"   18 hours ago        Up 18 hours
...
```

Mount your container FS into local directory:
```
$ docker-fs --id a80d96fa4c91 --mount ./mnt
...
```

Inspect `./mnt` content with `cd`, `ls`, `cat`, `mc` or any file manager you prefer.

To unmount directory interrupt running `docker-fs` process with `CTRL+C`.

(You can also unmount directory with command `fusermount -u $(pwd)/mnt`.)

## Technical details and limitations.

- `docker-fs` works via docker API, so it can work with either local or remote docker servers.
(currently only local docker through unix-socket is implemented).

- File system is implemented using [GO-FUSE](https://github.com/hanwen/go-fuse) library which implements FUSE (File systems in USEr space) protocol.

- Due to previous point (FUSE) `docker-fs` works on Linux, macOS, and possibly works somehow in WSL on Windows.

- macOS users should install [FUSE for macOS](https://osxfuse.github.io/) first.

- Currently docker-fs supports only reading and modification of existing files over mounted FS.
Creating of new files/directories, setting attributes is going to be done later.

- Directories, regular files and symlinks are well supported. Other types support is in progress.

- Empty directories are not shown due to current implementation.

## TODO

- Fix ussie with newly added directories.

- Tests.

- Add read-only mode.

- Option to make absolute symlinks to point on files inside mount directory.

- Caching.

- Mkdir and file crating support.

- Other FS features...

- Daemonization

- 
