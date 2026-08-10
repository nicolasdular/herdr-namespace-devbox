# Namespace Devbox for Herdr

Open a persistent [Namespace Devbox](https://namespace.so/) in a new [Herdr](https://herdr.dev/) tab. Each Git worktree gets a stable default Devbox that the plugin creates on first use and reconnects to later.

## Requirements

- macOS or Linux
- [Herdr](https://herdr.dev/docs/install/) 0.8.0 or newer
- A Namespace account and the Namespace `devbox` CLI

Install the Namespace CLI if it is not already available:

```sh
curl -fsSL https://get.namespace.so/devbox/install.sh | sh
devbox --version
```

## Install

Install the plugin directly from GitHub:

```sh
herdr plugin install nicolasdular/herdr-namespace-devbox
```

Herdr shows the plugin and its install command for review, then downloads the prebuilt binary for your operating system and architecture. Confirm that both actions are available:

```sh
herdr plugin action list --plugin namespace.devbox
```

## Use it

From a Herdr workspace, open its default Devbox:

```sh
herdr plugin action invoke namespace.devbox.start-devbox
```

The action opens a focused `Devbox · <workspace>` tab without changing the pane where it was invoked. Its first invocation starts the Namespace login flow when necessary, then creates a persistent Devbox for the current Git worktree. Later invocations reconnect to the existing Devbox in another new tab. Invoking the action from a nested directory still uses the Git worktree root.

To create a separate Devbox without changing the workspace default:

```sh
herdr plugin action invoke namespace.devbox.new-devbox
```

The additional Devbox receives a unique name and opens in its own `Devbox · <workspace>` tab.

To list and stop any of your Namespace Devboxes in an interactive popup:

```sh
herdr plugin action invoke namespace.devbox.manage-devboxes
```

Use the arrow keys or `j`/`k` to select a Devbox, `enter` to focus its open tab anywhere in Herdr (or open a new tab), `c` to create a new Devbox in its own tab, `s` to stop it, `d` to permanently delete it, `r` to refresh the list, and `q` or `esc` to close the popup. Stop and delete operations require confirmation. Opening or creating a new tab requires an active Herdr workspace.

To stop the workspace's default Devbox:

```sh
herdr plugin action invoke namespace.devbox.stop-devbox
```

## Add keybindings

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

With Herdr's default prefix, press `ctrl+b d` to manage Devboxes, `ctrl+b alt+d` to create a separate one, or `ctrl+b shift+s` to stop the workspace Devbox. These avoid Herdr's built-in bindings for closing a workspace (`prefix+shift+d`) and pane (`prefix+x`).

## Configuration

The plugin works without configuration by using a private, medium-sized `builtin:agents` Devbox with a one-hour idle timeout and a `herdr` Bash session. It clones the workspace's Git remote when one is available.

To customize a workspace, create `devbox.yaml` at the Git worktree root:

```yaml
name: project-devbox
image: builtin:agents
size: m
access_mode: private
auto_stop_idle_timeout: 1h
sessions:
  - name: herdr
    command: bash
```

The YAML `name`, when present, becomes the workspace's default Devbox name; otherwise the plugin generates a stable name. The `new-devbox` action uses the same specification with a unique name. If the YAML does not declare a session, the plugin adds the default `herdr` Bash session. Both YAML-backed and default specifications are normalized to JSON and streamed to the Namespace CLI.

## Update or uninstall

Reinstall the GitHub plugin to update it. Workspace `devbox.yaml` files remain in their repositories:

```sh
herdr plugin install nicolasdular/herdr-namespace-devbox
```

Uninstall the plugin with:

```sh
herdr plugin uninstall namespace.devbox
```

Uninstalling the plugin does not delete Devboxes from your Namespace account.

## Troubleshooting

Check the plugin registration, available actions, and recent logs:

```sh
herdr plugin list --plugin namespace.devbox
herdr plugin action list --plugin namespace.devbox
herdr plugin log list --plugin namespace.devbox
```

Check the Namespace CLI and authentication separately:

```sh
devbox --version
devbox auth check-login
```

If login is required, invoke either plugin action again and complete the displayed login flow.

## Development

Go and mise are only required when working on the plugin itself:

```sh
git clone https://github.com/nicolasdular/herdr-namespace-devbox.git
cd herdr-namespace-devbox
mise install
mise run install
mise run check
herdr plugin link "$PWD"
```

`herdr plugin link` uses the locally built `bin/herdr-namespace`; unlike `plugin install`, it does not run the manifest's build command. Remove the development link with:

```sh
herdr plugin unlink namespace.devbox
```

### Publishing a release

Set `version` in `herdr-plugin.toml`, commit the change, and push a matching tag such as `v0.2.0`. The release workflow uses GoReleaser to cross-compile macOS and Linux archives for Intel and ARM, generates `checksums.txt`, and publishes everything to the GitHub release. The tag and manifest version must match exactly.

```sh
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0
```

After the workflow succeeds, test a clean managed installation with `herdr plugin install nicolasdular/herdr-namespace-devbox`.
