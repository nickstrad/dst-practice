# make and git can fail with an Xcode license error

## Context
Midway through a session on this Mac, `make vet` and `git status` both
started exiting with code 69 and the message "You have not agreed to the
Xcode license agreements". Both had worked minutes earlier. `go` kept
working.

## Learning
On macOS, `/usr/bin/make` and `/usr/bin/git` are Apple shims that refuse
to run until the current Xcode license is accepted. An Xcode or Command
Line Tools update can reset that acceptance in the middle of a session.

## Why it matters
An agent that sees this error may assume the Makefile or the repo is
broken. Neither is. The fix needs sudo, so the agent cannot apply it.

## How to apply
Tell the user to run this in a terminal, then retry:

    sudo xcodebuild -license accept

Until then, use the Go commands directly. They do not go through the shim:

    go test ./...
    gofmt -l . && go vet ./...
    go run ./cmd/retrydemo

## Evidence
Exit code 69 from `make vet` and `git status` on 2026-09-25.

Resolved the same day after reinstalling Xcode 27.0 (build 27A266a) and
accepting the license. `make test`, `make vet`, and `git status` all ran
cleanly afterward. Treat this entry as a recurring gotcha, not an open
problem.

## Related
`Makefile`, `AGENTS.md` verify section.
