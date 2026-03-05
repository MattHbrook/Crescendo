# Crescendo

Music discovery & download app. Bridges [hifi-api](https://github.com/MattHbrook/hifi-api) with a local FLAC library.

## Quick Start

```bash
# Local development
make all            # run full quality gate
make build          # compile binary

# Docker
docker-compose up --build
```

## Architecture

- **Go** single-binary server with Chi router, HTMX, and Pico CSS
- **SQLite** for library index, artist mappings, and download history
- **hifi-api** container for Tidal API access
- **ffmpeg** for HI_RES_LOSSLESS DASH remuxing

## Development

See [CLAUDE.md](CLAUDE.md) for development conventions and quality gates.
