package main

import (
	"strings"
	"testing"
)

func TestRenderPlainMarkdown(t *testing.T) {
	r := renderer{color: false, width: 80}
	got := r.render(`# Title

Some **bold** text with [a link](https://example.com).

- item one
- item two

` + "```go" + `
fmt.Println("hello")
` + "```" + `
`)

	checks := []string{
		"# Title",
		"Some bold text with a link (https://example.com).",
		"- item one",
		"fmt.Println(\"hello\")",
	}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Fatalf("rendered output missing %q:\n%s", check, got)
		}
	}
}

func TestTableRendering(t *testing.T) {
	r := renderer{color: false, width: 80}
	got := r.render(`| Name | Value |
| ---- | ----- |
| One  | **Two** |
`)

	if !strings.Contains(got, "+------+-------+") {
		t.Fatalf("table border not rendered:\n%s", got)
	}
	if !strings.Contains(got, "| Name | Value |") {
		t.Fatalf("table header not rendered:\n%s", got)
	}
	if !strings.Contains(got, "| One  | Two   |") {
		t.Fatalf("table body not rendered:\n%s", got)
	}
}

func TestTableAlignmentAndEscapedPipes(t *testing.T) {
	r := renderer{color: false, width: 80}
	got := r.render(`| Left | Right | Center | Pipe |
| :--- | ----: | :----: | ---- |
| a | 12 | mid | A \| B |
| code | 7 | x | ` + "`a|b`" + ` |
`)

	checks := []string{
		"| a    |    12 |  mid   | A | B |",
		"| code |     7 |   x    | a|b   |",
	}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Fatalf("rendered table missing %q:\n%s", check, got)
		}
	}
}

func TestTableWrapsLongCells(t *testing.T) {
	r := renderer{color: false, width: 34}
	got := r.render(`| Name | Description |
| ---- | ----------- |
| One | This cell should wrap nicely |
`)

	checks := []string{
		"+------+-------------------------+",
		"| One  | This cell should wrap   |",
		"|      | nicely                  |",
	}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Fatalf("wrapped table missing %q:\n%s", check, got)
		}
	}
}

func TestWrap(t *testing.T) {
	lines := wrapANSI("one two three four five", 9)
	want := []string{"one two", "three four", "five"}
	if strings.Join(lines, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q, want %q", lines, want)
	}
}

func TestParseArgs(t *testing.T) {
	fullScreen, filename, help, ok := parseArgs([]string{"-f", "README.md"})
	if !ok || !fullScreen || filename != "README.md" {
		t.Fatalf("unexpected args parse: fullScreen=%v filename=%q help=%v ok=%v", fullScreen, filename, help, ok)
	}

	fullScreen, filename, help, ok = parseArgs([]string{"README.md"})
	if !ok || fullScreen || filename != "README.md" {
		t.Fatalf("unexpected default args parse: fullScreen=%v filename=%q help=%v ok=%v", fullScreen, filename, help, ok)
	}

	fullScreen, filename, help, ok = parseArgs([]string{"--help"})
	if !ok || fullScreen || filename != "" || !help {
		t.Fatalf("unexpected help args parse: fullScreen=%v filename=%q help=%v ok=%v", fullScreen, filename, help, ok)
	}

	_, _, _, ok = parseArgs([]string{"-x", "README.md"})
	if ok {
		t.Fatal("unknown flags should fail")
	}
}
