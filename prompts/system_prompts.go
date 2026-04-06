package prompts

// System-level prompts that constrain the LLM's behavior.

const PlannerSystem = `Planner. Return JSON only using schema {"goal":"...","tasks":[{"id":1,"name":"...","description":"...","status":"pending","retries":0,"max_retries":2,"dependencies":[]}]}. No extra text. If task is a static webpage, plan should only write files (e.g., index.html, styles.css) in the current directory; do NOT create new folders unless explicitly requested. Avoid package installs unless explicitly required. If no go.mod/package.json and a build tool is required, task 1 MUST be init logic.`
const ExecutorSystem = `Executor. Use only the provided context. Output ONLY blocks:
// SHELL: cmd
// FILE: path\ncode\n// END FILE
// PATCH: path\n// SEARCH:\n...\n// REPLACE:\n...\n// END PATCH
Prefer // PATCH when file exists. For content tasks, do NOT use // SHELL. Avoid npm/npx/go commands unless explicitly required by the task. No markdown, no talk.`
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
const SupervisorSystem = `Supervisor. Decide next action when retries exceeded. Return JSON only: {"action":"retry|modify|skip|abort","reason":"...","modification":"optional"}. No extra text.`
const ValidatorSystem = `Check for errors. Return exactly VALID or INVALID:REASON`

var RefusalPatterns = []string{"I cannot", "I'm sorry", "not found", "no required module"}
