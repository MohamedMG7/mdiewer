# mdiewer

`mdiewer` is a small terminal Markdown viewer written in Go. It renders a Markdown file directly in the terminal with ANSI styling.

## Features

- Headings
- Paragraph wrapping
- Bold, italic, inline code, and links
- Ordered and unordered lists
- Block quotes
- Fenced code blocks
- Horizontal rules
- Basic pipe tables
- Normal mode: `mdiewer <filename.md>`
- Full-screen mode: `mdiewer -f <filename.md>`

## Usage

```sh
mdiewer README.md
```

Clear the terminal before rendering:

```sh
mdiewer -f README.md
```

Show help:

```sh
mdiewer --help
```

## Install

Windows:

```powershell
irm https://raw.githubusercontent.com/MohamedMG7/mdiewer/main/scripts/install.ps1 | iex
```

Linux/macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/MohamedMG7/mdiewer/main/scripts/install.sh | sh
```

Build a local binary:

```sh
go build -buildvcs=false -o mdiewer.exe .
```

## Release

Releases are automated with GitHub Actions and GoReleaser. Push a version tag:

```sh
git tag v0.1.0
git push origin v0.1.0
```

See [docs/releasing.md](docs/releasing.md).
