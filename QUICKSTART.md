# Quick Start Guide

Get Nexbot up and running in 5 minutes! 🚀

## Prerequisites

- Go 1.21+ (if building from source)
- Telegram account
- [Z.ai API key](https://z.ai)
- Telegram bot token from [@BotFather](https://t.me/BotFather)

## Step 1: Install (2 minutes)

### Option A: Download Binary (Fastest)

```bash
# Download latest release for your platform
curl -L https://github.com/aatumaykin/nexbot/releases/latest/download/nexbot-$(uname -s)-$(uname -m) -o nexbot

# Or manually download from:
# https://github.com/aatumaykin/nexbot/releases

# Make it executable
chmod +x nexbot

# Move to PATH
sudo mv nexbot /usr/local/bin/
```

### Option B: Build from Source

```bash
# Clone repository
git clone https://github.com/aatumaykin/nexbot.git
cd nexbot

# Build
make build

# Install
make install
```

## Step 2: Get API Keys (2 minutes)

### Get Z.ai API Key

1. Go to [Z.ai](https://z.ai)
2. Sign up / Sign in
3. Go to API Keys section
4. Generate a new API key
5. Copy the key (starts with `zai-` or `sk-`)

### Get Telegram Bot Token

1. Open Telegram and search for [@BotFather](https://t.me/BotFather)
2. Send `/newbot` command
3. Follow the instructions:
   - Choose a name for your bot (e.g., "My Nexbot")
   - Choose a username (e.g., "my_nexbot_bot")
4. BotFather will give you a token (e.g., `1234567890:ABCdefGHIjklMNOpqrsTUVwxyz`)
5. Copy the token

## Step 3: Configure (30 seconds)

```bash
# Create configuration directory
mkdir -p ~/.nexbot

# Create config file
cat > ~/.nexbot/config.toml << 'EOF'
[workspace]
path = "~/.nexbot"

[llm]
provider = "zai"

[llm.zai]
api_key = "YOUR_ZAI_API_KEY_HERE"
base_url = "https://api.z.ai/api/coding/paas/v4"
model = "glm-4.7-flash"

[channels.telegram]
enabled = true
token = "YOUR_TELEGRAM_BOT_TOKEN_HERE"
allowed_users = []  # Empty means allow all users

[logging]
level = "info"
format = "json"
output = "stdout"
EOF

# Replace with your actual keys
nano ~/.nexbot/config.toml
```

Or use environment variables (more secure):

```bash
cat > ~/.nexbot/config.toml << 'EOF'
[workspace]
path = "~/.nexbot"

[llm]
provider = "zai"

[llm.zai]
api_key = "${ZAI_API_KEY}"
base_url = "https://api.z.ai/api/coding/paas/v4"
model = "glm-4.7-flash"

[channels.telegram]
enabled = true
token = "${TELEGRAM_BOT_TOKEN}"
allowed_users = []
EOF

# Set environment variables
export ZAI_API_KEY="zai-your-api-key-here"
export TELEGRAM_BOT_TOKEN="1234567890:ABCdefGHIjklMNOpqrsTUVwxyz"

# Add to ~/.bashrc or ~/.zshrc for persistence
echo 'export ZAI_API_KEY="zai-your-api-key-here"' >> ~/.bashrc
echo 'export TELEGRAM_BOT_TOKEN="1234567890:ABCdefGHIjklMNOpqrsTUVwxyz"' >> ~/.bashrc
source ~/.bashrc
```

## Step 4: Validate Configuration (10 seconds)

```bash
# Validate your configuration
nexbot config validate

# Should see: ✅ Configuration is valid
```

If validation fails, check:
- API key format (should start with `zai-` or `sk-`)
- Telegram token format (should be `bot_id:token`)
- No trailing whitespace in values

## Step 5: Run (10 seconds)

```bash
# Start Nexbot
nexbot serve
```

You should see output like:

```
✅ Configuration loaded
✅ Logger initialized
✅ Message bus started
✅ Telegram bot initialized (bot_id: 1234567890, username: my_nexbot_bot)
✅ Inbound message queue subscribed
✅ Outbound message queue subscribed
✅ Nexbot is running
```

## Step 6: Test Your Bot

1. Open Telegram
2. Search for your bot (the username you chose)
3. Start a chat
4. Send a message like: "Hello, who are you?"
5. Bot should respond!

## Common Issues

### Issue: "telegram token is required"
- **Solution:** Make sure you replaced `YOUR_TELEGRAM_BOT_TOKEN_HERE` in config

### Issue: "Z.ai API key is required"
- **Solution:** Make sure you replaced `YOUR_ZAI_API_KEY_HERE` in config

### Issue: "Authentication error"
- **Solution:** Check your API key is correct and not expired

### Issue: Bot doesn't respond
- **Solution:**
  - Check that bot is enabled in config: `channels.telegram.enabled = true`
  - Verify your Telegram token is correct
  - Check logs for errors

### Issue: "Permission denied" when installing
- **Solution:** Use `sudo make install` or install to `~/bin` instead: `make install-user`

## Next Steps

Now that Nexbot is running:

1. **Create bootstrap files** in `~/.nexbot/`:
   - `IDENTITY.md` - Core identity
   - `AGENTS.md` - Agent instructions
   - `SOUL.md` - Bot personality
   - `USER.md` - Your profile

2. **Create skills** in `~/.nexbot/skills/`:
   - Each skill is a `SKILL.md` file with YAML frontmatter
   - Teach the bot new capabilities

3. **Configure allowed users** for security:
   ```toml
   [channels.telegram]
   allowed_users = ["123456789", "987654321"]  # Only allow specific Telegram users
   ```

4. **Explore advanced configuration**:
   - See `config.example.toml` for all options
   - Read [CONFIG.md](CONFIG.md) for detailed reference

## Getting Help

- **Documentation:** Check [README.md](README.md) for overview
- **Config Reference:** [CONFIG.md](CONFIG.md) for all options
- **Issues:** [GitHub Issues](https://github.com/aatumaykin/nexbot/issues)
- **Test components:** `nexbot test llm` to test LLM connectivity

## Uninstalling

```bash
# Stop the bot (Ctrl+C)
# Remove binary
sudo rm /usr/local/bin/nexbot

# Remove configuration (optional)
rm -rf ~/.nexbot
```

---

**Time elapsed:** ~5 minutes 🎉

Happy chatting with Nexbot! 🤖
