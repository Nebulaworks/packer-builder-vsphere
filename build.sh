#!/bin/sh

set -eux

glide install -v
export CGO_ENABLED=0
export GOARCH=amd64
mkdir -p bin
rm -f bin/*

cd clone
GOOS=darwin  go build -o bin/packer-builder-vsphere-clone.macos
GOOS=linux   go build -o bin/packer-builder-vsphere-clone.linux
GOOS=windows go build -o bin/packer-builder-vsphere-clone.exe
cd ..

cd iso
GOOS=darwin  go build -o bin/packer-builder-vsphere-iso.macos
GOOS=linux   go build -o bin/packer-builder-vsphere-iso.linux
GOOS=windows go build -o bin/packer-builder-vsphere-iso.exe
cd ..
