# rig-core

Shared Go library for rig applications.

`rig-core` contains reusable infrastructure only. Application-specific logic
and user interfaces belong in the individual app repositories.

## Module

    github.com/isfopo/rig-core

## Packages

- `audio` — device enumeration, capture/playback/duplex streams, meter helpers
- `midi` — MIDI engine (RtMidi/test drivers) and monitor
- `tempo` — internal clock, timeline, and beat/bar grid
- `config` — shared TOML configuration loading helpers

## Usage

Consume `rig-core` as a git submodule in an app repository:

    git submodule add https://github.com/isfopo/rig-core.git core

Then reference it from `go.mod`:

    require github.com/isfopo/rig-core v0.0.0
    replace github.com/isfopo/rig-core => ./core

## Build

Requires Go >= 1.24 and a C compiler for CGO (RtMidi/miniaudio):

    go test ./...
    go vet ./...
