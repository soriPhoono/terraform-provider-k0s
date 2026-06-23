.DEFAULT_GOAL := build

build:
	CGO_ENABLED=0 go build -o terraform-provider-k0s

test:
	CGO_ENABLED=0 go test ./...

testacc:
	TF_ACC=1 CGO_ENABLED=0 go test ./... -v

generate:
	go generate .
	@echo "Docs regenerated in docs/"

fmt:
	go fmt ./...

lint:
	golangci-lint run

release:
	@echo "Tag the release first: git tag v1.0.0 && git push origin v1.0.0"
	@echo "Then run: goreleaser release --clean"

release-dry-run:
	goreleaser release --clean --skip=publish --skip=sign

.PHONY: build test testacc generate fmt lint release release-dry-run
