# Docker-fs.

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

# Technical details and limitations.

- `docker-fs` works via docker API, so it can work with either local or remote docker servers.
(currently only local docker through unix-socket is implemented).

- File system is implemented using [GO-FUSE](https://github.com/hanwen/go-fuse) library which implements FUSE (File systems in USEr space) protocol.

- Currently docker-fs supports only READ operations over mounted FS. But docker API allows modification on files so possibly it will be made in future.

- Directories and regular files are well supported. Other types (i.e. symlinks) support is in progress.

- Empty directories are not shown due to current implementation.

# TODO

- Fix ussie with newly added directories.
