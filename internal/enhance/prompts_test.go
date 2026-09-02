package enhance

import "testing"

func TestCleanStripsThinkBlock(t *testing.T) {
	// M3 channel-style <|...|> blocks (what M3 emits)
	in := "<|begin_think|>\nthinking...\n<|end_think|>\nRewrite the task."
	got := Clean(in)
	if got == "" {
		t.Fatalf("Clean returned empty")
	}
	// Should not start with the channel tag
	if len(got) > 0 && got[0] == '<' {
		t.Errorf("Clean left opening tag: %q", got[:min(20, len(got))])
	}
}

func TestCleanStripsFences(t *testing.T) {
	in := "```\nGoal: ship it\n1. step one\n2. step two\n```"
	got := Clean(in)
	if got == "" {
		t.Errorf("Clean returned empty for fenced input: %q", in)
	}
	if len(got) > 0 && got[0] == '`' {
		t.Errorf("Clean left opening fence: %q", got)
	}
}

func TestCleanStripsQuotes(t *testing.T) {
	in := "\u201cRewrite me\u201d"
	got := Clean(in)
	if got != "Rewrite me" {
		t.Errorf("Clean(%q) = %q, want %q", in, got, "Rewrite me")
	}
}

func TestUserMessageFormat(t *testing.T) {
	got := UserMessage("fix the login")
	want := "Rewrite this request as an agent prompt:\n\nfix the login\n\n---"
	if got != want {
		t.Errorf("UserMessage = %q, want %q", got, want)
	}
}

func TestCleanEmpty(t *testing.T) {
	if got := Clean(""); got != "" {
		t.Errorf("Clean(\"\") = %q, want \"\"", got)
	}
}
