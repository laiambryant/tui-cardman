# tui-cardman

A TUI to manage your trading card inventory

## Build

This project uses [go-sqlite3](https://github.com/mattn/go-sqlite3), which is a CGO package. Building requires:

1. A C compiler (GCC or similar)
2. The `CGO_ENABLED=1` environment variable
