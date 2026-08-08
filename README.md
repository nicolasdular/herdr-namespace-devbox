# Namespace Devbox for Herdr

Open a persistent [Namespace Devbox](https://namespace.so/) in the focused [Herdr](https://herdr.dev/) pane. Each Git worktree gets a stable default Devbox that the plugin creates on first use and reconnects to later.

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

The first invocation starts the Namespace login flow when necessary. It then creates a persistent Devbox for the current Git worktree, or reconnects to the existing one. Invoking the action from a nested directory still uses the Git worktree root.

To create a separate Devbox without changing the workspace default:

```sh
herdr plugin action invoke namespace.devbox.new-devbox
```

The additional Devbox receives a unique name.

## Add keybindings

Add these commands to Herdr's config at `~/.config/herdr/config.toml`:

```toml
[[keys.command]]
key = "prefix+d"
type = "plugin_action"
command = "namespace.devbox.start-devbox"
description = "open this workspace in a Namespace Devbox"

[[keys.command]]
key = "prefix+shift+d"
type = "plugin_action"
command = "namespace.devbox.new-devbox"
description = "open a new Namespace Devbox for this workspace"
```

Reload the config after saving it:

```sh
herdr server reload-config
```

With Herdr's default prefix, press `ctrl+b d` to open the workspace Devbox or `ctrl+b shift+d` to create a separate one.

## Configuration

The defaults work without a configuration file. To customize them, first find the plugin's managed configuration directory:

```sh
herdr plugin config-dir namespace.devbox
```

Create a `config.json` file in that directory:

```json
{
  "image": "builtin:agents",
  "size": "m",
  "accessMode": "private",
  "autoStopIdleTimeout": "1h",
  "sessionName": "herdr",
  "shell": "bash",
  "setupGithub": false
}
```

Omitted fields use the defaults shown above. Supported sizes are `s`, `m`, `l`, and `xl`; `accessMode` can be `private` or `shared`. You can also set `volumeSizeGb` to a positive integer or `site` to a Namespace site name. Unknown configuration keys are rejected.

### Workspace `devbox.yaml`

When a Git worktree contains `devbox.yaml` at its root, the plugin passes that file directly to Namespace instead of using `config.json`. The file must define `name`, which becomes the workspace's default Devbox name. The `new-devbox` action uses the same specification with a unique name suffix.

## Update or uninstall

Reinstall the GitHub plugin to update it. Your configuration is stored separately and remains in place:

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
