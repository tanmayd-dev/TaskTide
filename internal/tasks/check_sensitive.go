package tasks

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/tanmayd-dev/TaskTide/internal/models"
)

// sensitiveKeywords lists the case-insensitive markers scanned by
// DocCheckSensitive. Matches are annotated in place, not removed, so the
// document stays readable downstream while flagged terms remain visible.
var sensitiveKeywords = []string{
	"password", "passwd", "secret", "token", "api_key", "apikey",
	"private key", "credential", "authorization", "recovery code",
}

// sensitiveTermPattern compiles the keyword list into one regexp with
// case-insensitive matching and word-boundary-ish guarding on [_-] context
// (so api_key and api-key hit, but "apartment" does not). QuoteMeta runs
// before the space→[-_ ] substitution so the separator class survives as an
// actual class (multi-word keywords like "private key" become
// private[-_ ]key, not an escaped literal).
var sensitiveTermPattern = func() *regexp.Regexp {
	escaped := make([]string, 0, len(sensitiveKeywords))
	for _, kw := range sensitiveKeywords {
		esc := regexp.QuoteMeta(kw)
		escaped = append(escaped, strings.ReplaceAll(esc, " ", "[-_ ]"))
	}
	pattern := fmt.Sprintf(`(?i)\b(?:%s)\b`, strings.Join(escaped, "|"))
	return regexp.MustCompile(pattern)
}()

// normalizeTerm maps a term to its canonical aggregate form so case and
// separator variants (API_KEY, api-key, api_key) each count under the same
// keyword entry in the report.
func normalizeTerm(s string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	}), " ")
}

// DocCheckSensitive scans the input document for sensitive-looking terms,
// annotates every match, and appends a short report with per-keyword counts.
func DocCheckSensitive(task *models.WorkerTaskExecution) error {
	log.Printf("[DocCheckSensitive] Scanning task execution: %s\n", task.ID)

	content := task.TaskInput
	matches := sensitiveTermPattern.FindAllString(content, -1)

	totals := make(map[string]int, len(sensitiveKeywords))
	for _, m := range matches {
		totals[normalizeTerm(m)]++
	}

	annotated := sensitiveTermPattern.ReplaceAllString(content, "[!! $0 !!]")

	var report strings.Builder
	report.WriteString("\n--- doc.check_sensitive report ---\n")
	if len(totals) == 0 {
		report.WriteString("No sensitive terms detected.\n")
	} else {
		report.WriteString(fmt.Sprintf("Detected %d sensitive term occurrence(s):\n", len(matches)))
	}
	// Report in the keyword list's order for deterministic output.
	for _, kw := range sensitiveKeywords {
		// Variants that normalize to the same keyword (e.g. "API_KEY" or
		// "api-key" against "api_key") aggregate into one count.
		if n, ok := totals[normalizeTerm(kw)]; ok && n > 0 {
			report.WriteString(fmt.Sprintf("  %-14s : %d\n", kw, n))
		}
	}

	task.TaskOutput = annotated + report.String()
	return nil
}
