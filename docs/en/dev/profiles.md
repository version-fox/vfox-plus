# Plugin profiles

> [简体中文](../../zh-CN/dev/profiles.md)

vfox lists, installs, selects and removes versions the same way for every
plugin, but plugins offer more than that, and none of it shows in the output of
vfox: `vfox search java tem` lists one distribution of Java where `vfox search
java` lists OpenJDK only, `nodejs@24` installs the newest Node.js 24, and
plugins read environment variables that move their downloads to a mirror. A
**profile** writes this down as data: a JSON file that every interface of vfox+
reads. Adapting a plugin, or a new release of one, is a change of data, not of
the application.

The format is [schema.json](../../../shared/profile/schema.json), which editors
use when a file names it in `"$schema"`.

## Where profiles come from

For an installed plugin, each section of the profile comes from the first of
these that carries it:

| Priority | Source | Where |
| --- | --- | --- |
| 1 | The user | `profiles/<plugin>.json` next to the settings (`~/.config/vfox-plus/profiles/` on Linux, `%APPDATA%\vfox-plus\profiles\` on Windows) |
| 2 | The plugin | `lib/vfox-plus.json` in the installed plugin, which vfox packs and updates with it |
| 3 | The online directory | [`shared/profile`](../../../shared/profile) on the main branch of this repository |
| 4 | The application | the same files as it shipped them |
| 5 | Inference | only for a plugin that neither 3 nor 4 describes |

A section is taken whole: a file that has `variables` replaces every variable
of the sources after it, and leaves their guide, tips and other sections. An
empty list, such as `"tips": []`, hides the section.

A plugin is looked up by the name that vfox knows it by, and when nothing has
that name, by the name that the plugin declares, so that a plugin added under an
alias (`vfox add --alias`) keeps its profile.

**The online directory.** The application fetches `index.json`, which lists
every profile with the SHA-256 of its file, at most once a day (after an hour
when a fetch fails), through the update mirror of the settings, such as
`https://gh-proxy.org/`. It downloads only the files that differ from those it
shipped, checks them against the index, and keeps them in the cache of the user
(`vfox-plus/profiles`). The cache is dropped when the application ships other
profiles, so that an update of the application is not overridden by an older
cache.

**Inference.** For a plugin that neither the directory nor the application
describes, what no other source carries is inferred: the variables from the
`os.getenv` calls of the installed plugin that name a setting (`VFOX_*`, a
mirror, a base address, a proxy), the version files from
`PLUGIN.legacyFilenames` of `metadata.lua`, the guide and the documentation from
the description and homepage of `vfox info`. For every plugin, a profile without
filters gets those that the listed versions give (notes, and shared `-suffix`
tails).

**Capabilities** come from the files of the plugin: `vfox link --detect` is
offered unless the plugin has hooks without `hooks/detect.lua` (or, for a plugin
of one file, a `main.lua` without a `Detect` function).

## The format

A profile is a JSON object of at most 256 KiB:

```json
{
  "$schema": "https://raw.githubusercontent.com/version-fox/vfox-plus/main/shared/profile/schema.json",
  "schema": 2,
  "plugin": "example",
  "variables": [
    {
      "key": "VFOX_EXAMPLE_MIRROR",
      "label": {"en": "Download mirror", "zh": "下载镜像"},
      "help": {"en": "A mirror of example.org/dist.", "zh": "example.org/dist 的镜像。"},
      "presets": [{"value": "https://mirror.example.org/dist", "name": "Example"}]
    }
  ],
  "guide": {"en": "Write the release as the list shows it.", "zh": "按列表中的样子写版本。"}
}
```

`plugin` names the plugin of a file of the user or of the directory; the file
that a plugin ships may leave it out, since it describes the plugin that it is
in. Unknown fields are ignored, so that a newer file stays readable. A part that
cannot be read costs only itself: a bad item its item, a section with nothing
good in it that section, which then comes from the next source.

**Texts** are a string, which every language shows, or an object by language:
`{"en": "Mirror", "zh": "镜像"}`. The application has English and Simplified
Chinese; English is shown where Chinese is missing.

**Platforms.** Most parts of a list take `"when": {"os": [...], "arch": [...]}`,
as `GOOS` and `GOARCH` name them (a string or a list; a missing list stands for
every one). The platform is the one that vfox runs on.

| Section | Content |
| --- | --- |
| `search` | What follows the plugin name in `vfox search`: `label`, `default` (the choice that vfox lists without an argument), `choices` of `{value, name, maker, aliases, popular, everything}`. A list of searches with `when` gives platforms their own; the first for the platform wins |
| `naming` | How version names are read (below), or `"plain"` |
| `shorthands` | Names that `vfox install <plugin>@<name>` takes besides the listed versions: `{name, meaning}` |
| `variables` | The environment variables that the plugin reads (below) |
| `legacyFiles` | The version files that vfox reads through the plugin |
| `legacyStrategy` | Whether the setting `legacyVersionFile.strategy` of vfox changes what the plugin does with them |
| `facets` | The filters of the listed versions: `{id, label, kind, values, pattern}`; `kind` is `note` (the parts of the note, split at commas and semicolons), `suffix` (what follows the last `-` of the version) or `pattern` (what the first group of `pattern`, or its group `value`, finds in the version) |
| `guide` | A few sentences on the version names |
| `tips` | What is good to know before the first install |
| `docs` | Documentation: `{title, url}`, where `url` is an `https://` address or one by language |

### Variables

`key`, `label`, `help`, `advanced` (for unusual systems), `example` (shown in an
empty field) and `kind`:

| `kind` | The value | Fields |
| --- | --- | --- |
| `url` (default) | an `http://` or `https://` address | `presets`, `https://` only |
| `choice` | one of the presets | `presets` of `{value, name}` |
| `toggle` | on or off | `on` (default `"1"`), `off` (empty: unset) |
| `text` | one line of text | `presets` |

### Naming

`naming` takes a version name apart, so that `21.0.12.1+1-tem` shows Temurin and
`3.47.7-x64` shows x64:

```json
"naming": {
  "sources": [{"value": "tem", "name": "Temurin"}, {"value": "open", "name": "OpenJDK"}],
  "defaultSource": "open",
  "rules": [
    {"pattern": "^(?<version>.*?)(?:-(?<source>{sources}))?(?:-(?<flavor>fx))?$", "labels": {"fx": "JavaFX"}}
  ]
}
```

The first rule whose `pattern` (a regular expression of Go, RE2) matches a name
reads it. The group `version` is the version, the whole name without it; the
group `source` names one of `sources`, `defaultSource` when it matches nothing;
every other named group that matches adds a flavor, called as `labels` says (by
the captured text in lower case). `flavors` adds fixed ones. `{sources}` stands
for the alternation of the values of the sources.

## What a profile may not do

A profile is data, held to the same rules wherever it comes from:

- It may not offer variables that change what runs: `PATH`, `HOME` and the other
  directories of the user, the shell (`BASH_ENV`, `PROMPT_COMMAND`, …), the
  dynamic loader (`LD_*`, `DYLD_*`), `VFOX_HOME` and `__VFOX_*`, the options of
  language runtimes (`NODE_OPTIONS`, `JAVA_TOOL_OPTIONS`, `PYTHONPATH`, …),
  compilers and build tools (`CC`, `MAKEFLAGS`, …), proxies and certificates,
  Git, npm and tokens. The list is in
  [security.go](../../../mygo/core/profile/security.go). Such settings belong to
  the environment variables of the settings, where the user types them.
- Documentation links and the preset addresses of `url` variables are
  `https://` only. Search choices and shorthands are single arguments: no
  leading `-` and no spaces.
- A value that the application did not ship, from the plugin, the directory or a
  file of the user, is shown and confirmed before it is set. Typing a value is
  the user's own choice and asks nothing.

## Describing your plugin

Add `lib/vfox-plus.json` to the plugin: vfox packs `lib`, so the profile arrives
with the plugin and works offline, in every vfox plus that has the plugin. Check
it against the plugin with a checkout of this repository:

```bash
go -C shared/tooling run ./cmd/vfxtool profile lint --plugin /path/to/vfox-example
```

or in the CI of the plugin:

```yaml
jobs:
  profile:
    uses: version-fox/vfox-plus/.github/workflows/profile-lint.yml@main
```

The check fails for what the application cannot read, for a key twice on one
platform and for a variable that no file of the plugin names, besides its
profile and its documents.

## Changing the profiles of this repository

The plugins of the version-fox organisation have their profiles in
[`shared/profile/plugins`](../../../shared/profile/plugins). After a change:

```bash
go -C shared/tooling run ./cmd/vfxtool profile lint --strict
go -C shared/tooling run ./cmd/vfxtool profile index
```

`index` writes `shared/profile/index.json`; the tests of `core/profile` fail
while it is stale. Installed applications get a change within a day, from the
online directory, and the next release ships it.
