TAILWIND_BIN := bin/tailwindcss
INPUT_CSS := src/portfolio_manager/web/tailwind/input.css
OUTPUT_CSS := src/portfolio_manager/web/static/css/app.css

.PHONY: setup css-watch css-build dev \
	go-tools go-gen go-build go-vet go-test go-cover go-lint go-run go-check

GOBIN := $(shell go env GOPATH)/bin

## Download Tailwind CLI and DaisyUI
setup:
	bash scripts/setup-tailwind.sh

## Watch mode: rebuild CSS on template changes
css-watch:
	$(TAILWIND_BIN) -i $(INPUT_CSS) -o $(OUTPUT_CSS) --watch

## Build minified CSS for production
css-build:
	$(TAILWIND_BIN) -i $(INPUT_CSS) -o $(OUTPUT_CSS) --minify

## Run web server + CSS watcher together
dev:
	@$(TAILWIND_BIN) -i $(INPUT_CSS) -o $(OUTPUT_CSS) --watch & \
	TAILWIND_PID=$$!; \
	trap "kill $$TAILWIND_PID 2>/dev/null" EXIT; \
	uv run portfolio-web

# --- Go ---------------------------------------------------------------------

## Install code-generation tooling into $(GOBIN)
go-tools:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/a-h/templ/cmd/templ@latest

## Regenerate sqlc query code and templ templates
go-gen:
	$(GOBIN)/sqlc generate
	$(GOBIN)/templ generate

go-build:
	go build ./...

go-vet:
	go vet ./...

go-test:
	go test ./...

## Coverage with the 85% gate (matches the Python project's cov-fail-under)
go-cover:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	@total=$$(go tool cover -func=coverage.out | awk '/^total:/ {gsub("%","",$$3); print $$3}'); \
	echo "total coverage: $$total%"; \
	awk "BEGIN { exit ($$total < 85.0) }" || { echo "coverage below 85%"; exit 1; }

go-lint:
	golangci-lint run ./...

## Full local gate: generate, build, vet, lint, test
go-check: go-gen go-build go-vet go-lint go-test

go-run:
	go run ./cmd/portfolio-web
