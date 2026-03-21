TAILWIND_BIN := bin/tailwindcss
INPUT_CSS := src/portfolio_manager/web/tailwind/input.css
OUTPUT_CSS := src/portfolio_manager/web/static/css/app.css

.PHONY: setup css-watch css-build dev

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
	@echo "Starting CSS watcher in background..."
	$(TAILWIND_BIN) -i $(INPUT_CSS) -o $(OUTPUT_CSS) --watch &
	uv run portfolio-web
