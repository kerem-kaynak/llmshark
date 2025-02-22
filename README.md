# LLMShark

LLMShark is a Terminal User Interface (TUI) application designed to streamline your PostgreSQL database interactions, particularly when working with Large Language Models (LLMs). It allows you to explore your database structure, add/edit comments, and export schema information in Markdown format, all from your terminal.

Sharing the data model and explaining column relationships can be tedious and time-consuming. LLMShark was born out of this pain, providing a way to quickly generate a Markdown representation of your database schema, ready to be used in your LLM prompts.

## Features

- 🌳 Tree-based database schema explorer
- 💬 Add and edit comments on tables and columns
- 📝 Markdown export capability for LLM prompting
- 🔒 Secure credential management
- 🎨 User-friendly terminal interface

## Security

- Database credentials are encrypted using AES-GCM
- Encryption keys are stored separately from credentials
- Credentials are saved in your home directory (`~/.llmshark`)
- File permissions are set to 600 (user read/write only)

## Installation

### Using the Installation Script (Recommended)

1. Clone the repository:
```bash
git clone https://github.com/kerem-kaynak/llmshark.git
   ```
2. Navigate to the project directory:
```bash
cd llmshark
```
3. Run the installation script:
```bash
make install
```

The script will automatically:
- Build the LLMShark binary
- Install it to your Go binary directory
- Add the Go binary directory to your PATH

### Manual Installation

```bash
go install github.com/kerem-kaynak/llmshark@latest
```

## Usage

Simply run:
```bash
llmshark
```

On first run, you'll be prompted for your PostgreSQL connection details:
- Host
- Port
- Database name
- Username
- Password

These credentials will be securely stored for future use.

### Navigation

- `↑/↓` or `j/k`: Navigate items
- `→/←` or `l/h`: Expand/collapse items
- `Space`: Select/deselect items
- `c`: Add/edit comment on selected item
- `m`: Copy schema as markdown
- `d`: Deselect all items
- `e`: Edit connection details
- `q`: Quit

## LLM Prompting Workflow

LLMShark simplifies the process of prompting LLMs about your database:

1. **Explore your schema:** Use LLMShark to navigate your database structure.
2. **Add descriptions:** Add helpful descriptions to tables and columns using the `c` key. These descriptions will be included in the Markdown output.
3. **Select relevant parts:** Use the `Space` key to select the schemas, tables, and columns relevant to your prompt.
4. **Export to Markdown:** Press `m` to copy the selected schema information to your clipboard in Markdown format.
5. **Paste into your LLM prompt:** Paste the Markdown output into your LLM prompt to provide context about your database.

This workflow allows you to quickly and accurately provide LLMs with the information they need to understand your database and generate effective queries or insights.

## Configuration

LLMShark stores its configuration in `~/.llmshark/`:
- `credentials.enc`: Encrypted database credentials
- `credentials.enc.key`: Encryption key

## Credential Management

LLMShark uses AES-GCM encryption to secure your database credentials:

1. On first run, a random 32-byte key is generated
2. Credentials are encrypted using AES-GCM with this key
3. Encrypted credentials are stored in `~/.llmshark/credentials.enc`
4. The encryption key is stored separately in `~/.llmshark/credentials.enc.key`
5. Both files are created with 600 permissions (user read/write only)

To reset credentials:
1. Delete the files in `~/.llmshark/`
2. Run `llmshark` again

## Development

### Requirements

- Go 1.21 or higher
- PostgreSQL 12 or higher

### Building from Source

```bash
make build
```

## License

This project is licensed under the [MIT License](LICENSE.md).

## Contributing

Contributions are welcome! Please fork the repository, create a new branch for your feature or bug fix, and submit a pull request.
