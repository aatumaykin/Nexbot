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

### Agent Operations

#### spawn
Create a subagent for parallel task execution with isolated session.

**Parameters:**
- `task` (string, required) — Task description for subagent
- `timeout_seconds` (number, optional) — Timeout in seconds (default: 300)

**Returns:**
- Result of task execution by subagent

**How it works:**
1. LLM calls spawn("task description")
2. A new isolated subagent is created with its own session
3. Subagent executes task with full access to all tools
4. **Synchronous execution**: Parent agent waits for result
5. Subagent returns result to parent session
6. Subagent is automatically deleted after task completion
7. LLM sees result and continues conversation

**Capabilities:**
- ✅ Full tool access (file, shell, spawn for nested tasks!)
- ✅ Isolated session and memory
- ✅ Automatic cleanup after completion
- ✅ Timeout handling

**Example:**
```
User: Analyze last 10 git commits and create summary

LLM: I'll create a subagent for this analysis.
[spawn: "Read git log -10 --oneline, analyze changes by type
        (feat, fix, docs, chore), create summary"]

[Subagent executes task]
[Result returned to LLM]

LLM: Here's analysis of last 10 commits:

**Summary:**
- Total: 10 commits
- Features (feat): 4
- Fixes (fix): 3
- Documentation (docs): 2
- Chores (chore): 1

**Details:**
1. feat: Add spawn tool implementation
2. fix: Fix memory leak in subagent manager
...
```

**Nested Spawn:**
Subagents can create other subagents for parallel task execution:

```
[Parent Agent]
  └─> [spawn: "Check system status and create 3 subagents:
         one for network, one for services, one for logs"]
       └─> [Subagent A - coordinator]
             ├─> [spawn: "Check network"]
             ├─> [spawn: "Check services"]
             └─> [spawn: "Check logs"]
             (all execute in parallel)
             └─> Results collected
                  └─> Report returned to Parent
```

**Use cases:**
- Parallel file processing
- Log analysis
- System monitoring
- Batch operations
- Complex multi-step tasks

**Limitations:**
- Maximum concurrent: configured in `[subagent].max_concurrent` (default: 10)
- Timeout: default 300 seconds, configurable per call
- Subagents are deleted after task completion (no persistent state)
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
