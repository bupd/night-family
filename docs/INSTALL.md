# Install

There are two install modes: **dev** (you just want to try it) and
**daemon** (you want it running every night on a Linux box).

## Dev

```bash
go install github.com/bupd/night-family/cmd/nfd@latest
go install github.com/bupd/night-family/cmd/nf@latest

nfd --db :memory: &                  # defaults to the mock provider
nf night trigger                     # fires a full plan through the mock
nf run list                          # 12 runs, all succeeded
open http://127.0.0.1:7337           # dashboard
```

That's the whole thing. Stop nfd with `Ctrl-C` or `kill %1`.

## Daemon (Linux, systemd)

### 1. Build + install the binary

```bash
git clone https://github.com/bupd/night-family
cd night-family
task build
sudo install -m 0755 bin/nfd /usr/local/bin/nfd
sudo install -m 0755 bin/nf  /usr/local/bin/nf
```

### 2. Auth the supporting CLIs

As the **user** the daemon will run under (not root):

```bash
gh auth login                        # needed for gh pr create
claude login                         # needed if --provider=claude
```

Confirm both work:

```bash
gh auth status
claude --version
```

### 3. Drop a per-user config + optional family overrides

```bash
mkdir -p ~/.config/night-family/family
# Drop any custom family YAMLs here — the embedded defaults still seed first.
```

### 4. Install the systemd unit

```bash
sudo cp deploy/systemd/nfd.service /etc/systemd/system/nfd@.service
sudo systemctl daemon-reload
sudo systemctl enable --now nfd@$USER.service
sudo journalctl -u nfd@$USER.service -f
```

`nfd@<user>.service` is the templated form; the `%i` placeholder inside
the unit file resolves to the username. That way multiple users on the
same host each get their own daemon.

### 5. Configuration via /etc/default/nfd (optional)

The unit file reads `/etc/default/nfd` for environment overrides:

```ini
# /etc/default/nfd
NF_PROVIDER=claude
NF_REVIEWERS=coderabbitai,cubic-dev-ai,alice
```

Any flag the daemon supports can be exported through the env file;
edit the `ExecStart=` line in the unit to reference more `${FOO}`
variables as you need them.

## macOS (launchd)

TBD — the systemd unit is a straightforward port to a launchd plist;
haven't shipped it yet because the maintainer runs Linux. Patches
welcome.

## Docker

Not recommended for v1. night-family shells out to `git`, `gh`, and
`claude`, all of which expect host filesystems and auth tokens that
are painful to thread into a container. If there's demand we can
publish a container image with the binaries and have users mount the
auth dirs — file an issue.

## Uninstall

```bash
sudo systemctl disable --now nfd@$USER.service
sudo rm /etc/systemd/system/nfd@.service /usr/local/bin/nfd /usr/local/bin/nf
rm -rf ~/.local/share/night-family   # Deletes the SQLite DB. Irreversible.
rm -rf ~/.config/night-family        # Deletes your config.
```
