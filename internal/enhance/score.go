package enhance

import (
	"regexp"
	"strings"
)

// Score returns a 0–100 quality heuristic for a prompt text.
// Ported from the Desktop plugin's score() function.
// The baseline is 40; bonuses add up to 60.
func Score(text string) int {
	if text == "" {
		return 0
	}
	lines := splitLines(text)
	trimmed := filter(lines, func(l string) bool { return strings.TrimSpace(l) != "" })
	if len(trimmed) == 0 {
		return 0
	}

	// Count numbered items (1. 2. 3.)
	numbered := 0
	numberRe := regexp.MustCompile(`^\d+\.\s`)
	for _, l := range trimmed {
		if numberRe.MatchString(l) {
			numbered++
		}
	}

	// Goal heuristic: first line is 6+ chars, starts with uppercase, not a number
	hasGoal := false
	if len(trimmed) > 0 {
		first := strings.TrimSpace(trimmed[0])
		if len(first) >= 6 && first[0] >= 'A' && first[0] <= 'Z' && (first[0] < '0' || first[0] > '9') {
			hasGoal = true
		}
	}

	// First word is an action verb
	words := strings.Fields(strings.ToLower(text))
	hasVerb := false
	if len(words) > 0 {
		first := stripPunct(words[0])
		hasVerb = actionVerbs[first]
	}

	// Contains specific domain nouns
	specifics := regexp.MustCompile(`(?i)\b(file|table|column|class|function|method|module|component|api|endpoint|test|case|user|customer|order|product|item|message|thread|task|ticket|issue|branch|commit|pr|deploy|build|run|sprint)\b`).MatchString(text)

	// Word count in the "just right" range
	wordCount := len(words)
	goodLength := wordCount >= 30 && wordCount <= 200

	s := 40 // baseline
	if hasGoal {
		s += 10
	}
	if numbered >= 1 {
		s += min(20, numbered*4)
	}
	if hasVerb {
		s += 10
	}
	if specifics {
		s += 10
	}
	if goodLength {
		s += 10
	}
	if s > 100 {
		s = 100
	}
	return s
}

// ScorePair holds before/after scores for display.
type ScorePair struct {
	Before int
	After  int
}

func stripPunct(s string) string {
	return strings.Trim(s, ".,!?;:\"'")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func filter(vs []string, fn func(string) bool) []string {
	r := make([]string, 0)
	for _, v := range vs {
		if fn(v) {
			r = append(r, v)
		}
	}
	return r
}

// actionVerbs is a set of common imperative action verbs.
// Go has no built-in set — use a map for O(1) lookup.
var actionVerbs = map[string]bool{
	"build":     true, "create": true, "make": true, "fix": true, "write": true,
	"add":      true, "remove":   true, "update": true, "edit":   true,
	"design":   true, "implement": true, "deploy": true, "test":  true,
	"review":   true, "refactor":  true, "rewrite": true, "ship":  true,
	"merge":    true, "open":      true, "close":   true, "send":  true,
	"list":     true, "check":     true, "find":    true, "search": true,
	"read":     true, "parse":     true, "render":  true, "save":  true,
	"load":     true, "install":   true, "configure": true, "set": true,
	"delete":   true, "sort":      true, "filter":  true, "show":  true,
	"explain":  true, "describe":  true, "compare": true, "analyze": true,
	"analyse":  true, "measure":   true, "log":    true, "track":  true,
	"monitor":  true, "schedule":  true, "generate": true, "compute": true,
	"calculate": true, "draft": true, "plan": true, "break": true,
	"split":    true, "combine":   true, "tag":    true, "label":  true,
	"rename":   true, "move":      true, "copy":   true, "export": true,
	"import":   true, "submit":    true, "validate": true, "verify": true,
	"confirm":  true, "ensure":    true, "trace":  true,
}
