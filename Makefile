.PHONY: *

test:
	go test -v $(shell go list ./... | grep -v e2e)

test-e2e:
	go test -v $(shell go list ./... | grep e2e) -timeout 60m -count 1

build:
	goreleaser release --clean --skip=publish --snapshot

release:
	goreleaser release --clean
