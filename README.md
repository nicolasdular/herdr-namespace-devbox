# Namespace Devbox for Herdr

Open a persistent [Namespace Devbox](https://namespace.so/) in a new [Herdr](https://herdr.dev/) tab. Each Git worktree is treated as a workspace, even when an action runs from a nested directory.

## Demo


https://github.com/user-attachments/assets/d8ed8f1e-131d-4b6e-90de-2a59fc9f3fff


## Requirements

- macOS or Linux
- [Herdr](https://herdr.dev/docs/install/) 0.8.0 or newer
- A Namespace account and the Namespace `devbox` CLI

Install the Namespace CLI:

```sh
curl -fsSL https://get.namespace.so/devbox/install.sh | sh
devbox --version
```

## Install

Install the plugin directly from GitHub:

```sh
herdr plugin install nicolasdular/herdr-namespace-devbox
```

Herdr asks you to review the plugin and install command, then downloads the binary for your platform. Verify it with:

```sh
herdr plugin action list --plugin namespace.devbox
```

## Use the plugin

### Open the workspace's default Devbox

From a Herdr workspace, run:

```sh
herdr plugin action invoke namespace.devbox.start-devbox
```

This opens a focused `Devbox · <workspace>` tab without changing the current pane. The Devbox is created when needed, and each invocation opens a new tab. Namespace login starts first when required.

### Create an additional Devbox

```sh
herdr plugin action invoke namespace.devbox.new-devbox
```

This creates a separate Devbox without changing the workspace default.

### Manage all Devboxes

```sh
herdr plugin action invoke namespace.devbox.manage-devboxes
```

The popup supports the following keys:

| Key | Action |
| --- | --- |
| Arrow keys or `j`/`k` | Select a Devbox. |
| `enter` | Focus the Devbox's existing tab anywhere in Herdr, or open it in a new tab if it is not already open. |
| `c` | Open the creation form for a new Devbox. The form can optionally upload tracked local changes. |
| `s` | Stop the selected Devbox after confirmation. |
| `d` | Permanently delete the selected Devbox after confirmation. |
| `r` | Refresh the list. |
| `q` or `esc` | Close the popup. |

A workspace is required only to create or open a tab. New Devboxes default to a clean repository checkout. In the creation form, enable **Local changes** to apply staged and unstaged modifications to tracked files before connecting; untracked and ignored files are never uploaded.

### Stop the workspace's default Devbox

```sh
herdr plugin action invoke namespace.devbox.stop-devbox
```

To stop a Devbox created with `new-devbox`, use `manage-devboxes` instead.

## Keybindings

Add these commands to Herdr's config at `~/.config/herdr/config.toml`:

```toml
[[keys.command]]
key = "prefix+d"
type = "plugin_action"
command = "namespace.devbox.manage-devboxes"
description = "manage Namespace Devboxes"

[[keys.command]]
key = "prefix+alt+d"
type = "plugin_action"
command = "namespace.devbox.new-devbox"
description = "open a new Namespace Devbox for this workspace"

[[keys.command]]
key = "prefix+shift+s"
type = "plugin_action"
command = "namespace.devbox.stop-devbox"
description = "stop this workspace's Namespace Devbox"
```

Reload the config after saving it:

```sh
herdr server reload-config
```

With the default `ctrl+b` prefix, these become `ctrl+b d`, `ctrl+b alt+d`, and `ctrl+b shift+s`. They avoid the built-in workspace (`prefix+shift+d`) and pane (`prefix+x`) closing shortcuts.

## Configuration

The default is a private, medium-sized (`m`) `builtin:agents` Devbox with a one-hour idle timeout and a Bash session named `herdr`. The workspace's `origin` remote is cloned when available.

To customize a workspace, create `devbox.yaml` at its root. For example:

```yaml
name: project-devbox
image: builtin:agents
size: m
access_mode: private
auto_stop_idle_timeout: 1h
env:
  - name: MISE_DISABLE_TOOLS
    value: postgres
sessions:
  - name: herdr
    command: bash
```

`devbox.yaml` replaces the defaults; it does not merge with them. Include every setting you need. If `sessions` is omitted, the plugin adds the default `herdr` Bash session.

`start-devbox` and `stop-devbox` use the YAML `name`, or a stable generated name when absent. `new-devbox` always replaces it with a unique name.

The plugin converts the YAML to JSON for the Namespace CLI. Unknown fields cause an error.

Personal creation options live in Herdr's global plugin `config.json`, keeping
`devbox.yaml` compatible with the official Namespace specification. Find the
configuration directory with:

```sh
herdr plugin config-dir namespace.devbox
```

Configure the dotfiles repository:

```json
{
  "dotfiles": "github.com/acme/dotfiles"
}
```

Dotfiles are applied globally to newly created Devboxes through the
`devbox create --dotfiles` option. Changing this setting does not reconfigure
an existing Devbox.

## Update or uninstall

Reinstall to update; `devbox.yaml` files are unchanged:

```sh
herdr plugin install nicolasdular/herdr-namespace-devbox
```

Uninstall with:

```sh
herdr plugin uninstall namespace.devbox
```

Namespace Devboxes are not deleted.

## Troubleshooting

Check the plugin registration, available actions, and recent logs:

```sh
herdr plugin list --plugin namespace.devbox
herdr plugin action list --plugin namespace.devbox
herdr plugin log list --plugin namespace.devbox
```

Check the CLI and authentication:

```sh
devbox --version
devbox auth check-login
```

## Development

Plugin development requires Go and mise:

```sh
git clone https://github.com/nicolasdular/herdr-namespace-devbox.git
cd herdr-namespace-devbox
mise install
mise run install
mise run check
herdr plugin link "$PWD"
```

`herdr plugin link` uses the locally built `bin/herdr-namespace`; it does not build it. Run `mise run install` first.

Remove the development link with:

```sh
herdr plugin unlink namespace.devbox
```

### Publishing a release

Set `version` in `herdr-plugin.toml`, commit it, and push a matching tag. Both versions must match exactly.

```sh
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0
```

GoReleaser publishes macOS and Linux archives for Intel and ARM, plus `checksums.txt`. When it finishes, test a clean installation with `herdr plugin install nicolasdular/herdr-namespace-devbox`.
