.PHONY: build run test lint tidy clean frontend ensure-frontend-stub

BIN := bin/clipfolio

ensure-frontend-stub:
	@mkdir -p internal/api/dashboarddist internal/api/playerdist
	@[ -f internal/api/dashboarddist/index.html ] || printf '<!doctype html><title>clipfolio</title><body>Run `make frontend` to build the dashboard.</body>' > internal/api/dashboarddist/index.html
	@[ -f internal/api/playerdist/player.js ] || printf 'console.warn("clipfolio: run \`make frontend\` to build the embeddable player");' > internal/api/playerdist/player.js

frontend:
	cd web/dashboard && npm install && npm run build
	cd web/player && npm install && npm run build

build: ensure-frontend-stub
	go build -o $(BIN) ./cmd/clipfolio

run: ensure-frontend-stub
	go run ./cmd/clipfolio

# -p 1 is required, not just faster-safe: internal/db and internal/api both
# run integration tests against the *same* CLIPFOLIO_TEST_DATABASE_URL and
# truncate shared tables, so running packages concurrently corrupts each
# other's fixtures.
test: ensure-frontend-stub
	go test -p 1 ./...

lint: ensure-frontend-stub
	golangci-lint run

tidy:
	go mod tidy

clean:
	rm -rf bin internal/api/dashboarddist internal/api/playerdist web/dashboard/dist web/player/dist
