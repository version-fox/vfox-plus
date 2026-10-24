# Contributing to vfox-plus

Thank you for helping! This repository holds the vfox-plus application (see
the [README](README.md)); most of what follows applies to all of its parts.

## Ground rules

- Code comments, documentation, commit messages and pull requests are written in
  **English**. User-visible texts go through the translation catalogues in
  [`shared/i18n`](shared/i18n) and are available in English and Simplified
  Chinese.
- Every source file starts with the Apache-2.0 license header used in this
  repository. Check with
  `docker run --rm -v "$PWD:/github/workspace" apache/skywalking-eyes:0.9.0 header check`
  and add missing headers with `header fix`.
- Runtime dependencies must be licensed under terms compatible with the Apache
  License 2.0.
- Behaviour shared by the layers — the vfox CLI contract, screens,
  design tokens, translations, icons, settings — is specified in [`docs`](docs)
  and [`shared`](shared). Change the specification together with the code.
- Files use UTF-8, LF line endings and a final newline (see
  [`.editorconfig`](.editorconfig) and [`.gitattributes`](.gitattributes)).
- Binary files are not committed. Upload screenshots and other images to an
  image host and link them (see [Screenshots](#screenshots)); generate binary
  build inputs, such as the icon fonts, at build time.
  CI rejects binary files.

## Workflow

1. Open an issue or comment on an existing one before large changes.
2. Create a branch from `main` and keep pull requests focused.
3. Run the checks; they are the same as in CI and listed in the
   [mygo README](mygo/README.md#everyday-commands):

   ```sh
   cd mygo
   golangci-lint fmt --diff
   golangci-lint run && golangci-lint run --build-tags e2e
   CGO_ENABLED=1 go test -race -coverprofile=coverage.out ./... && bash tools/coverage.sh coverage.out 85
   ```

   To run every check, the end-to-end tests and the packaging without installing
   anything but Docker, and without any risk for the vfox installation on your
   machine, use the sandbox, `mygo/sandbox/sandbox.sh all` (see
   [shared/sandbox](shared/sandbox/README.md)).

4. Describe what changed and how you tested it in the pull request. Include
   screenshots for visual changes, in light and dark themes.

## Continuous integration

| Workflow | Runs on | Checks |
| --- | --- | --- |
| [`repository.yml`](.github/workflows/repository.yml) | every change | License headers (SkyWalking Eyes), workflows (actionlint), shell scripts (ShellCheck), shared JSON, no committed binary files, the tests of the memory report |
| [`memory.yml`](.github/workflows/memory.yml) | the end of the build workflow, on pull requests | One comment on the pull request with the memory of the application, and the difference from `main` (see [Memory in pull requests](docs/memory.md#memory-in-pull-requests)) |
| [`mygo.yml`](.github/workflows/mygo.yml) | `mygo/`, `shared/` | Formatting and lint (golangci-lint, with and without the end-to-end build tag), unit tests with the race detector and the coverage gates, the licenses of the Go modules, end-to-end scenarios in the application against captured output and against the real vfox, the memory measurement, release build with a smoke test, `.deb` and `.tar.gz` validated with lintian and installed into a clean Ubuntu container |
| [`sandboxes.yml`](.github/workflows/sandboxes.yml) | `mygo/sandbox/`, `mygo/versions.env`, `shared/sandbox/`, `shared/e2e/` | Builds the Docker sandbox and runs `sandbox.sh all` in it |

Dependabot keeps the Go modules and the GitHub Actions up to date; actions are
pinned to commit SHAs.

The workflows install their Ubuntu packages with
[`install-linux-packages`](.github/actions/install-linux-packages/action.yml), which
keeps the downloaded archives in a cache (one for each list of packages and each
week) and downloads what the cache lacks eight files at a time, continuing what was cut
and trying the next mirror when one is slow, in 15 minutes at most. The Ubuntu mirror of
the runners is slow for some of them at times: on 2026-10-23 a job got 40 KB/s for a
connection while others downloaded 90 MB in seconds. apt does not check an archive that is in the
directory, so the action checks every archive by its hash. A run that cannot finish
saves what it downloaded, so that the next one gets further.

A pull request that rebuilds the application gets a comment with its memory and
the difference from `main`, which the next commit replaces instead of adding
another one. A change that is more than the noise between CI runs is in bold; read
[Memory in pull requests](docs/memory.md#memory-in-pull-requests) before you trust a small
one.

## Common changes

### Texts

Add or change messages in [`shared/i18n/app_en.arb`](shared/i18n/app_en.arb)
with a `description` (and placeholder metadata), then translate them in
[`app_zh.arb`](shared/i18n/app_zh.arb). See [docs/i18n.md](docs/i18n.md),
including the glossary.

### Icons

Add the Material Symbol to [`shared/icons.json`](shared/icons.json) under a
name that describes its meaning, then regenerate the icon fonts
(`cd mygo && bash tools/prepare.sh`, which cuts them with
[`shared/tools/icon_font.py`](shared/tools/icon_font.py) when it builds).
Fonts are generated at build time and not committed.

### vfox output

When vfox changes, recapture the fixtures with
[`shared/fixtures/capture.sh`](shared/fixtures/capture.sh) and follow
[`shared/fixtures/README.md`](shared/fixtures/README.md). Document the change in
[docs/vfox-cli-contract.md](docs/vfox-cli-contract.md).

### Screenshots

The images of the documentation are not committed: they are hosted on
[freeimage.host](https://freeimage.host), a free image host that needs no
account, and linked. `mygo/sandbox/sandbox.sh screenshots` renders the
screenshots into `dist/sandbox/mygo/screenshots/`, reproducibly: every page, in both
themes and languages, the compact layout and an installation in progress, with
the Greybird title bar of xfwm4, the way modern GNOME applications look.
The [README](README.md) links `home`, `sdks`, `dark-available`,
`installing`, `tasks`, `plugins`, `zh-settings`,
`compact-home` and `compact-detail`. Upload a new image with

```sh
curl -s -F key=6d207e02198a847aa98d0a2a901485a5 -F action=upload -F format=json \
  -F source=@dist/sandbox/mygo/screenshots/home.png https://freeimage.host/api/1/upload |
  python3 -c 'import json, sys; print(json.load(sys.stdin)["image"]["url"])'
```

which prints its URL (the key is the public one that freeimage.host gives to
everyone on its [API page](https://freeimage.host/api)), and replace the link in
the README. If curl stops with error 56, the connection broke: run it again.
Every upload gets a new URL, so cached copies of an old image never show. The
uploader of an anonymous image cannot delete it; upload nothing that must not
stay public.
