# Long-term Memory

## Important Information

Remember these details for future interactions:

- User prefers concise responses
- Primary timezone: UTC (update from USER.md)
- Main project: Nexbot (AI agent in Go)
- Backup location: ~/backups

## Project Notes

### Nexbot
- **Tech Stack:** Go 1.21+, telego, Z.ai LLM
- **Architecture:** Simple loop + message bus
- **Goal:** Ultra-lightweight personal AI assistant (~8-10K lines)
- **Status:** MVP in progress

### Nanobot (Reference)
- **Language:** Python 3.11+
- **Lines of Code:** ~4,602 (99% smaller than Clawdbot)
- **Key Features:** Message bus, progressive skills loading, subagents
- **Inspiration:** Nanobot's simple loop architecture

## User Preferences

- Prefer code snippets over long explanations
- Use bullet points for lists
- Confirm before executing commands
- Show confidence levels for decisions
- Keep responses under 3-4 sentences when possible

## Commands & Shortcuts

- Make builds: `make build-all`
- Run tests: `make test`
- Start bot: `./nexbot` or `make run`
- Check status: `./nexbot status`
- Validate config: `./nexbot validate`

## API Keys & Tokens

- **Z.ai API:** Stored in .env (never commit!)
- **Telegram Token:** Stored in .env (never commit!)

---

**Note:** This file is loaded into system prompt for context. Edit to add important information.
