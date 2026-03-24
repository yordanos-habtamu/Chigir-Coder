package prompts

import "fmt"

// PlannerPrompt builds the user-side prompt for plan generation.
func PlannerPrompt(input string, learnings string, skill string) string {
	res := fmt.Sprintf("TASK: %s\nReturn a numbered list or JSON array only. Prefer file-writing steps over shell commands.", input)
	if skill != "" {
		res += fmt.Sprintf("\nSKILL:\n%s", skill)
	}
	if learnings != "" {
		res += fmt.Sprintf("\nLEARNING: %s", learnings)
	}
	return res
}

func ExecutorPrompt(step string, stepIndex int, fullPlan []string, codeContext string, skill string) string {
	planText := ""
	for i, s := range fullPlan {
		if i == stepIndex {
			planText += fmt.Sprintf("→%d.%s\n", i+1, s)
		} else {
			planText += fmt.Sprintf(" %d.%s\n", i+1, s)
		}
	}
	res := fmt.Sprintf("PLAN:\n%s\nSTEP:%d\nCONTEXT:\n%s", planText, stepIndex+1, codeContext)
	if skill != "" {
		res += fmt.Sprintf("\nSKILL:\n%s", skill)
	}
	res += "\nOutput ONLY allowed blocks. Prefer // PATCH when file exists. Avoid npm/npx/go unless required. No markdown."
	return res
}

func FixerPrompt(brokenCode string, errorDetail string) string {
	return fmt.Sprintf("FIX: %s\nCODE:\n%s\nReturn ONLY allowed blocks. Prefer // PATCH when file exists. No markdown.", errorDetail, brokenCode)
}

func ValidatorPrompt(code string) string {
	return fmt.Sprintf("VALIDATE:\n%s\nResult: VALID or INVALID:REASON", code)
}

func CommandFixerPrompt(cmd string, output string, errDetail string) string {
	return fmt.Sprintf("COMMAND FAILED:\nCMD: %s\nSTDOUT/STDERR:\n%s\nERROR:\n%s\nReturn ONLY allowed blocks.", cmd, output, errDetail)
}
