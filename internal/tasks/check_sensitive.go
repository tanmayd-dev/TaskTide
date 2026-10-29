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
// (so api_key and api-key hit, but "apartment" does not).
var sensitiveTermPattern = func() *regexp.Regexp {
	escaped := make([]string, 0, len(sensitiveKeywords))
	for _, kw := range sensitiveKeywords {
		escaped = append(escaped, regexp.QuoteMeta(strings.ReplaceAll(kw, " ", "[-_ ]")))
	}
	pattern := fmt.Sprintf(`(?i)\b(?:%s)\b`, strings.Join(escaped, "|"))
	return regexp.MustCompile(pattern)
}()

// DocCheckSensitive scans the input document for sensitive-looking terms,
// annotates every match, and appends a short report with per-keyword counts.
func DocCheckSensitive(task *models.WorkerTaskExecution) error {
	log.Printf("[DocCheckSensitive] Scanning task execution: %s\n", task.ID)

	content := task.TaskInput
	matches := sensitiveTermPattern.FindAllString(content, -1)

	totals := make(map[string]int, len(sensitiveKeywords))
	for _, m := range matches {
		totals[strings.ToLower(m)]++
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
		// Accumulate counts for variants that normalize to the same keyword
		// (e.g. "API_KEY" lowercases to "api_key").
		if n, ok := totals[strings.ToLower(kw)]; ok && n > 0 {
			report.WriteString(fmt.Sprintf("  %-14s : %d\n", kw, n))
		}
	}

	task.TaskOutput = annotated + report.String()
	return nil
}
