Good—now we lock everything into something you can actually build without drifting.

I’m going to give you a **complete, implementation-ready blueprint** for your system:

> A deterministic coding engine with planning, execution, validation, and self-learning (with plan diffing)

No fluff—this is structured so you can start coding immediately.

---

# 🧠 SYSTEM NAME (optional but useful)

**zen-coder**
(A deterministic AI coding orchestrator for weak models)

---

# 🎯 CORE OBJECTIVE

Build a CLI tool that:

1. Takes messy user input
2. Compiles it into structured prompts
3. Generates a plan
4. Allows user to revise the plan
5. Learns from plan diffs
6. Executes step-by-step
7. Validates + fixes outputs
8. Stores knowledge for reuse

---

# 🧱 HIGH-LEVEL REQUIREMENTS

---

## ✅ Functional Requirements

### 1. Input Handling

* Accept free-form user input
* Normalize and extract keywords

---

### 2. Prompt Compiler

* Convert input → structured prompt
* Inject constraints + output format
* Use templates

---

### 3. Planner

* Generate step-by-step plan (AI-assisted)
* Output must be:

  * ordered
  * atomic
  * editable

---

### 4. Plan Editor (CRITICAL FEATURE)

* Show plan to user
* Allow editing (via:

  * CLI editor OR
  * opening `$EDITOR`)
* Save both:

  * original plan
  * revised plan

---

### 5. Diff Learning Engine

* Compute:

  * added steps
  * removed steps
* Store learning patterns
* Map patterns → keywords

---

### 6. Execution Engine

* Execute one step at a time
* Inject:

  * plan
  * current step
  * minimal context

---

### 7. Validator

* Detect:

  * incomplete code
  * TODOs
  * empty responses

---

### 8. Fixer Loop

* Auto-retry failed outputs
* Send correction prompt

---

### 9. Memory System (Self-learning)

* Store:

  * input
  * plan
  * learned improvements
* Reuse for similar inputs

---

### 10. Context Manager

* Keep only:

  * relevant structs
  * function signatures
* Avoid full file dumps

---

# ⚙️ NON-FUNCTIONAL REQUIREMENTS

* Fast (CLI-based)
* Low token usage
* Deterministic behavior
* Extensible knowledge base
* Model-agnostic (works with your Trinity model)

---

# 🧩 SYSTEM FLOW

```text
User Input
   ↓
[Compiler]
   ↓
[Planner]
   ↓
[Plan Editor] ← YOU MODIFY HERE
   ↓
[Diff Engine] → [Learning Store]
   ↓
Execution Loop:
   → Executor (step-by-step)
   → Validator
   → Fixer
   → Context Update
   ↓
[Memory Store]
```

---

# 📁 FOLDER STRUCTURE (clean + scalable)

```bash
zen-coder/
│
├── main.go
│
├── /cmd/                  # CLI commands
│   └── run.go
│
├── /compiler/
│   └── compiler.go
│
├── /planner/
│   └── planner.go
│
├── /executor/
│   └── executor.go
│
├── /validator/
│   └── validator.go
│
├── /fixer/
│   └── fixer.go
│
├── /memory/
│   ├── memory.go
│   ├── history.json
│   └── learning.json
│
├── /diff/
│   └── diff.go
│
├── /context/
│   └── context.go
│
├── /prompts/
│   ├── templates.go
│   └── system_prompts.go
│
├── /models/
│   └── client.go          # API calls to Trinity
│
└── /utils/
    └── helpers.go
```

---

# 🧠 DATA STRUCTURES (important)

---

## Plan

```go
type Plan struct {
    Steps []string
}
```

---

## Record (memory)

```go
type Record struct {
    Input        string
    OriginalPlan []string
    RevisedPlan  []string
    Tags         []string
}
```

---

## Learning Pattern

```go
type Learning struct {
    Pattern string   // e.g. "auth"
    Added   []string
    Count   int
}
```

---

# 🔧 CORE MODULE BREAKDOWN

---

## 1. Compiler

**Responsibility:**

* Clean input
* Build structured prompt

---

## 2. Planner

**Responsibility:**

* Generate plan via model
* Parse steps

---

## 3. Plan Editor

Options:

* Write to temp file → open `$EDITOR`
* Read back edited version

---

## 4. Diff Engine

**Responsibility:**

* Compare original vs revised
* Extract learning

---

## 5. Learning Engine

**Responsibility:**

* Store improvements
* Apply to future plans

---

## 6. Executor

**Responsibility:**

* Execute one step only
* Prevent drift

---

## 7. Validator

**Responsibility:**

* Detect bad outputs

---

## 8. Fixer

**Responsibility:**

* Retry with correction prompt

---

## 9. Context Manager

**Responsibility:**

* Maintain minimal state
* Compress code context

---

## 10. Model Client

**Responsibility:**

* Send prompts
* Handle responses

---

# 🔁 EXECUTION LOOP (FINAL FORM)

```go
plan := GeneratePlan(input)

editedPlan := OpenEditor(plan)

diff := DiffPlans(plan, editedPlan)

Learn(diff, input)

for step := range editedPlan {
    output := ExecuteStep(step, context)

    if !Validate(output) {
        output = Fix(output)
    }

    context = UpdateContext(output)
}

SaveMemory(input, plan, editedPlan)
```

---

# 🔥 PROMPT STRATEGY (VERY IMPORTANT)

---

## Planner Prompt

```text
Break this feature into atomic coding steps.
Return ONLY a numbered list.
```

---

## Executor Prompt

```text
You are executing step X:

[STEP]

PLAN:
[full plan]

RULES:
- Only execute this step
- Do not move ahead
- Code only
```

---

## Fixer Prompt

```text
Fix this code. No explanation. Return only corrected version.
```

---

# 🚀 PHASED BUILD PLAN (so you don’t get overwhelmed)

---

## Phase 1 (2–4 hours)

* CLI input
* Planner
* Executor (basic)
* No memory

---

## Phase 2

* Plan editor
* Diff engine
* Learning storage

---

## Phase 3

* Validator + Fixer loop
* Context manager

---

## Phase 4 (optional)

* Smarter matching
* Plan reuse
* Ranking system

---

# 🧠 FINAL INSIGHT

What you’re building is:

> **A control system for unreliable intelligence**

Not:

* an AI tool
* not an IDE plugin

But:

> a **deterministic execution pipeline**

---


