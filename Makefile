VERSION := 1.3.7
PREFIX := data/data/com.termux/files/usr
DEB_NAME := music-dl_$(VERSION)_aarch64.deb
DEBIAN_DIR := deb-package/DEBIAN
BIN_DIR := deb-package/$(PREFIX)/bin

deb:
	go build -ldflags="-s -w" -o $(BIN_DIR)/music-dl ./cmd/music-dl
	dpkg-deb --build deb-package $(DEB_NAME)

fmt:
	go fmt ./...

vet:
	go vet ./...

build:
	go build -o bin/music-dl ./cmd/music-dl