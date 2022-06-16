SHELL=/usr/bin/env bash

.PHONY: update-plugin
update-plugin:
	@./scripts/update_plugin.sh
.PHONY: test
test:
	go test -v ./...

.PHONY: build
build:
	docker run \
  --rm \
  -e GOARCH=amd64 \
  -e GOOS=linux \
  -w /build \
  -v `pwd`:/build \
  golang:1.18 \
  go build -o /build/bin/commenter cmd/commenter/main.go || exit 1
