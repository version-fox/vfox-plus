# Use vfox-plus

> [简体中文](../../zh-CN/user/using.md)

The application has five destinations: Home, SDKs, Plugins, Tasks and
Settings. On wide windows they sit in the side rail; on narrow windows they
move to the bottom bar.

## Home

- The vfox version and where its executable was found, with a refresh button.
- Three statistic tiles: plugins, installed versions and versions in use.
  Tapping one opens SDKs.
- The plugins that have a version in use; tapping one opens it on the SDKs
  page.
- How to use the versions in a terminal, with the line to add to `~/.bashrc`
  and a copy button.
- Shortcuts to the plugin registry and to the vfox documentation.

## SDKs

A list of your installed plugins with their current version. Pick one to see
its detail: description, homepage, and two tabs.

**Installed** lists the installed versions, newest first. The current version
is marked; the others offer to become the global default. Each version can
also be pinned to a project (in its own directory), have its install path
copied, or be uninstalled. Removing the plugin itself deletes all its
versions, after a confirmation.

**Available** searches the versions you can install: filter by text, show
only versions with a note (such as LTS) or only installed ones. Installing
shows live download progress and can be cancelled.

## Plugins

- Search the registry and filter official or community plugins.
- Each card tells whether the plugin is already installed and opens its
  homepage.
- Add a plugin from the registry, from a URL, or from a local `.zip` archive
  or `.lua` script file — plus update all plugins at once.
- Installed plugins offer update and remove.

## Tasks

Every install, uninstall, switch, add, remove and update becomes a task,
newest first. Each task shows its state (queued, running, succeeded, failed,
cancelled), live progress for running installations, and the output log with
a count of dropped earlier lines; a failed task keeps its full log on disk
and shows where.
Failed and cancelled tasks can be retried; finished ones can be cleared. When
a task finishes while you look at another page, a message reports it.

## Settings

- **Appearance**: light, dark or follow the system; use the desktop accent
  colour or pick one of the swatches; the language (English or 简体中文).
- **vfox**: which executable is used (detected automatically, or pick your
  own) and extra environment variables such as download mirrors.
- **vfox configuration**: proxy, registry, storage directory, legacy version
  files, caches and timeouts — switches, menus and text fields; clearing a
  field resets it.
- **About**: version, source code and dependency licenses.

## When vfox is missing

If no vfox executable is found, or the one found does not work, every page
explains how to install vfox and offers to locate the executable or retry.
Settings stay reachable so you can point the application at vfox yourself.
