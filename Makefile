.PHONY: go-tools go-gen go-build go-vet go-test go-cover go-lint go-check

GOBIN := $(shell go env GOPATH)/bin

# --- Go ---------------------------------------------------------------------

## Install code-generation tooling into $(GOBIN)
go-tools:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

## Regenerate sqlc query code
go-gen:
	$(GOBIN)/sqlc generate

go-build:
	go build ./...

go-vet:
	go vet ./...

go-test:
	go test ./...

## Coverage with the 85% gate (excludes generated: sqlc, cmd, container, models)
go-cover:
	@PKGS=$$(go list ./... | grep -vE '/(db/sqlc|cmd|container|models)($$|/)' | tr '\n' ' '); \
	go test $$PKGS -coverprofile=coverage.out -covermode=atomic; \
	total=$$(go tool cover -func=coverage.out | awk '/^total:/ {gsub("%","",$$3); print $$3}'); \
	echo "total coverage: $$total%"; \
	awk "BEGIN { exit ($$total < 85.0) }" || { echo "coverage below 85%"; exit 1; }

go-lint:
	golangci-lint run ./...

## Full local gate: generate, build, vet, lint, test
go-check: go-gen go-build go-vet go-lint go-test
