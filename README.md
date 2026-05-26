# ink

Markdown files in a local library can be listed and opened as styled HTML in a browser.

## Usage

```sh
ink ls
ink view filename.md
ink edit filename.md
ink edit a.md b.md

ink config set library ~/Downloads ~/Documents/notes
ink config add library ~/work/docs
ink config remove library ~/Downloads
ink config get library
ink config list
```

List markdown files in the configured library:

```sh
ink ls
```

Open a markdown file in the browser:

```sh
ink view filename.md
```

`filename.md` is resolved relative to the configured `library` directory.
If multiple libraries contain a file with the same name, `ink view` and `ink edit` stop and print recommended absolute-path commands. Run one of them to choose the file explicitly:

```sh
ink view /Users/me/Downloads/filename.md
ink view /Users/me/Documents/notes/filename.md
```

Edit markdown files in the browser:

```sh
ink edit
ink edit filename.md
ink edit a.md b.md
```

`ink edit` starts one local editor server on `127.0.0.1` and opens a browser workspace. The file list shows `.md` files directly under the configured libraries. Click a file to open it in a tab, edit it, and press `Save` to write changes back to the original markdown file. Keep the command running while editing, and stop it with `Ctrl-C` when finished.

Set the markdown library directory:

```sh
ink config set library ~/Downloads ~/Documents/notes
```

Add or remove one library directory:

```sh
ink config add library ~/work/docs
ink config remove library ~/Downloads
```

Show one config value:

```sh
ink config get library
```

Show all config values:

```sh
ink config list
```

If `library` is not configured, `ink` uses `~/Downloads`.

## Installation

Install with Homebrew:

```sh
brew tap econron/tap
brew install ink
```

Check the installed command:

```sh
ink --help
```

## Files

`ink` stores user configuration under:

```txt
~/.ink/config.json
```

The config file stores library directories as an array:

```json
{
  "library": [
    "/Users/me/Downloads",
    "/Users/me/Documents/notes"
  ]
}
```

Rendered HTML previews are written under:

```txt
~/.ink/cache/pages/
```

Each markdown file gets a stable cache HTML file based on its absolute path. Opening the same markdown again overwrites the same preview file, while different markdown files can remain open in separate browser tabs.

On each `ink view`, cache HTML files older than 7 days are removed automatically. The file opened in the browser is not deleted immediately, so browser reloads continue to work.

Do not run `ink` with `sudo`. Configuration and cache files are intended to be created by the current user under that user's home directory.

## Development

Build the CLI from this repository:

```sh
make build
```

This creates an `ink` binary in the repository root. Run the local binary as:

```sh
./ink --help
./ink ls
./ink view filename.md
./ink edit filename.md
./ink edit a.md b.md
```

Run tests:

```sh
make test
```

Run lint:

```sh
make lint
```

## Release

Tagged releases are built with GoReleaser from GitHub Actions.

Before the first release, create the tap repository:

```sh
gh repo create econron/homebrew-tap --public
```

The release workflow needs a repository secret named `TAP_GITHUB_TOKEN`. It must have contents write access to `econron/homebrew-tap`.

To publish a release, push a `v*` tag from `main`:

```sh
git checkout main
git pull origin main
git tag v0.1.0
git push origin v0.1.0
```

GitHub Releases are created in `econron/ink`. The Homebrew formula is published to `econron/homebrew-tap`.
