# GoCreate 🚀

A powerful **Model Context Protocol (MCP) server** built in Go that provides comprehensive development tools for AI assistants. GoCreate enables seamless interaction between AI models and your development environment through a standardized protocol.

Built with [**gomcp**](https://github.com/localrivet/gomcp) - the complete Go implementation of the Model Context Protocol.

## ✨ Features

### 🗂️ **File System Operations**
- **Read Files**: Read file contents with optional line-based pagination
- **Write Files**: Create or completely replace file contents
- **Multiple File Reading**: Read multiple files simultaneously
- **Directory Management**: Create directories and list contents with detailed metadata
- **File Operations**: Move, rename files and directories
- **File Search**: Find files by name using case-insensitive substring matching
- **File Info**: Get detailed metadata about files and directories

### ✏️ **Code Editing**
- **Block Editing**: Surgical text replacements with diff-based error reporting
- **Precise Editing**: Line-based editing with start/end line specifications
- **Large File Support**: Handles files up to 100MB with memory-efficient processing
- **Context-Aware Replacements**: Smart replacement with near-miss detection

### 🔍 **Search Capabilities**
- **Code Search**: Powered by ripgrep for fast text/regex pattern searching
- **Advanced Filtering**: File pattern matching, case-insensitive search
- **Context Lines**: Configurable context around matches
- **Timeout Support**: Configurable search timeouts

### 💻 **Terminal & Process Management**
- **Command Execution**: Execute terminal commands with timeout support
- **Session Management**: Manage multiple terminal sessions
- **Process Control**: List running processes and terminate by PID
- **Output Reading**: Read command output from running sessions
- **Cross-Platform**: Support for Unix-like systems and Windows

### ⚙️ **Configuration Management**
- **Dynamic Config**: Get and set configuration values at runtime
- **Security Controls**: Configurable blocked commands for safety
- **JSON-based**: Human-readable configuration format

## 🛠️ Installation

### Prerequisites
- Go 1.24 or later
- Git
- [ripgrep](https://github.com/BurntSushi/ripgrep) (for code search functionality)

### Build from Source
```bash
git clone https://github.com/localrivet/gocreate.git
cd gocreate
go mod download
go build -o gocreate
```

## 🚀 Usage

### As MCP Server
GoCreate implements the [Model Context Protocol](https://modelcontextprotocol.io/) and can be used with any MCP-compatible client:

```bash
./gocreate
```

### Configuration with Claude Desktop
Add to your Claude Desktop configuration (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "gocreate": {
      "command": "/path/to/gocreate",
      "args": [],
      "env": {}
    }
  }
}
```

### Configuration with Other MCP Clients
GoCreate follows the standard MCP protocol and works with any MCP-compatible client. See the [gomcp documentation](https://github.com/localrivet/gomcp) for client examples.

## 🔧 Available Tools

### File System Tools

| Tool | Description | Arguments |
|------|-------------|-----------|
| `read_file` | Read file contents with optional pagination | `file_path`, `start_line?`, `end_line?` |
| `write_file` | Write content to file | `file_path`, `content` |
| `read_multiple_files` | Read multiple files at once | `file_paths[]` |
| `create_directory` | Create directory | `path` |
| `list_directory` | List directory contents | `path` |
| `move_file` | Move/rename files | `source_path`, `destination_path` |
| `search_files` | Find files by name | `path`, `pattern`, `timeout_ms?` |
| `get_file_info` | Get file metadata | `path` |

### Editing Tools

| Tool | Description | Arguments |
|------|-------------|-----------|
| `edit_block` | Replace text blocks | `file_path`, `old_string`, `new_string`, `expected_replacements?` |
| `precise_edit` | Line-based editing | `file_path`, `start_line`, `end_line`, `new_content` |

### Search Tools

| Tool | Description | Arguments |
|------|-------------|-----------|
| `search_code` | Search code with ripgrep | `path`, `pattern`, `file_pattern?`, `ignore_case?`, `max_results?`, `include_hidden?`, `context_lines?`, `timeout_ms?` |

### Terminal Tools

| Tool | Description | Arguments |
|------|-------------|-----------|
| `execute_command` | Execute terminal command | `command`, `timeout_ms?`, `shell?`, `use_powershell?` |
| `read_output` | Read command output | `pid` |
| `force_terminate` | Terminate session | `pid` |
| `list_sessions` | List active sessions | - |
| `execute_in_terminal` | Client-side terminal execution | `command`, `cwd?` |

### Process Tools

| Tool | Description | Arguments |
|------|-------------|-----------|
| `list_processes` | List running processes | - |
| `kill_process` | Terminate process by PID | `pid` |

### Configuration Tools

| Tool | Description | Arguments |
|------|-------------|-----------|
| `get_config` | Get current configuration | - |
| `set_config_value` | Set configuration value | `key`, `value` |

## 🔒 Security Features

- **Command Blocking**: Configurable list of blocked commands for security
- **File Size Limits**: 100MB limit for editing operations
- **Input Validation**: Comprehensive argument validation
- **Safe Defaults**: Secure default configurations

### Default Blocked Commands
- File system: `rm`, `mkfs`, `format`, `mount`, `umount`, `fdisk`, `dd`
- System admin: `sudo`, `su`, `passwd`, `adduser`, `useradd`, `usermod`
- System control: `shutdown`, `reboot`, `halt`, `poweroff`, `init`
- Network/Security: `iptables`, `firewall`, `netsh`

## 📁 Project Structure

```
gocreate/
├── main.go                 # Server entry point
├── config/                 # Configuration management
├── handlers/
│   ├── config/            # Configuration tools
│   ├── edit/              # Text editing tools
│   ├── filesystem/        # File system operations
│   ├── process/           # Process management
│   ├── search/            # Search functionality
│   └── terminal/          # Terminal operations
├── go.mod                 # Go module definition
└── README.md             # This file
```

## 🔄 Protocol Support

GoCreate implements the **[Model Context Protocol](https://modelcontextprotocol.io/)** with support for:
- ✅ **Tools**: All 20+ development tools
- ✅ **Structured Logging**: Using Go's `log/slog` package
- ✅ **Error Handling**: Comprehensive validation and error reporting
- ✅ **Timeout Management**: Configurable timeouts for long-running operations
- ✅ **Cross-Platform**: Unix-like systems and Windows support
- ✅ **Type Safety**: Leverages Go's type system for safety and expressiveness
- ✅ **Automatic Version Negotiation**: Seamless compatibility with MCP clients

## 🏗️ Built With

- **[gomcp](https://github.com/localrivet/gomcp)** - Complete Go implementation of Model Context Protocol
- **[ripgrep](https://github.com/BurntSushi/ripgrep)** - Fast text search engine
- **[go-diff](https://github.com/sergi/go-diff)** - Diff functionality for precise editing
- **Go 1.24+** - Modern Go features and performance

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [**gomcp**](https://github.com/localrivet/gomcp) - Go implementation of Model Context Protocol
- Search powered by [**ripgrep**](https://github.com/BurntSushi/ripgrep)
- Diff functionality using [**go-diff**](https://github.com/sergi/go-diff)
- Inspired by the [Model Context Protocol specification](https://modelcontextprotocol.io/)

## 🔗 Related Projects

- [**gomcp**](https://github.com/localrivet/gomcp) - The underlying MCP library
- [**Model Context Protocol**](https://modelcontextprotocol.io/) - Official protocol specification and documentation

---

**GoCreate** - Empowering AI assistants with comprehensive development tools 🛠️✨

*Built with ❤️ using [gomcp](https://github.com/localrivet/gomcp)*