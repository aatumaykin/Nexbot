---
name: example-skill
description: "Example skill demonstrating the OpenClaw-compatible SKILL.md format"
emoji: "📝"
version: 1.0.0
author: nexbot-team
category: example
---

# Example Skill

This is an example skill demonstrating the SKILL.md format used by Nexbot.

## Purpose

Show how to create custom skills for Nexbot using the OpenClaw-compatible format.

## When to Use

- When demonstrating skill system
- When learning about skill format
- When creating new custom skills

## How to Use

### 1. Create Skill Directory

```
skills/
└── my-skill/
    └── SKILL.md
```

### 2. Add YAML Frontmatter

The frontmatter contains metadata about the skill:

```yaml
---
name: my-skill
description: "Short description of what this skill does"
emoji: "🎯"
version: 1.0.0
author: your-name
category: automation
---
```

### 3. Add Skill Instructions

Below the frontmatter, add Markdown instructions for the agent:

```markdown
# My Skill

## Purpose
What this skill does.

## How to Use
Step-by-step instructions.
```

## Input Schema

This skill doesn't require specific inputs. Skills that require tools should define the expected tool parameters.

## Output Schema

Returns:
- Success message when skill is demonstrated
- Example of skill usage

## Examples

### Example 1: Demonstrate Skill

**User:** Show me how skills work

**Nexbot:** I'll demonstrate the skill system.

Skills are markdown files with YAML frontmatter that teach me how to use specific tools or perform certain tasks.

Here's the structure:
```
skills/
└── skill-name/
    └── SKILL.md
```

The SKILL.md file has:
- YAML frontmatter (metadata)
- Markdown body (instructions)

✅ Demonstrated skill system

### Example 2: List Available Skills

**User:** What skills do you have?

**Nexbot:** Let me check the skills directory.

[Lists skills from workspace/skills/]

Available skills:
- 📝 example-skill - Example skill demonstrating SKILL.md format
- [More skills will be listed here]

## Implementation Details

This skill is loaded progressively by Nexbot:
- Not always loaded (not in system prompt)
- Available on demand (agent can read file)
- Instructions included when agent references this skill

To use this skill, the agent would say:
> "Let me read the example-skill SKILL.md file to understand how it works."
