# Getting Started with bishop

bishop is an actively maintained fork of gsh focused on a faster development cycle while remaining compatible and regularly contributing improvements upstream.

Upstream project: https://github.com/atinylittleshell/gsh
Fork repository: https://github.com/robottwo/bishop

If you're new, start here to install, build, and run bishop quickly.

## Requirements

- macOS or Linux
- Go 1.21+ installed and in your PATH

## Install and Build

Packaging for bishop is in progress. For now, build from source:

```bash
git clone https://github.com/robottwo/bishop.git
cd bishop
make build
# The binary will be in ./bin/bish
```

To make the binary available on your PATH:

```bash
sudo install -m 0755 bin/bish /usr/local/bin/bish
```

### Upgrading

Bishop includes self-update support. When a new version is available, it can automatically detect and offer to update.

## Launching bishop

### Manual

Start bishop from an existing shell:

```bash
bish
```

### Automatically from your shell

Add bishop to your shell configuration so it starts automatically:

```bash
# bash
echo "bish" | tee -a ~/.bashrc
```

```bash
# zsh
echo "bish" | tee -a ~/.zshrc
# If you have an alias named bish, use the full path
echo "/usr/local/bin/bish" | tee -a ~/.zshrc
```

### As your login shell

Not recommended yet, but if you know what you are doing:

```bash
which bish
echo "/path/to/bish" | sudo tee -a /etc/shells
chsh -s "/path/to/bish"
```

## Default Key Bindings

Familiar, ergonomic defaults for navigation and editing:

- Character Forward: Right Arrow, Ctrl+F
- Character Backward: Left Arrow, Ctrl+B
- Word Forward: Alt+Right Arrow, Ctrl+Right Arrow, Alt+F
- Word Backward: Alt+Left Arrow, Ctrl+Left Arrow, Alt+B
- Delete Word Backward: Alt+Backspace, Ctrl+W
- Delete Word Forward: Alt+Delete, Alt+D
- Delete After Cursor: Ctrl+K
- Delete Before Cursor: Ctrl+U
- Delete Character Backward: Backspace, Ctrl+H
- Delete Character Forward: Delete, Ctrl+D
- Line Start: Home, Ctrl+A
- Line End: End, Ctrl+E
- Paste: Ctrl+V
- Yank (Paste Last Cut Text): Ctrl+Y
- Yank-Pop (Cycle Previous Cuts): Alt+Y
- History Previous: Up Arrow, Ctrl+P
- History Next: Down Arrow, Ctrl+N
- History Search: Ctrl+R
- Tab Completion: Tab, Shift+Tab

Bash- and zsh-style kill ring shortcuts are supported: Ctrl+K (cut to end of line), Ctrl+U (cut to start of line), and Ctrl+W (cut the previous word) store the removed text so it can be yanked back with Ctrl+Y. Sequential kills in the same direction append to the latest entry, and Alt+Y yank-pop cycles through earlier kills.

### History Search

Press Ctrl+R to open an interactive history search with fuzzy matching. While in history search:

- Type to filter commands
- Up/Down arrows to navigate results
- Ctrl+F to toggle between "All" and "Directory" filter modes
- Enter to select a command
- Esc to cancel

## Next Steps

- Configure bishop: see ./CONFIGURATION.md
- Explore features and workflows: see ./FEATURES.md
- Learn about the Agent: see ./AGENT.md
- Use specialized Subagents: see ./SUBAGENTS.md

If you run into issues, open an issue at https://github.com/robottwo/bishop/issues.