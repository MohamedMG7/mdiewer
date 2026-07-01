package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"unicode"
)

const (
	reset     = "\x1b[0m"
	bold      = "\x1b[1m"
	dim       = "\x1b[2m"
	italic    = "\x1b[3m"
	underline = "\x1b[4m"
	blue      = "\x1b[34m"
	cyan      = "\x1b[36m"
	green     = "\x1b[32m"
	magenta   = "\x1b[35m"
	yellow    = "\x1b[33m"
)

var (
	headingPattern        = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*#*\s*$`)
	unorderedListPattern  = regexp.MustCompile(`^(\s*)([-+*])\s+(.+)$`)
	orderedListPattern    = regexp.MustCompile(`^(\s*)(\d+[.)])\s+(.+)$`)
	horizontalRulePattern = regexp.MustCompile(`^\s{0,3}((---+)|(\*\*\*+)|(___+))\s*$`)
	ansiPattern           = regexp.MustCompile(`\x1b\[[0-9;]*m`)
)

type renderer struct {
	color bool
	width int
}

type tableAlign int

const (
	alignLeft tableAlign = iota
	alignCenter
	alignRight
)

func main() {
	fullScreen, filename, help, ok := parseArgs(os.Args[1:])
	if help {
		printHelp(os.Stdout)
		return
	}

	if !ok {
		printHelp(os.Stderr)
		os.Exit(2)
	}

	source, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mdiewer: %v\n", err)
		os.Exit(1)
	}

	r := renderer{
		color: os.Getenv("NO_COLOR") == "",
		width: max(30, terminalWidth()),
	}

	if fullScreen {
		clearTerminal()
	}
	fmt.Print(r.render(string(source)))
}

func printHelp(out *os.File) {
	fmt.Fprintln(out, "mdiewer renders a Markdown file in your terminal.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  mdiewer <filename.md>")
	fmt.Fprintln(out, "  mdiewer -f <filename.md>")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Options:")
	fmt.Fprintln(out, "  -f          Clear the terminal before rendering")
	fmt.Fprintln(out, "  -h, --help  Show this help")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Examples:")
	fmt.Fprintln(out, "  mdiewer README.md")
	fmt.Fprintln(out, "  mdiewer -f ./docs/spec.md")
}

func clearTerminal() {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		if cmd.Run() == nil {
			return
		}
	}

	fmt.Print("\x1b[2J\x1b[H")
}

func parseArgs(args []string) (bool, string, bool, bool) {
	fullScreen := false
	filename := ""
	help := false

	for _, arg := range args {
		switch arg {
		case "-f":
			if fullScreen {
				return false, "", false, false
			}
			fullScreen = true
		case "-h", "--help":
			if len(args) != 1 {
				return false, "", false, false
			}
			help = true
		default:
			if strings.HasPrefix(arg, "-") || filename != "" {
				return false, "", false, false
			}
			filename = arg
		}
	}

	return fullScreen, filename, help, help || filename != ""
}

func terminalWidth() int {
	value := strings.TrimSpace(os.Getenv("COLUMNS"))
	if value == "" {
		return 100
	}
	width, err := strconv.Atoi(value)
	if err != nil || width <= 0 {
		return 100
	}
	return width
}

func (r renderer) render(markdown string) string {
	lines := strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n")
	var out strings.Builder
	inFence := false
	fenceMarker := ""

	for i := 0; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], " \t")
		trimmed := strings.TrimSpace(line)

		if isFenceStart(trimmed) {
			marker, label := parseFence(trimmed)
			if !inFence {
				inFence = true
				fenceMarker = marker
				if label != "" {
					displayLabel := label
					if r.color {
						displayLabel = " " + label + " "
					}
					r.writeLine(&out, r.style(displayLabel, dim+yellow))
				}
				continue
			}
			if strings.HasPrefix(trimmed, fenceMarker) {
				inFence = false
				fenceMarker = ""
				r.blankLine(&out)
				continue
			}
		}

		if inFence {
			r.writeLine(&out, r.style("  "+line, cyan))
			continue
		}

		if trimmed == "" {
			r.blankLine(&out)
			continue
		}

		if table, next := r.tryRenderTable(lines, i); table != "" {
			out.WriteString(table)
			i = next
			continue
		}

		if horizontalRulePattern.MatchString(trimmed) {
			r.writeLine(&out, r.style(strings.Repeat("-", min(r.width, 72)), dim))
			r.blankLine(&out)
			continue
		}

		if match := headingPattern.FindStringSubmatch(trimmed); match != nil {
			r.renderHeading(&out, len(match[1]), match[2])
			continue
		}

		if strings.HasPrefix(trimmed, ">") {
			r.renderQuote(&out, trimmed)
			continue
		}

		if match := unorderedListPattern.FindStringSubmatch(line); match != nil {
			r.renderListItem(&out, match[1], "-", match[3])
			continue
		}

		if match := orderedListPattern.FindStringSubmatch(line); match != nil {
			r.renderListItem(&out, match[1], match[2], match[3])
			continue
		}

		paragraph, next := collectParagraph(lines, i)
		r.renderParagraph(&out, paragraph)
		i = next
	}

	return strings.TrimRight(out.String(), "\n") + "\n"
}

func (r renderer) tryRenderTable(lines []string, start int) (string, int) {
	if start+1 >= len(lines) || !looksLikeTableRow(lines[start]) {
		return "", start
	}

	alignments, ok := parseTableSeparator(lines[start+1])
	if !ok {
		return "", start
	}

	var rows [][]string
	rows = append(rows, splitTableRow(lines[start]))
	i := start + 2
	for ; i < len(lines) && looksLikeTableRow(lines[i]); i++ {
		rows = append(rows, splitTableRow(lines[i]))
	}

	colCount := 0
	colCount = max(colCount, len(alignments))
	for _, row := range rows {
		colCount = max(colCount, len(row))
	}
	for len(alignments) < colCount {
		alignments = append(alignments, alignLeft)
	}

	rawWidths := make([]int, colCount)
	for _, row := range rows {
		for c, cell := range row {
			rawWidths[c] = max(rawWidths[c], visibleLen(r.renderInline(strings.TrimSpace(cell))))
		}
	}

	widths := fitTableWidths(rawWidths, r.width)
	var out strings.Builder
	r.renderTableBorder(&out, widths)
	r.renderTableRow(&out, rows[0], widths, alignments, true)
	r.renderTableBorder(&out, widths)
	for _, row := range rows[1:] {
		r.renderTableRow(&out, row, widths, alignments, false)
	}
	r.renderTableBorder(&out, widths)
	out.WriteString("\n")
	return out.String(), i - 1
}

func fitTableWidths(rawWidths []int, terminalWidth int) []int {
	widths := append([]int(nil), rawWidths...)
	if len(widths) == 0 {
		return widths
	}

	for i, width := range widths {
		widths[i] = max(3, width)
	}

	available := terminalWidth - (3*len(widths) + 1)
	if available < len(widths)*3 {
		available = len(widths) * 3
	}

	for sumInts(widths) > available {
		largest := -1
		for i, width := range widths {
			if width <= 3 {
				continue
			}
			if largest == -1 || width > widths[largest] {
				largest = i
			}
		}
		if largest == -1 {
			break
		}
		widths[largest]--
	}

	return widths
}

func (r renderer) renderTableBorder(out *strings.Builder, widths []int) {
	out.WriteString(r.style("+", dim))
	for _, width := range widths {
		out.WriteString(r.style(strings.Repeat("-", width+2), dim))
		out.WriteString(r.style("+", dim))
	}
	out.WriteByte('\n')
}

func (r renderer) renderTableRow(out *strings.Builder, row []string, widths []int, alignments []tableAlign, header bool) {
	cells := make([][]string, len(widths))
	height := 1
	for c := range widths {
		cell := ""
		if c < len(row) {
			cell = r.renderInline(strings.TrimSpace(row[c]))
			if header {
				cell = r.style(cell, bold)
			}
		}
		cells[c] = wrapANSITableCell(cell, widths[c])
		height = max(height, len(cells[c]))
	}

	for lineIndex := 0; lineIndex < height; lineIndex++ {
		out.WriteString(r.style("|", dim))
		for c, width := range widths {
			cellLine := ""
			if lineIndex < len(cells[c]) {
				cellLine = cells[c][lineIndex]
			}
			out.WriteByte(' ')
			out.WriteString(padVisible(cellLine, width, alignments[c]))
			out.WriteByte(' ')
			out.WriteString(r.style("|", dim))
		}
		out.WriteByte('\n')
	}
}

func looksLikeTableRow(line string) bool {
	return len(splitTableRow(line)) >= 2
}

func parseTableSeparator(line string) ([]tableAlign, bool) {
	cells := splitTableRow(line)
	if len(cells) == 0 {
		return nil, false
	}

	alignments := make([]tableAlign, 0, len(cells))
	for _, cell := range cells {
		cell = strings.TrimSpace(cell)
		if len(cell) < 3 {
			return nil, false
		}

		leftColon := strings.HasPrefix(cell, ":")
		rightColon := strings.HasSuffix(cell, ":")
		dashes := strings.Trim(cell, ":")
		if len(dashes) < 3 || strings.Trim(dashes, "-") != "" {
			return nil, false
		}

		switch {
		case leftColon && rightColon:
			alignments = append(alignments, alignCenter)
		case rightColon:
			alignments = append(alignments, alignRight)
		default:
			alignments = append(alignments, alignLeft)
		}
	}

	return alignments, true
}

func splitTableRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil
	}

	var cells []string
	var cell strings.Builder
	escaped := false
	inCode := false

	for i := 0; i < len(trimmed); i++ {
		ch := trimmed[i]
		switch {
		case escaped:
			if ch != '|' {
				cell.WriteByte('\\')
			}
			cell.WriteByte(ch)
			escaped = false
		case ch == '\\':
			escaped = true
		case ch == '`':
			inCode = !inCode
			cell.WriteByte(ch)
		case ch == '|' && !inCode:
			cells = append(cells, cell.String())
			cell.Reset()
		default:
			cell.WriteByte(ch)
		}
	}
	if escaped {
		cell.WriteByte('\\')
	}
	cells = append(cells, cell.String())

	if len(cells) > 0 && strings.TrimSpace(cells[0]) == "" {
		cells = cells[1:]
	}
	if len(cells) > 0 && strings.TrimSpace(cells[len(cells)-1]) == "" {
		cells = cells[:len(cells)-1]
	}
	return cells
}

func (r renderer) renderHeading(out *strings.Builder, level int, text string) {
	prefix := strings.Repeat("#", level) + " "
	content := r.renderInline(text)
	switch level {
	case 1:
		r.writeLine(out, r.style(prefix+content, bold+underline+magenta))
		r.writeLine(out, r.style(strings.Repeat("=", min(visibleLen(text)+2, r.width)), magenta))
	case 2:
		r.writeLine(out, r.style(prefix+content, bold+blue))
	default:
		r.writeLine(out, r.style(prefix+content, bold))
	}
	r.blankLine(out)
}

func (r renderer) renderQuote(out *strings.Builder, line string) {
	content := strings.TrimSpace(strings.TrimLeft(line, ">"))
	wrapped := wrapANSI(r.renderInline(content), r.width-4)
	for _, wrappedLine := range wrapped {
		r.writeLine(out, r.style("> ", green)+wrappedLine)
	}
}

func (r renderer) renderListItem(out *strings.Builder, indent, marker, text string) {
	prefix := strings.Repeat(" ", visualIndent(indent)) + r.style(marker+" ", yellow)
	continuation := strings.Repeat(" ", visualIndent(indent)+visibleLen(marker)+1)
	wrapped := wrapANSI(r.renderInline(text), r.width-visibleLen(stripANSI(prefix)))
	for i, line := range wrapped {
		if i == 0 {
			r.writeLine(out, prefix+line)
			continue
		}
		r.writeLine(out, continuation+line)
	}
}

func (r renderer) renderParagraph(out *strings.Builder, text string) {
	for _, line := range wrapANSI(r.renderInline(text), r.width) {
		r.writeLine(out, line)
	}
	r.blankLine(out)
}

func collectParagraph(lines []string, start int) (string, int) {
	parts := []string{strings.TrimSpace(lines[start])}
	i := start + 1
	for ; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], " \t")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" ||
			headingPattern.MatchString(trimmed) ||
			horizontalRulePattern.MatchString(trimmed) ||
			isFenceStart(trimmed) ||
			strings.HasPrefix(trimmed, ">") ||
			unorderedListPattern.MatchString(line) ||
			orderedListPattern.MatchString(line) ||
			(i+1 < len(lines) && looksLikeTableRow(line) && isTableSeparator(lines[i+1])) {
			break
		}
		parts = append(parts, trimmed)
	}
	return strings.Join(parts, " "), i - 1
}

func isTableSeparator(line string) bool {
	_, ok := parseTableSeparator(line)
	return ok
}

func (r renderer) renderInline(text string) string {
	var out strings.Builder
	for i := 0; i < len(text); {
		switch {
		case text[i] == '`':
			if end := strings.IndexByte(text[i+1:], '`'); end >= 0 {
				code := text[i+1 : i+1+end]
				out.WriteString(r.style(code, cyan))
				i += end + 2
				continue
			}
		case strings.HasPrefix(text[i:], "**"):
			if end := strings.Index(text[i+2:], "**"); end >= 0 {
				out.WriteString(r.style(r.renderInline(text[i+2:i+2+end]), bold))
				i += end + 4
				continue
			}
		case strings.HasPrefix(text[i:], "__"):
			if end := strings.Index(text[i+2:], "__"); end >= 0 {
				out.WriteString(r.style(r.renderInline(text[i+2:i+2+end]), bold))
				i += end + 4
				continue
			}
		case text[i] == '*' || text[i] == '_':
			marker := text[i]
			if end := strings.IndexByte(text[i+1:], marker); end >= 0 {
				inside := text[i+1 : i+1+end]
				if strings.TrimSpace(inside) != "" {
					out.WriteString(r.style(inside, italic))
					i += end + 2
					continue
				}
			}
		case text[i] == '[':
			if rendered, consumed, ok := r.tryRenderLink(text[i:]); ok {
				out.WriteString(rendered)
				i += consumed
				continue
			}
		}
		out.WriteByte(text[i])
		i++
	}
	return out.String()
}

func (r renderer) tryRenderLink(text string) (string, int, bool) {
	closeText := strings.IndexByte(text, ']')
	if closeText <= 0 || closeText+1 >= len(text) || text[closeText+1] != '(' {
		return "", 0, false
	}
	closeURL := strings.IndexByte(text[closeText+2:], ')')
	if closeURL < 0 {
		return "", 0, false
	}

	label := text[1:closeText]
	url := text[closeText+2 : closeText+2+closeURL]
	rendered := r.style(label, underline+blue)
	if url != "" {
		rendered += r.style(" ("+url+")", dim)
	}
	return rendered, closeText + closeURL + 3, true
}

func (r renderer) style(text, sequence string) string {
	if !r.color || text == "" {
		return text
	}
	return sequence + text + reset
}

func (r renderer) writeLine(out *strings.Builder, line string) {
	out.WriteString(line)
	out.WriteByte('\n')
}

func (r renderer) blankLine(out *strings.Builder) {
	if out.Len() == 0 {
		return
	}
	text := out.String()
	if strings.HasSuffix(text, "\n\n") {
		return
	}
	out.WriteByte('\n')
}

func wrapANSI(text string, width int) []string {
	width = max(10, width)
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	current := words[0]
	for _, word := range words[1:] {
		nextLen := visibleLen(current) + 1 + visibleLen(word)
		if nextLen <= width {
			current += " " + word
			continue
		}
		lines = append(lines, current)
		current = word
	}
	lines = append(lines, current)
	return lines
}

func wrapANSITableCell(text string, width int) []string {
	width = max(1, width)
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	current := words[0]
	for _, word := range words[1:] {
		if visibleLen(current)+1+visibleLen(word) <= width {
			current += " " + word
			continue
		}
		lines = append(lines, current)
		current = word
	}
	lines = append(lines, current)
	return lines
}

func padVisible(text string, width int, align tableAlign) string {
	padding := max(0, width-visibleLen(text))
	switch align {
	case alignRight:
		return strings.Repeat(" ", padding) + text
	case alignCenter:
		left := padding / 2
		right := padding - left
		return strings.Repeat(" ", left) + text + strings.Repeat(" ", right)
	default:
		return text + strings.Repeat(" ", padding)
	}
}

func sumInts(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}

func visibleLen(text string) int {
	return len([]rune(stripANSI(text)))
}

func stripANSI(text string) string {
	return ansiPattern.ReplaceAllString(text, "")
}

func isFenceStart(line string) bool {
	return strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~")
}

func parseFence(line string) (string, string) {
	if len(line) < 3 {
		return line, ""
	}
	marker := line[:3]
	label := strings.TrimSpace(line[3:])
	return marker, label
}

func visualIndent(indent string) int {
	count := 0
	for _, ch := range indent {
		if ch == '\t' {
			count += 4
			continue
		}
		if unicode.IsSpace(ch) {
			count++
		}
	}
	return count
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
