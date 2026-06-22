.DEFAULT_GOAL := build

build:
	CGO_ENABLED=0 go build -o terraform-provider-k0s

test:
	CGO_ENABLED=0 go test ./...

testacc:
	TF_ACC=1 CGO_ENABLED=0 go test ./... -v

generate:
	go generate ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run

.PHONY: build test testacc generate fmt lint
