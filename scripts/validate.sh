#!/usr/bin/env bash
set -euo pipefail

ROOT="$(pwd)"
echo "🔎 Validation starting in ${ROOT}"

has_cmd() {
  command -v "$1" >/dev/null 2>&1
}

has_script() {
  local name="$1"
  if ! has_cmd node; then
    return 1
  fi
  node -e "const p=require('./package.json');process.exit(p.scripts && p.scripts['${name}']?0:1)"
}

run_script() {
  local pm="$1"
  local script="$2"
  echo "▶︎ Running ${pm} run ${script}"
  "${pm}" run "${script}"
}

ran_any=false

# Go validation
if [ -f "go.mod" ]; then
  if has_cmd go; then
    echo "▶︎ Running go test ./..."
    go test ./...
    ran_any=true
  else
    echo "⚠️  go.mod found but 'go' not available; skipping Go tests"
  fi
fi

# Frontend validation
if [ -f "package.json" ]; then
  pm=""
  if has_cmd pnpm && [ -f "pnpm-lock.yaml" ]; then
    pm="pnpm"
  elif has_cmd yarn && [ -f "yarn.lock" ]; then
    pm="yarn"
  elif has_cmd npm; then
    pm="npm"
  elif has_cmd corepack; then
    echo "⚠️  corepack present but no package manager detected; skipping frontend scripts"
  else
    echo "⚠️  package.json found but no npm/yarn/pnpm available; skipping frontend scripts"
  fi

  if [ -n "${pm}" ]; then
    if has_script "lint"; then
      run_script "${pm}" "lint"
      ran_any=true
    elif has_script "typecheck"; then
      run_script "${pm}" "typecheck"
      ran_any=true
    elif has_script "test"; then
      run_script "${pm}" "test"
      ran_any=true
    elif has_script "build"; then
      run_script "${pm}" "build"
      ran_any=true
    else
      echo "⚠️  No lint/typecheck/test/build scripts found in package.json; skipping frontend validation"
    fi
  fi
fi

if [ "${ran_any}" = false ]; then
  echo "⚠️  No validation commands executed (no applicable project files found)"
fi

echo "✅ Validation complete"
