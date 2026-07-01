# Releasing mdiewer

Releases are automated with GitHub Actions and GoReleaser.

## One-Time Setup

Create the GitHub repository and push `main`.

The release workflow uses GitHub's built-in `GITHUB_TOKEN`, so no extra token is needed for normal GitHub Releases.

## Create A Release

Tag the version and push the tag:

```sh
git tag v0.1.0
git push origin v0.1.0
```

GitHub Actions will:

- run tests
- build Windows, Linux, and macOS binaries
- create `.zip` and `.tar.gz` archives
- create `checksums.txt`
- publish the GitHub Release

## Install Scripts

Windows:

```powershell
irm https://raw.githubusercontent.com/MohamedMG7/mdiewer/main/scripts/install.ps1 | iex
```

Linux/macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/MohamedMG7/mdiewer/main/scripts/install.sh | sh
```

The scripts install the binary into a user-level PATH location.

## Package Managers

Use the GitHub Release assets as the source for package managers.

- Scoop can point directly to `mdiewer-windows-amd64.zip` and use `"bin": "mdiewer.exe"`. A starter manifest lives at `packaging/scoop/mdiewer.json`.
- Homebrew can point to the macOS/Linux tarballs.
- winget should point to a Windows installer once one exists.
- Chocolatey can package and shim `mdiewer.exe`.
