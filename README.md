# dst-practice

Small systems built to practice deterministic simulation testing (DST).
See `docs/options.md` for the full list of projects and the patterns each one
exercises.

The rule every system under test follows: it never spawns a goroutine, never
reads the wall clock, and never touches the network or filesystem directly.
It receives a `Clock` (and `Rand`, `Storage`, `Network` as needed) at
construction. Production passes real ones. Tests pass fakes.

## Layout

    projects/       one folder per project from docs/options.md
    projects/retry/ project 1: retry with exponential backoff and full jitter
    sim/clock/      Clock interface, Real clock, Fake clock, architecture.md
    cmd/retrydemo/  runs the retrier against the real clock
    docs/           coding style, knowledge base, and docs for agents
    AGENTS.md       start here if you are an AI agent

Readers and agents: start at `docs/index.md` for the docs.

## Usage

    make help                # list all targets
    make test                # go test ./...
    make test-seeds RUNS=100000   # more random seeds
    make seed SEED=42        # replay one seed
    make demo                # watch the retrier in real time
    make vet                 # gofmt check and go vet

Without make:

    go test ./...
    go test ./projects/retry -runs 100000
    go test ./projects/retry -run TestInvariantsAcrossSeeds -seed 42
    go run ./cmd/retrydemo
