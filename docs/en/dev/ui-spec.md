# User interface specification

> [简体中文](../../zh-CN/dev/ui-spec.md)

This document describes what vfox-plus shows and does.
Visual details — colours, shapes, type and motion — are in
[the design system](design-system.md); texts are in
[`shared/i18n`](../../../shared/i18n) and referred to here by their keys, such as
`navHome`. Icons are referred to by their names in
[`shared/icons.json`](../../../shared/icons.json), such as `sdks`.

## Window and navigation

The application has five top-level destinations:

| Destination | Label key     | Icon       | Purpose                                              |
| ----------- | ------------- | ---------- | ---------------------------------------------------- |
| Home        | `navHome`     | `home`     | What is in use and how healthy it is.                |
| SDKs        | `navSdks`     | `sdks`     | Installed plugins and their versions.                |
| Plugins     | `navPlugins`  | `plugins`  | The plugin registry.                                 |
| Tasks       | `navTasks`    | `tasks`    | Running and finished operations. Shows a badge with the number of queued and running tasks. |
| Settings    | `navSettings` | `settings` | Appearance, language, other devices, vfox executable, vfox configuration, about. |

The layout follows the Material 3 window size classes of the window width:

- **Compact** (< 600 dp): a navigation bar at the bottom.
- **Medium and wider** (≥ 600 dp): the navigation rail at the start. Its menu
  button toggles between the collapsed rail (labels below icons) and the
  expanded rail (labels next to icons); the choice is remembered
  (`railExpanded`). The rail holds a floating action button, `navAddPlugin`,
  that opens Plugins.

Pages sit on rounded panes, 16 dp from the window edges. Switching destinations
fades through where the interface draws transitions (the web interface), and switches
at once on the desktop and in the phone app (see [the design system](design-system.md#motion)).
The selected destination's icon is filled;
the others are outlined.

The minimum window size is 360 × 480 dp. The default size is 1280 × 820 dp.

## Common states

Every view that loads data has four states:

- **Loading**: the loading indicator, centred, with a short text when the wait
  can be long (`availableLoading`).
- **Error**: the error icon on a soft-burst shape, a title saying what failed
  (`sdksLoadError`, …), vfox's error message and a retry button
  (`actionRetry`).
- **Empty**: an icon on an expressive shape, a title, an explanation and the
  action that resolves it (for example `sdksEmptyTitle` with a button to
  Plugins).
- **Content**. While reloading, the previous content stays visible.

Long-running operations never block the interface: they become tasks (see
[Tasks](#tasks)).

While the web interface does not receive the events of the application, a chip at
the top says that it reconnects (`reconnecting`); it goes away when they arrive
again. The chip is the only sign of the connection: while it is there, what the
pages show may be old.

## vfox unavailable

While vfox is being located, every page except Settings shows the loading
indicator with `vfoxCheckingTitle`. If no executable is found it shows
`vfoxMissingTitle` / `vfoxMissingBody` with three actions: open the
installation guide (<https://vfox.dev/guides/quick-start.html>), choose the
executable (`vfoxLocate`) and retry. If the executable does not report a
version it shows `vfoxBrokenTitle` / `vfoxBrokenBody` with the same actions.
Settings stay reachable in every case.

## Home

- Header: the application name, `vfox <version> · <executable path>`, and a
  refresh button.
- `homeActiveTitle`: a segmented list of the plugins that have a version in use,
  with their avatar and version. Tapping an item opens it on the SDKs page.
  Without any, the list shows `homeActiveEmpty`.
- `doctorTitle`: the health of vfox and of the installed SDKs, from `vfox doctor --json`
  (see [the CLI contract](vfox-cli-contract.md#health-aliases-and-linked-versions)).
  It is there while vfox is available, asks by itself the first time, and asks again
  when a task finishes. With nothing to report it is one item, `doctorNoProblems`,
  with `doctorCheck`, which asks again. Otherwise there is one item for each of the
  first five findings: its message, `<plugin>@<version> · <severity>` and, for
  `sdk.no-receipt`, `doctorAdopt` (`vfox doctor --adopt <plugin>@<version>`), for
  `sdk.drift`, `versionReinstall`; `doctorProblems` counts them all. What vfox finds
  about the process that asked, that its shell is not activated, is not among them
  (the card below shows how to activate a terminal). A failure to ask shows its
  message with `doctorCheck`.
- A card that says how a terminal comes to use these versions (`homeShellTitle`):
  a dropdown of the shells of the machine, the one that `SHELL` names first, or the
  usual one of the system (PowerShell on Windows, Zsh on macOS, Bash elsewhere;
  see [the CLI contract](vfox-cli-contract.md#activating-a-shell)), and for the
  chosen shell an instruction, the command with a copy button and, for Clink, a
  button that opens its script (`homeShellGetScript`). The instruction is
  `homeShellBody` for a line of a startup file (`~/.bashrc`, `~/.zshrc`,
  `~/.config/fish/config.fish`, `$PROFILE`), `homeShellOnce` for Nushell, whose
  command is run once, and `homeShellScript` for Clink. The choice is not
  remembered.

The page has no counts of what its lists hold and no shortcut to a page that the
navigation reaches (Plugins; the vfox documentation is in the settings).

## SDKs

A list-detail layout. When the page is at least 840 dp wide, the list (360 dp)
and the detail pane are side by side. The detail pane is never empty: until a
plugin is selected it shows the first plugin that has a version in use, or else the
first plugin (the page shows it without making it the selection), and it fades
through when the selection changes. With no plugin at all the list, which then has
`sdksEmptyTitle` and its action, takes the whole page. On narrower pages the list
and the detail replace each other along the horizontal shared axis, and the detail
has a back button.

### List

- Header with a refresh button and a search bar (`sdksSearchHint`) that filters
  by name.
- A segmented list of the installed plugins: avatar, name, current version
  (`sdksCurrent` or `sdksNoCurrent`) and number of installed versions
  (`sdksVersionCount`). The selected plugin is highlighted.
- Empty: `sdksEmptyTitle` / `sdksEmptyBody` with `navAddPlugin`.

### Detail

- Header: a 72 dp avatar, the name, a tag with the current version and one with
  the plugin version (`sdkPluginVersion`), and the plugin description. Actions:
  update the plugin (`sdkUpdatePlugin`; replaced by a loading indicator while it
  runs) and a menu with `actionOpenHomepage` and `sdkRemovePlugin`. On narrow
  panes the actions move to a row above the avatar.
- A connected button group switches between `sdkTabInstalled` (with the count)
  and `sdkTabAvailable`, and, for a plugin that has a
  [profile](architecture.md#plugin-profiles), `sdkTabOptions` (icon `options`).
  The tabs fade through.

**Installed** lists the installed versions, newest first:

- The current version has a filled `current` icon, an emphasized label and the
  tag `versionInUse`; its menu offers `versionUseInProject`, `versionUnuse`,
  `versionCopyPath`, `versionReinstall` (`vfox install --reinstall`), `aliasSet`
  (the [alias dialog](#choices-and-names)) and `versionUninstall`.
- Other versions have a split button: `versionUse` makes the version the global
  default; its menu offers `versionUseGlobally` and `versionUseInProject`.
  Their overflow menu offers `versionCopyPath`, `versionReinstall`, `aliasSet` and
  `versionUninstall`.
- A version that has an alias (`vfox alias`) shows `@<alias>` at the end of its second
  line. Linked versions keep the name that the user gave them.
- While a task runs for a version, its actions are replaced by a loading
  indicator.
- Empty: `sdkInstalledEmpty` with `sdkBrowseVersions`.
- For a plugin with a profile, a version name is taken apart under its
  title: the distribution (`Temurin`, `OpenJDK`) and the flavours (`JavaFX`,
  `x64`, `OpenHarmony`) as a second line.

**Available** runs `vfox search` the first time it is opened. The first row holds:

- the **search button** `availableChoiceButton` ("Distribution: Temurin"; the
  label of the choice is the profile's, the icon `registry`), which opens the
  [choices dialog](#choices-and-names) of what the search lists, when the
  profile has such a search (`java`); the choice becomes the argument of
  `vfox search` and is kept in the settings (`searches`) per plugin, so that it
  survives a restart and is the same in the web interface, and the list of the earlier
  choice goes away at once;
- at its end, the icon button `availableMoreWays` with the menu of the other ways
  to add an SDK, which every plugin has:
  - `availableInstallByName` opens the [install by name dialog](#choices-and-names);
  - `linkDetect` queues `vfox link --detect --yes <plugin>`: the plugin finds the
    SDKs that are already on the machine and vfox links them, instead of
    downloading them again. A plugin that cannot detect says so in the task;
  - `linkManual` opens the [link dialog](#choices-and-names), for an SDK in a
    directory that the user names.

The row stays while the list loads or fails, so that another choice can be made, and
an SDK that the list does not have can be added. Then:

- A filter field (`availableSearchHint`) matching versions and notes, a
  connected button group `availableFilterAll` / `availableFilterTagged`
  (versions with a note such as LTS) / `availableFilterInstalled`, and the
  number of listed versions (`availableCount`).
- **Filter chips**, one row per facet of the profile that at least one listed
  version has a value of, with the label of the facet in front (`Support`:
  LTS, JavaFX; `Channel`: stable, beta, dev, OpenHarmony; `Architecture`: x64,
  arm64), and `availableClearFilters` while a chip is on. A chip is a filter
  chip of Material 3 (see [the design system](design-system.md#components)):
  chips of one facet are alternatives, facets narrow each other and so do the
  filter field and the button group.
- A lazily built segmented list. Each version shows its note as a tag and the
  bundled tools (`npm 11.19.1`); for a search that lists several
  distributions, also a tag with the distribution. The trailing action is a split
  button: `versionInstall`, and in its menu `versionInstallUse` (`vfox install
  --use`: it makes the version the global default once it is installed); for an
  installed version, a disabled `versionInstalled`; while installing, a wavy
  progress indicator with `taskDownloading` and a cancel button.

**Options** is built from the profile of the plugin, in segmented lists. It holds
settings and what no control says; what a button or a dialog already says is not
said again (a name is explained in the install by name dialog, the shorthands
are its chips):

- `optionsVariables`: one item per environment variable the plugin reads (the
  GitHub mirror of Java and Flutter, the download mirror of Node.js, the
  storage mirror of Flutter, the C library of Node.js, tagged
  `optionsVariableAdvanced`), with the variable name, its value (with the name
  of the mirror, if it is one that was checked) or `optionsVariableUnset` (Default), and
  `optionsVariableChange`, which opens the [variable dialog](#choices-and-names);
- `optionsLegacy`: the switch `configLegacyEnable` of the vfox configuration, with
  the version files that the plugin lets vfox read as tags under its title
  (`.sdkmanrc`; `.node-version` and `.nvmrc`), and, where the strategy matters
  (Node.js), `configLegacyStrategy`. A plugin that reads no version files has no such
  section, and one that is not affected by the strategy has no choice of it;
- `aliasTitle`: the aliases of the plugin (`vfox alias list <plugin>`), loaded when
  the tab opens, each with its target and `aliasUnset`, and `aliasSet`, which opens
  the [alias dialog](#choices-and-names), with a refresh button that asks again; a plugin
  without aliases has the button and nothing else;
- `optionsTips` (Notes), at most a few lines of the profile that no control says;
- `optionsDocs`: links that open the documentation of the plugin in the
  language of the application.

### Confirmations and dialogs

- Removing a plugin: `sdkRemoveTitle` / `sdkRemoveBody` (with the number of
  versions that will be deleted), destructive confirmation `sdkRemoveConfirm`.
- Uninstalling: `versionUninstallTitle` with `versionUninstallBody`, or
  `versionUninstallCurrentBody` for the current version.
- Using in a project: `useProjectTitle` / `useProjectBody`, a directory field
  with a browse button; the directory must exist (`useProjectInvalid`).

Dialogs scale in on a spring.

#### Choices and names

- **Choices** (`searchChoose…`): a dialog titled with the label of the choice
  (`Distribution`), a filter field (`searchChooseHint`, matching the short name,
  the long name, the name and the maker, in both languages) and a list of the
  choices in the order of the profile, the popular ones first; each shows its
  name, its maker and its short name, a check mark where it is the current one
  and `searchChoiceDefault` where it is the one vfox lists without an argument.
  Choosing one closes the dialog and lists it.
- **Install by name** (`installName…`): a field `installNameField`, which takes
  what follows `<plugin>@` in `vfox install`, with the guide of the profile of the
  plugin under it (how a name is written), and, for
  a plugin with a profile, `installNameExamples`, one filter chip for each
  shorthand that fills the field, and the meaning of the shorthand that is
  typed. An empty name (`installNameRequired`) or one that has a space or a dash in
  front (`installNameInvalid`) is refused. A switch `installUseAfter`, off at first,
  adds `--use`. The confirmation queues `vfox install --yes [--use] <plugin>@<name>`
  and shows Tasks.
- **Link** (`linkPath…`): the name that vfox lists the version under
  (`linkVersionField`, with the rules of a version name) and the directory that holds
  the SDK (`linkPathField`, with a browse button; it is required), and a switch
  `linkForce`, off at first, that points an already linked version somewhere else
  (`--force`). The confirmation queues `vfox link [--force] <plugin>@<version> <path>`
  and shows Tasks.
- **Alias** (`aliasSet`): the alias (`aliasField`, with `aliasHint` in the empty field) and the version it
  names (`linkVersionField`, which starts with the version of the row that opened
  the dialog, or else the current one). Both follow the rules of a version name.
  The confirmation queues `vfox alias set <plugin> <alias> <version>`; `aliasUnset`
  queues `vfox alias unset <plugin> <alias>`.
- **Variable** (`variable…`): the help of the variable and its name; for an
  address, a monospaced field `variableAddress` (an address needs `http://` or
  `https://`, `variableInvalidAddress`); `variablePresets`, a chip for every
  value that was checked, and a chip `variableDefaultOption` that takes the
  variable away. The chips of a choice (`musl`, `glibc`) are its only values
  (`variableInvalidChoice`). The value goes to the extra environment of the
  settings, which the settings page shows too.

## Plugins

- Header: `pluginsTitle`; actions: update all plugins,
  refresh, and `pluginsAddCustom`.
- A search bar (`pluginsSearchHint`, matching name and homepage) and a
  connected button group `pluginsFilterAll` / `pluginsFilterOfficial` /
  `pluginsFilterCommunity`.
- A responsive grid of cards: avatar, name, a `verified` icon for official
  plugins (tooltip `pluginsOfficial`), the repository path, a button that opens
  the homepage, and `pluginsAdd` — or `pluginsAdded` for installed plugins,
  which opens the plugin on the SDKs page.
- `pluginsAddCustom` asks for a source (URL or path of a `.zip` archive or a
  `.lua` script, with a file chooser limited to these types, labelled
  `pluginsCustomFileType`) and an optional alias. vfox moves local `.lua` files
  away, so the application passes it a temporary copy (see
  [the CLI contract](vfox-cli-contract.md#known-quirks)).

## Tasks

A segmented list of tasks, newest first, with `tasksClearFinished` in the
header. Each task shows:

- a status: loading indicator (running), `queued`, `succeeded`, `error`, or
  `cancel` (cancelled);
- its title (`taskInstall`, `taskUse`, …) and status text; failed tasks show
  vfox's error message;
- for running installations, the wavy progress indicator;
- cancel (queued or running), retry (failed or cancelled) and a toggle that
  expands the output: the command line and the output in a monospaced, selectable
  block with a copy button. Only the newest 500 lines of a finished task are
  kept; older lines are counted, not shown (`taskDroppedLines`).

When a task finishes while another page is shown, a snack bar reports it
(`taskSucceeded`). Failures (`taskFailed` plus vfox's message) stay until
dismissed and offer `taskView`, which opens Tasks.

## Settings

Grouped in segmented lists:

- **Appearance**: theme (`themeSystem` / `themeLight` / `themeDark` as a
  connected button group), the animation (`settingsMotion`: `motionSystem` /
  `motionFull` / `motionReduced` as a connected button group; see
  [motion](design-system.md#motion)), `settingsSystemColor` (switch), eight colour swatches
  (choosing one turns the desktop colour off; the selected swatch morphs into a
  flower shape), and the language (`languageSystem` / `English` / `简体中文`;
  language names are always written in their own language).
- **Other devices**: a section whose first switch lets the Docker gateway, phones and other
  computers connect (`settingsRemote`, a switch that asks for a confirmation first, since it opens
  the computer to the network; its subtitle says why it cannot, `webRemoteFailed`, when the port is
  taken). The application shows no web interface on its own computer, so there is no switch for one.
  The lightweight mode (`settingsLightweight`, a switch that keeps the application running without
  its window for the other devices; needs other devices, which its subtitle says when they are off,
  `settingsNeedsOtherDevices`). While other devices are allowed, the list adds the port (`settingsWebPort`, a text field saved on Enter or when leaving the field,
  and refused when it is not a port from 1024 to 65535), the public address (`settingsWebPublicAddress`, a
  text field for the address at which a reverse proxy, a tunnel or the gateway publishes the application,
  which the pairing links and the code then carry), the public tunnel (`settingsPublish`, a switch that
  asks for a confirmation first: a Cloudflare quick tunnel), *Pair a phone* (`webPair`: the addresses of
  the computer, and a dialog with the QR code of a pairing link, the link, the choice among the
  addresses when there are several, and `webPairNew`) and the certificate (`webCertificate`: made by
  the application, or the files of the user, with the SHA-256 fingerprint in lines that can be
  selected). While the lightweight mode is on and other devices are allowed, *Quit vfox+* (`webQuit`,
  after a confirmation) ends the application, which closing the window no longer does. With no
  window, the application has an icon in the tray (`trayToolTip` says what it is) whose menu has
  *Open vfox+* (`trayOpen`) and *Quit vfox+* (`webQuit`, without a confirmation: there is no window
  to ask in). The icon is the fox of the application
  icon without its plate; see [the architecture](architecture.md#the-tray).
- **vfox**: the executable with its origin (`settingsExecutableDetected`,
  `settingsExecutableCustom` or `settingsExecutableMissing`) and version, a
  browse button and, for a custom executable, `settingsExecutableDetect`; the
  extra environment variables as `KEY=VALUE` lines with validation
  (`settingsEnvironmentInvalid`).
- **vfox configuration**: an editor for each key listed in
  [the CLI contract](vfox-cli-contract.md#configuration) — switches for booleans
  (among them `diagnostics.onUse`), a menu for `legacyVersionFile.strategy` and for
  `path.mode` (`pathModeLegacy`, `pathModeBaseline`, `pathModeAggressive`), text fields
  (saved on Enter or when leaving the field; clearing a field resets the key) for the
  rest, `path.preserve` and `diagnostics.ignore` among them, which hold lists in JSON,
  and `configOther` for keys the GUI does not know. `aliases` is not edited here: the
  aliases of a plugin are in its options.
- **About**: application name, version and license, links to the source code
  and the vfox documentation, and the open-source licenses of the dependencies,
  including the bundled Material Symbols icons (Apache-2.0, Google).

## Accessibility

- Every icon-only button has a tooltip, which also serves as its accessible
  name.
- Selected items, toggles and button groups expose their selected state.
- Headers are marked as headers.
- All motion stops or becomes instantaneous when the platform asks for reduced
  motion, unless the person chose full animation on the desktop (`motionFull`), and
  on the desktop when they chose reduced animation (see
  [the design system](design-system.md#motion)).
- Text follows the platform text scale; layouts wrap rather than clip.

## Phone app

The phone app ([`mygo/phone`](../../../mygo/phone)) shows the five tabs of the window with the same
names and icons: Home, SDKs, Plugins, Tasks and Settings. On a phone they are a navigation bar at the
bottom of the screen, as the compact layout above describes. The tasks tab shows the same badge of
queued and running tasks.

The pages above the tabs are the detail of an SDK, a task with its output, the full log of a task, and
the pairing page. A paired phone shows nothing above the tabs, except while it checks the saved computer
(*Connecting to the computer…*, with a progress indicator). When the computer does not answer, a bar above
the tabs says *The computer did not answer. Check the network, and that vfox+ runs on the computer.* and
offers *Retry*; the tabs stay on the screen.

The pairing page is shown when the phone has no computer, when it must pair again (*This phone must pair
again*), when the computer's certificate changed (*The computer's certificate changed*), and as a page above
the tabs for *Pair a new link*. It has a short explanation (*How to pair*) and a field for the pairing link,
which three buttons fill: *Paste*, *Photo* (the camera reads the code) and *Picture* (an image of the code).
*Connect* checks the computer and opens the dialog *Trust this computer?*. The dialog shows
*Checking the computer…* while the check runs, then the fingerprint of the certificate, with *Cancel* and
*Trust and pair*. A line in red warns when the fingerprint is not the one paired before. The phone pairs
only when the person chooses *Trust and pair*.

Settings holds the card *This phone* (the computer it is paired with, *Pair a new link* and *Disconnect*),
the connections of other devices (*Connections*: *Other devices*, with the port and the lightweight mode;
*Pair a phone*; and *Quit vfox+*, which quits the application on the computer), *Application* (*vfox
executable*, *Application language*, *Interface language* and *Environment variables*) and *vfox config*,
the configuration of vfox. The application language is the computer's, and the phone sets it there. The
interface language is a preference of the phone's own, which changes only its screens: it follows the
system's languages, or is one of English and Simplified Chinese.

Its components, colours, shapes and springs are the desktop's; see [the design system](design-system.md#phone-app).
