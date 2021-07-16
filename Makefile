VERSION ?= latest
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)

help:
	echo "=== Download goreleaser from https://github.com/goreleaser/goreleaser/releases ===="
	echo "=== Check https://goreleaser.com/ for alternative installation instructions ==== "
	echo "make build - build bastion cli requires Go 1.15"
	echo "make snapshot - create release binaries"

build:
	go build -o bin/bastion -ldflags "-X 'github.com/base2Services/bastion-cli/entrypoint.Version=$(VERSION)' -X 'github.com/base2Services/bastion-cli/entrypoint.Build=$(GIT_COMMIT)'"

install: build
	cp ./bin/bastion /usr/local/bin/bastion

snapshot:
	goreleaser --snapshot --rm-dist

release:
	goreleaser --rm-dist

