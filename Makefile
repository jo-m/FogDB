SHELL := bash
.SHELLFLAGS := -o pipefail -c

.PHONY: format
format:
	gofmt -w .
	go mod tidy

.PHONY: lint
lint:
	@go mod tidy -diff
	@gofmt -l .; test -z "$$(gofmt -l .)"
	@go vet ./...
	@go tool staticcheck -f stylish ./...
	@go tool revive -set_exit_status -formatter stylish $(shell go list ./... | grep -v 'frontend/')
	@go tool govulncheck ./...
	@go tool gosec -quiet -exclude G101,G304 ./...

.PHONY: test
test:
	go test ./...

.PHONY: bench
bench:
	go test -bench=. -run=Bench ./...

.PHONY: check
check: lint
	go build ./...
	@go test ./... | { grep -vE '^(ok|\?)' || true; }
