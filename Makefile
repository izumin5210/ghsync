.DEFAULT_GOAL := build

PATH := ${PWD}/bin:${PATH}
export PATH
export GO111MODULE=on

.PHONY: gen
gen:
	go generate ./...

.PHONY: build
build:
	go build -o=./bin/ghsync ./cmd/ghsync

.PHONY: install
install:
	go install ./cmd/ghsync

.PHONY: test
test:
	go test -v ./...

.PHONY: cover
cover:
	go test -v -coverprofile coverage.txt -covermode atomic ./...
