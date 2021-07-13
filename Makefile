help:
	echo "=== Download goreleaser from https://github.com/goreleaser/goreleaser/releases ===="
	echo "=== Check https://goreleaser.com/ for alternative installation instructions ==== "
	echo "make build - build bastion cli requires Go 1.15"
	echo "make snapshot - create release binaries"

build:
	go build -o bin/bastion

install:
	go build -o bastion && cp ./bastion /usr/local/bin/bastion

snapshot:
	goreleaser --snapshot --rm-dist

release:
	goreleaser --rm-dist

