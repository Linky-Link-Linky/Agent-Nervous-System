package mcp

import (
	"strings"
	"unicode"
)

// OptimizeResult describes what was pruned.
type OptimizeResult struct {
	Pruned    bool
	PrunedLen int
	Output    string
}

// OptimizeContext prunes repetitive data strings from content.
// Returns the optimized output and what was removed.
func OptimizeContext(content string) OptimizeResult {
	original := len(content)
	var sb strings.Builder
	sb.Grow(original)

	// Scan for repeated blocks of the same line or line prefix
	lines := strings.Split(content, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if len(line) < 10 {
			if i > 0 {
				sb.WriteByte('\n')
			}
			sb.WriteString(line)
			continue
		}
		// Count consecutive repetitions of the same line
		repeatCount := 1
		for j := i + 1; j < len(lines); j++ {
			if lines[j] == line {
				repeatCount++
			} else {
				break
			}
		}
		if repeatCount >= 4 {
			if sb.Len() > 0 {
				sb.WriteByte('\n')
			}
			sb.WriteString(line)
			sb.WriteString(" [" + itoa(repeatCount) + "x repeated]")
			i += repeatCount - 1
		} else {
			if i > 0 {
				sb.WriteByte('\n')
			}
			sb.WriteString(line)
		}
	}

	// Also collapse consecutive duplicate blocks of shorter same-line content
	// by scanning for long base64-like strings
	result := sb.String()

	// Prune very long continuous alphanumeric+/= sequences (base64-like)
	pruned := pruneLongBase64(&result)

	// Prune repeated whitespace lines (more than 5 consecutive blank lines)
	result = collapseBlankLines(result)

	if len(result) < original || pruned {
		return OptimizeResult{
			Pruned:    true,
			PrunedLen: original - len(result),
			Output:    result,
		}
	}
	return OptimizeResult{Output: result}
}

func isBase64char(ch rune) bool {
	return unicode.IsDigit(ch) || unicode.IsLetter(ch) || ch == '+' || ch == '/' || ch == '='
}

func pruneLongBase64(s *string) bool {
	var out strings.Builder
	out.Grow(len(*s))
	runes := []rune(*s)
	i := 0
	pruned := false
	for i < len(runes) {
		if isBase64char(runes[i]) {
			start := i
			for i < len(runes) && isBase64char(runes[i]) {
				i++
			}
			runLen := i - start
			if runLen > 200 {
				out.WriteString("[LONG DATA: " + itoa(runLen) + " chars]")
				pruned = true
			} else {
				out.WriteString(string(runes[start:i]))
			}
		} else {
			out.WriteRune(runes[i])
			i++
		}
	}
	*s = out.String()
	return pruned
}

func collapseBlankLines(s string) string {
	var out strings.Builder
	out.Grow(len(s))
	consecutiveNewlines := 0
	for _, ch := range s {
		if ch == '\n' {
			consecutiveNewlines++
			if consecutiveNewlines <= 2 {
				out.WriteRune(ch)
			}
		} else {
			consecutiveNewlines = 0
			out.WriteRune(ch)
		}
	}
	return out.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
