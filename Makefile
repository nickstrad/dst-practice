# Common commands. Run `make` or `make help` to list them.

SEED ?= 0
RUNS ?= 1000
FUZZTIME ?= 10s
FUZZ_TARGETS := FuzzBackoff FuzzPolicyValidation FuzzDo

.PHONY: help test test-seeds seed fuzz cover vet fmt demo clean

help: ## show this list
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk -F ':.*## ' '{printf "  %-12s %s\n", $$1, $$2}'

test: ## run all tests
	go test ./...

test-seeds: ## run the seeded invariant tests with more seeds (RUNS=100000)
	go test ./retry -run TestInvariantsAcrossSeeds -runs $(RUNS)

seed: ## replay one seed (SEED=42)
	@if [ "$(SEED)" = "0" ]; then echo "usage: make seed SEED=<n>"; exit 1; fi
	go test ./retry -run TestInvariantsAcrossSeeds -v -seed $(SEED)

fuzz: ## run each fuzz target for FUZZTIME (default 10s)
	@for f in $(FUZZ_TARGETS); do \
		echo "== $$f"; \
		go test ./retry -run '^$$' -fuzz "^$$f$$" -fuzztime $(FUZZTIME) || exit 1; \
	done

cover: ## statement coverage per function (Go has no MC/DC tool)
	go test ./... -coverprofile=cover.out
	go tool cover -func=cover.out

vet: ## gofmt check and go vet
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed:"; gofmt -l .; exit 1)
	go vet ./...

fmt: ## gofmt the tree
	gofmt -w .

demo: ## run the retry demo against the real clock
	go run ./cmd/retrydemo

clean: ## remove build and test artifacts
	go clean ./...
	rm -f *.test *.out
