package enhance

import "regexp"

// systemPrompt is the prompt injected as the assistant's system message.
// Keep in sync with the Desktop plugin's inline SYSTEM string.
const systemPrompt = `You are a prompt engineer. Rewrite user requests into clear, well-structured agent prompts.

Your job is NOT to answer the request — it is to reformulate it so an AI agent can execute it correctly on the first attempt.

## Format

Write ONE goal line followed by numbered items (1. 2. 3.). Preserve the original language.

**For short or already-clear inputs:** return the same intent in fewer words as 1-3 tight bullets. Do not pad.

**For substantive inputs (multi-sentence, real task, no clear shape):** produce a production-grade brief:

1. **Goal** — one line stating what success looks like
2. **Role & context** — who the agent is, what tools it has, any runtime constraints
3. **Scope** — what is in scope (IN:) and what is out of scope (OUT:)
4. **Requirements** — 3-7 concrete numbered requirements, each one a single verifiable action
5. **Deliverable** — what form the output takes: files, format, coverage
6. **Quality gates** — what "done" means, how the agent should verify completion
7. **Anti-patterns** — 2-4 things to NOT do

Use only the sections that apply. Do not invent sections.

## Rules

- Match the original language (English/Malay/etc.)
- Do NOT answer the request — restate it as an actionable task
- Do NOT add goals the user did not mention
- Do NOT write tutorial-style output ("First, do X, then Y...")
- Prefer concrete nouns over vague ones ("table" not "the data structure")
- Cap output at 1200 characters`

// userWrap wraps raw user input into the user message sent to the model.
const userWrap = "Rewrite this request as an agent prompt:\n\n%s\n\n---"

// UserMessage formats raw user text into the user-role message sent to the model.
func UserMessage(input string) string {
	return "Rewrite this request as an agent prompt:\n\n" + input + "\n\n---"
}

// Cleaning regexes — ported from the Desktop plugin's stripQuotes.
var (
	thinkBlock  = regexp.MustCompile(`(?is)<\|[\s\S]*?\|>`)
	channelWrap = regexp.MustCompile(`(?m)^<\|[\w:-]+\|>[^\n]*\n?`)
	fenceOpen   = regexp.MustCompile("(?m)^```[a-zA-Z0-9_-]*\n?")
	fenceClose  = regexp.MustCompile("(?m)\n?```$")
	outerQuotes = regexp.MustCompile("^['\u201c\u201d\u2018\u2019]+|['\u201c\u201d\u2018\u2019]+$")
)

// Clean strips thinking blocks, channel wrappers, markdown fences, and outer
// quotes from the raw model output. This is the final step before returning
// the enhanced prompt to the user.
func Clean(text string) string {
	if text == "" {
		return ""
	}
	s := thinkBlock.ReplaceAllString(text, "")
	s = channelWrap.ReplaceAllString(s, "")
	s = fenceOpen.ReplaceAllString(s, "")
	s = fenceClose.ReplaceAllString(s, "")
	s = outerQuotes.ReplaceAllString(s, "")
	return trimLines(s)
}

// trimLines removes leading/trailing blank lines while preserving intentional
// blank lines in the middle of the output.
func trimLines(s string) string {
	lines := splitLines(s)
	i := 0
	for i < len(lines) && lines[i] == "" {
		i++
	}
	j := len(lines)
	for j > i && lines[j-1] == "" {
		j--
	}
	return joinLines(lines[i:j])
}

func splitLines(s string) []string {
	return regexp.MustCompile(`\r?\n`).Split(s, -1)
}

func joinLines(lines []string) string {
	result := make([]byte, 0, len(lines)*20)
	for i, l := range lines {
		if i > 0 {
			result = append(result, '\n')
		}
		result = append(result, l...)
	}
	return string(result)
}
