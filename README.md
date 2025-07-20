# jumphost

A terminal UI (TUI) tool to quickly select and SSH into hosts defined in your `~/.ssh/config` file.

## Features
- Parses your `~/.ssh/config` and lists all host aliases (ignoring wildcards)
- Interactive TUI for host selection (powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea))
- Secure password entry with a TUI input field (no default SSH prompt)
- Multi-screen interface: host list → password input → login progress
- Host management: delete entries directly from the SSH config
- Cross-platform: Linux, macOS, and Windows support
- Statically linked binaries with no external dependencies

## Prerequisites
- **sshpass** (required for password-based SSH; the app checks for it at startup)

### Install sshpass
- **macOS:**
  ```sh
  brew install hudochenkov/sshpass/sshpass
  ```
- **Linux (Debian/Ubuntu):**
  ```sh
  sudo apt install sshpass
  ```
- **Windows:** Install via WSL, Git Bash, or Cygwin
- For other platforms, install sshpass using your platform's package manager.

## Installation

### Option 1: Download Pre-built Binary (Recommended)

1. Go to the [Releases](https://github.com/yourusername/ssh-hosts/releases) page
2. Download the appropriate binary for your platform:
   - **Linux**: `linux-amd64/jumphost` or `linux-arm64/jumphost`
   - **macOS**: `darwin-amd64/jumphost` or `darwin-arm64/jumphost`
   - **Windows**: `windows-amd64/jumphost.exe` or `windows-arm64/jumphost.exe`

3. Make it executable (Linux/macOS):
   ```sh
   chmod +x linux-amd64/jumphost
   ```

4. Move to your PATH (optional):
   ```sh
   # Linux/macOS
   sudo mv linux-amd64/jumphost /usr/local/bin/jumphost
   
   # Windows (run as Administrator)
   move windows-amd64/jumphost.exe C:\Windows\System32\jumphost.exe
   ```

### Option 2: Build from Source

```sh
git clone https://github.com/yourusername/ssh-hosts.git
cd ssh-hosts
go build -o jumphost
```

## Usage

1. **Run the application:**
   ```sh
   ./jumphost
   ```

2. **Navigate the interface:**
   - Use arrow keys to navigate the host list
   - Press `Enter` to connect to the selected host
   - Press `Delete` or `x` to remove the selected host from SSH config
   - Enter your password in the TUI input field
   - Press `Esc` to go back to the host list
   - Press `Ctrl+C` to quit

3. **SSH Connection:**
   - The program will attempt to connect using your password
   - If successful, you'll be dropped into an SSH session
   - If the password is wrong, you'll return to the password input screen

## Configuration

The program automatically reads your `~/.ssh/config` file and lists all host aliases (excluding wildcards like `*` or `?`).

### Example `~/.ssh/config`
```
Host test-server
    Hostname 172.31.30.182
    User root

Host production-server
    Hostname 1.1.1.1
    User admin

Host staging-server
    Hostname staging.example.com
    User deploy
    Port 2222
```

## Development

### Prerequisites
- Go 1.21 or newer

### Build
```sh
go build -o jumphost
```

### Test
```sh
go test -v
```

### Run
```sh
./jumphost
```

## CI/CD

This project uses GitHub Actions for:

- **Testing**: Runs tests on every push and pull request
- **Building**: Creates binaries for multiple platforms
- **Releasing**: Automatically creates releases when you push version tags

### Creating a Release
```sh
git tag v1.0.0
git push origin v1.0.0
```

This will automatically:
- Build binaries for Linux, macOS, and Windows (AMD64 & ARM64)
- Create a GitHub release with the tag
- Upload all binaries and checksums as release assets
- Generate release notes from commits

## Security

- Passwords are entered through a secure TUI input field
- No passwords are stored or logged
- Uses `sshpass` for non-interactive SSH authentication
- Statically linked binaries reduce attack surface

## Troubleshooting

### "sshpass is not installed"
Install sshpass using your platform's package manager (see Prerequisites section).

### "No hosts found in ~/.ssh/config"
Make sure your SSH config file exists and contains valid host entries.

### Terminal display issues
The app sets `TERM=xterm-256color` to ensure compatibility across different terminal emulators.

## License

MIT License - see [LICENSE](LICENSE) file for details. 