# Namespace Devbox for Herdr

Open or reconnect to a persistent, workspace-associated Namespace Devbox terminal in the focused Herdr pane.

## Requirements

- mise
- Herdr 0.8.0 or newer
- Go 1.26 or newer
- Namespace Devbox CLI, authenticated with `devbox login`

## Install and link

```sh
mise install
mise run install # builds bin/herdr-namespace
mise run check
herdr plugin link /absolute/path/to/herdr-namespace
herdr plugin action list --plugin namespace.devbox
```

Invoke the action directly:

```sh
herdr plugin action invoke start-devbox --plugin namespace.devbox
```

`start-devbox` reconnects to the default Devbox for the current Git worktree. If it does not exist yet, the action creates it. Invoking the action from a nested directory still uses the Git worktree root.

Create a separate Devbox explicitly:

```sh
herdr plugin action invoke new-devbox --plugin namespace.devbox
```

The additional Devbox gets a unique name and does not change the workspace's default Devbox.

Or add a keybinding to Herdr's config:

```toml
[[keys.command]]
key = "prefix+d"
type = "plugin_action"
command = "namespace.devbox.start-devbox"
description = "open this workspace in a Namespace Devbox"
```

An optional separate keybinding can always create a new Devbox:

```toml
[[keys.command]]
key = "prefix+shift+d"
type = "plugin_action"
command = "namespace.devbox.new-devbox"
description = "open a new Namespace Devbox for this workspace"
```

## Configuration

Find the managed configuration directory:

```sh
herdr plugin config-dir namespace.devbox
```

Create `config.json` there if the defaults need changing:

```json
{
  "image": "builtin:agents",
  "size": "m",
  "accessMode": "private",
  "autoStopIdleTimeout": "1h",
  "sessionName": "herdr",
  "shell": "bash",
  "setupGithub": false,
  "volumeSizeGb": 100
}
```

Supported sizes are `s`, `m`, `l`, and `xl`. Unknown configuration keys are rejected.

If the workspace contains a `devbox.yaml`, it is passed directly to Namespace for creation instead of generating a spec from the plugin configuration. It must define `name`, which becomes the workspace's default Devbox name.
