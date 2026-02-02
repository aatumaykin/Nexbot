# Tools Reference

## Built-in Tools

### File Operations

#### read_file
Read the contents of a file.

**Parameters:**
- `path` (string, required) — Path to file to read

**Returns:** File contents as string

**Example:**
```
User: What's in config.toml?
Nexbot: Let me read the file...
✅ Read config.toml: [config follows]
```

#### write_file
Write content to a file (creates or overwrites).

**Parameters:**
- `path` (string, required) — Path to file to write
- `content` (string, required) — Content to write

**Returns:** Success message

**Example:**
```
User: Create a new file called notes.md
Nexbot: I'll create that file for you.
✅ Created notes.md with placeholder content
```

#### list_dir
List directory contents.

**Parameters:**
- `path` (string, required) — Path to directory

**Returns:** List of files and directories

**Example:**
```
User: Show me what's in ~/projects
Nexbot: Let me list the directory...
✅ Found 5 directories: nexbot, nanobot, myapp, ...
```

### Shell Operations

#### shell_exec
Execute a shell command (whitelisted only).

**Parameters:**
- `command` (string, required) — Command to execute

**Returns:** Command output

**Whitelisted Commands:** ls, cat, grep, find, cd, pwd, echo, date

**Example:**
```
User: Check the current directory
Nexbot: I'll run the pwd command...
✅ Current directory: /home/user/nexbot
```

---

## Using Tools

When the LLM decides to use a tool, it will:
1. Call the tool with required parameters
2. Wait for the result
3. Process the result and respond

## Tool Safety

- All file operations are limited to whitelisted directories
- Shell commands are whitelisted for security
- Sensitive information is masked in logs

## Custom Tools

You can add custom tools by creating skills. See [SKILL_FORMAT.md](SKILL_FORMAT.md).
