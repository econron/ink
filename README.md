# ink

Markdown files in a local library can be listed and opened as styled HTML in a browser.

## Setup

Build the CLI from this repository.

```sh
go build -o ink ./cmd
```

This creates an `ink` binary in the repository root. During local development, run commands as:

```sh
./ink <command>
```

Run the test suite with:

```sh
go test ./...
```

## Configuration

`ink` stores user configuration under:

```txt
~/.ink/config.json
```

Set the markdown library directory:

```sh
./ink config set library ~/Downloads
```

Show one config value:

```sh
./ink config get library
```

Show all config values:

```sh
./ink config list
```

If `library` is not configured, `ink` uses `~/Downloads`.

## Usage

List markdown files in the configured library:

```sh
./ink ls
```

Open a markdown file in the browser:

```sh
./ink view filename.md
```

`filename.md` is resolved relative to the configured `library` directory.

## Cache

Rendered HTML previews are written under:

```txt
~/.ink/cache/pages/
```

Each markdown file gets a stable cache HTML file based on its absolute path. Opening the same markdown again overwrites the same preview file, while different markdown files can remain open in separate browser tabs.

On each `ink view`, cache HTML files older than 7 days are removed automatically. The file opened in the browser is not deleted immediately, so browser reloads continue to work.

## Release

Tagged releases are built with GoReleaser from GitHub Actions.

To publish a release:

```sh
git tag v0.1.0
git push origin v0.1.0
```

GitHub Releases are created in `econron/ink`. The Homebrew formula is published to the tap repository `econron/homebrew-tap`.

Before the first release, create the tap repository:

```sh
gh repo create econron/homebrew-tap --public
```

Homebrew users can install it with:

```sh
brew tap econron/tap
brew install ink
```

The release workflow needs a repository secret named `TAP_GITHUB_TOKEN`. It must have contents write access to `econron/homebrew-tap`.

## Notes

Do not run `ink` with `sudo`. Configuration and cache files are intended to be created by the current user under that user's home directory.
