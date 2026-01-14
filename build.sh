#!/bin/bash -e
# Tests coredhcp then cross-compiles it for various targets.
# All the binaries are produced in the `bin` directory, grouped by architecture.

cd "$(dirname "$0")"
BIN=$PWD/bin

go test ./...

# a space-separated list
REQUIRED_OS="linux"

# a space-separated list
REQUIRED_TARGETS="amd64 arm64 arm"

if [ $# -gt 0 ]; then
  REQUIRED_TARGETS="$@"
fi

export CGO_ENABLED=0

cd cmds
for os in $REQUIRED_OS; do
  for arch in $REQUIRED_TARGETS; do
    echo "$os/$arch"
    mkdir -p $BIN/$os/$arch
    for cmd in *; do
      if [ -d $cmd ]; then
        echo "  $cmd"
        GOOS=$os GOARCH=$arch go build -o $BIN/$os/$arch/$cmd ./$cmd
      fi
    done
  done
done

