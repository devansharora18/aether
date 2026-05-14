# Aether

Aether is a modern, interactive Terminal User Interface (TUI) and command-line wrapper for the Debian/Ubuntu `apt` package manager, written in Go. It brings a `pacman`-like syntax and an intuitive graphical console to `apt`.

## Features

- **Interactive TUI**: Browse, search, install, and remove packages using a keyboard-driven visual interface.
- **Smart Privilege Escalation**: Run Aether as a normal user. It will only prompt for `sudo` when an operation actually requires root privileges.
- **Familiar CLI Flags**: Uses intuitive flags originally popularized by Arch Linux's `pacman`.
- **Live Progress Support**: High-quality visual progress bars while fetching and installing packages.
- **Source Management**: View and modify your APT sources list (`.list` and `.sources`) directly from the TUI.
- **Safe Execution**: Gracefully handles APT front-end locks and dpkg dependencies.

## Installation

Ensure you have a recent version of Go installed.

### Quick Install (One-liner)

```bash
git clone https://github.com/devansharora18/aether.git /tmp/aether && cd /tmp/aether && ./install.sh && cd - && rm -rf /tmp/aether
```

### Curl Install (No Git)

```bash
curl -fsSL https://raw.githubusercontent.com/devansharora18/aether/main/install-aether.sh | bash
```

### Manual Install

```bash
git clone https://github.com/devansharora18/aether.git
cd aether
./install.sh
```

You can customize the installation using script arguments (e.g., `./install.sh --prefix ~/.local --no-sudo`). Run `./install.sh --help` for all options.

Or install directly via `go install`:

```bash
go install github.com/devansharora18/aether@latest
```

## Usage

### Interactive Mode

Simply run the command with no arguments to launch the TUI:

```bash
aether
```

### Command Line Interface

Aether supports standard package management operations via CLI flags:

| Flag | Action |
| --- | --- |
| `-S <pkg...>` | Install packages |
| `-R <pkg...>` | Remove packages |
| `-Rn <pkg...>` | Purge packages (remove + configuration files) |
| `-Rc` | Remove unused dependencies (autoremove) |
| `-Sy` | Update package database (`apt update`) |
| `-Syu` | Update and upgrade all packages (`apt upgrade`) |
| `-Ss <query>` | Search packages |
| `-Qi <pkg...>` | Show detailed package info |
| `-Ql` | List installed packages |
| `-Qu` | List upgradable packages |
| `-Sc` | Clean package cache |
| `-v` | Verbose mode (streams raw apt output) |

## Examples

Search for a package:
```bash
aether -Ss htop
```

Install a package (will prompt for sudo automatically):
```bash
aether -S htop btop
```

Full system upgrade:
```bash
aether -Syu
```

## License

This project is licensed under the [GNU General Public License v3.0](LICENSE).
