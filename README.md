# zen-coder (PromptOs)

A deterministic, plan-first coding orchestrator that can run standalone or via MCP (Cline/Roo Code). It generates a strict plan, executes step-by-step, validates outputs, applies patches, and self-heals when commands fail.

---

## Why it exists
- **Reliability**: strict plan → review → execute loop
- **Precision**: patch-based edits for minimal diffs
- **Safety**: command allowlist and sandboxed execution
- **Speed**: small token budgets, deterministic prompts

---

## Quick Start

### 1) Build
```bash
go build -o zen-coder .
```

### 2) Configure
You can use **either** local config or environment variables.

Local config (repo root): `./config.yaml`
```yaml
provider: nvidia
base_url: "https://integrate.api.nvidia.com/v1/chat/completions"
model: "qwen/qwen3.5-122b-a10b"
api_key: "YOUR_KEY"
max_tokens: 300
skill: auto
```

Environment:
```bash
export ZEN_PROVIDER=nvidia
export ZEN_BASE_URL="https://integrate.api.nvidia.com/v1/chat/completions"
export ZEN_MODEL="qwen/qwen3.5-122b-a10b"
export ZEN_API_KEY="YOUR_KEY"
```

### 3) Run
```bash
./zen-coder run "build a beautiful landing page for a soccer team"
```

---

## CLI

### Run
```bash
./zen-coder run "your task"
```

Common flags:
```bash
--provider nvidia
--model qwen/qwen3.5-122b-a10b
--base-url https://integrate.api.nvidia.com/v1/chat/completions
--max-tokens 300
--output json
--skill landing-page
--no-edit
```

### Init config
```bash
./zen-coder init
```

### Serve via MCP
```bash
./zen-coder serve
```

### Skills
```bash
./zen-coder skill list
./zen-coder skill add my-skill
./zen-coder skill add my-skill --file /path/to/skill.md
```

---

## Config Locations (in order)
1. `./config.yaml`
2. `./config/config.yaml`
3. `~/.config/zen-coder/config.yaml`
4. `/etc/zen-coder/config.yaml`

> The active config path prints at startup.

---

## Skills System
Skills are Claude-style prompt modules that guide planning and execution.

**Built-in skills:**
- `landing-page`, `portfolio`, `dashboard`, `marketing`, `docs`
- `api`, `cli`, `refactor`, `tests`, `data-pipeline`
- `react`, `next`, `general`

**Skill file format** (`skills/<name>.md`):
```
SKILL: landing-page
GOAL: ...
STRUCTURE: ...
STYLE: ...
DELIVERABLES: ...
CONSTRAINTS: ...
```

Skills are loaded from:
- `./skills/`
- `<binary dir>/skills/`
- `~/.config/zen-coder/skills/`
- `/etc/zen-coder/skills/`

---

## Reliability Features (Phase 1)

### 1) Strict Plan → Review → Execute
- Planner outputs **numbered list** or **JSON array** of steps only.

### 2) Diff-Aware Writing
- Uses patch blocks when files already exist:
```
// PATCH: path/to/file
// SEARCH:
old text
// REPLACE:
new text
// END PATCH
```

### 3) Self-Healing Loop
- If a shell command fails, the error is fed back to the model.
- The model returns fixes (commands, patches, or files) automatically.

### 4) Command Sandboxing
- Commands are restricted by `command_allowlist` in config.

---

## Output Formats
The executor only accepts these blocks:
```
// SHELL: cmd
// FILE: path
content
// END FILE
// PATCH: path
// SEARCH:
...
// REPLACE:
...
// END PATCH
```

---

## MCP (Cline/Roo Code)
Example MCP config:
```json
{
  "mcpServers": {
    "zen-coder": {
      "command": "/path/to/zen-coder",
      "args": ["serve"],
      "env": {
        "ZEN_API_KEY": "YOUR_KEY",
        "ZEN_PROVIDER": "nvidia",
        "ZEN_MODEL": "qwen/qwen3.5-122b-a10b",
        "ZEN_BASE_URL": "https://integrate.api.nvidia.com/v1/chat/completions"
      },
      "autoApprove": ["zen_run"]
    }
  }
}
```

---

## Architecture (High Level)
- `compiler/` — normalizes task into structured prompt
- `planner/` — plan generation + parsing
- `executor/` — runs steps, validates, self-heals
- `applier/` — writes files + applies patches
- `validator/` — checks output quality and structure
- `skills/` — prompt modules

---

## Common Troubleshooting

### “Missing Authentication header”
- Ensure `ZEN_API_KEY` is set or `api_key` exists in config.
- Avoid empty `ZEN_API_KEY` env var overriding config.

### 401 Unauthorized
- Confirm the model name is correct (include vendor prefix).
- Ensure the key is valid for the NVIDIA endpoint.

### 404 page not found
- If using the full endpoint, ensure you’re running the latest binary.
- Prefer `https://integrate.api.nvidia.com/v1` as base URL.

---

## Roadmap (Next Phases)
- Context summarization
- Project auto-detection
- Smart file placement rules
- Lint/format/test gates
- Design system injection for UI tasks

---

## Security
- Never commit API keys
- Use env vars or local `config.yaml`
- Rotate keys if leaked

---

## License
MIT (update as needed)
