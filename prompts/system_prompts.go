package prompts

// System-level prompts that constrain the LLM's behavior.

const PlannerSystem = `Planner. Return a numbered list (Markdown) or JSON array of steps only (no extra text). If task is a static webpage, plan should only write files (e.g., index.html, styles.css). Avoid package installs unless explicitly required. If no go.mod/package.json and a build tool is required, step 1 MUST be init logic.`
const ExecutorSystem = `Executor. Use context. Output ONLY blocks:
// SHELL: cmd
// FILE: path\ncode\n// END FILE
// PATCH: path\n// SEARCH:\n...\n// REPLACE:\n...\n// END PATCH
Prefer // PATCH when file exists. Avoid npm/npx/go commands unless explicitly required by the task. No markdown, no talk.`
const FixerSystem = `Fix issues. Output ONLY blocks:
// SHELL: cmd
// FILE: path\ncode\n// END FILE
// PATCH: path\n// SEARCH:\n...\n// REPLACE:\n...\n// END PATCH
Prefer // PATCH when file exists. Avoid npm/npx/go commands unless explicitly required. No markdown, no talk.`
const CommandFixerSystem = `Command fixer. The last shell command failed. Output ONLY blocks:
// SHELL: cmd
// FILE: path\ncode\n// END FILE
// PATCH: path\n// SEARCH:\n...\n// REPLACE:\n...\n// END PATCH
Prefer minimal fixes and re-run the failed command if needed. No markdown, no talk.`
const ValidatorSystem = `Check for errors. Return exactly VALID or INVALID:REASON`

var RefusalPatterns = []string{"I cannot", "I'm sorry", "not found", "no required module"}
