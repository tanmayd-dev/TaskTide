package tasks

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func TestDocCheckSensitiveAnnotatesAndCounts(t *testing.T) {
	input := "the password is hunter2 and the api_key = abc123"
	task := &models.WorkerTaskExecution{ID: "sens-1", TaskInput: input}
	if err := DocCheckSensitive(task); err != nil {
		t.Fatalf("DocCheckSensitive returned error: %v", err)
	}

	if !strings.Contains(task.TaskOutput, "[!! password !!]") {
		t.Fatalf("output should annotate 'password', got:\n%s", task.TaskOutput)
	}
	if !strings.Contains(task.TaskOutput, "[!! api_key !!]") {
		t.Fatalf("output should annotate 'api_key', got:\n%s", task.TaskOutput)
	}
	// Non-sensitive content must survive annotation untouched.
	if !strings.Contains(task.TaskOutput, "hunter2") {
		t.Fatalf("non-sensitive content lost, got:\n%s", task.TaskOutput)
	}
	if !strings.Contains(task.TaskOutput, "--- doc.check_sensitive report ---") {
		t.Fatalf("output should contain the report header, got:\n%s", task.TaskOutput)
	}
	if !strings.Contains(task.TaskOutput, "Detected 2 sensitive term occurrence(s):") {
		t.Fatalf("output should count 2 occurrences, got:\n%s", task.TaskOutput)
	}
	// Keyword report lists each detected keyword with its count.
	report := task.TaskOutput[strings.Index(task.TaskOutput, "--- doc.check_sensitive report ---"):]
	if !strings.Contains(report, "password") || !strings.Contains(report, "api_key") {
		t.Fatalf("report should list detected keywords, got:\n%s", report)
	}
}

func TestDocCheckSensitiveNoMatches(t *testing.T) {
	task := &models.WorkerTaskExecution{ID: "sens-2", TaskInput: "nothing sensitive here"}
	if err := DocCheckSensitive(task); err != nil {
		t.Fatalf("DocCheckSensitive returned error: %v", err)
	}
	if !strings.Contains(task.TaskOutput, "No sensitive terms detected.") {
		t.Fatalf("expected clean-document report, got:\n%s", task.TaskOutput)
	}
	if strings.Contains(task.TaskOutput, "[!!") {
		t.Fatalf("nothing should be annotated, got:\n%s", task.TaskOutput)
	}
}

func TestDocCheckSensitiveVariantsAndBoundaries(t *testing.T) {
	// Case-insensitive matches annotate the original casing. "[-_ ]" variants
	// (api-key) hit the api_key keyword. Underscore is a Go \b word char, so
	// MY_TOKEN must NOT match; "apartment" must NOT match 'token' (word
	// boundary); multi-word "private key" matches "private-key" too.
	input := "SECRET secret my-token API_KEY apartment private-key"
	task := &models.WorkerTaskExecution{ID: "sens-3", TaskInput: input}
	if err := DocCheckSensitive(task); err != nil {
		t.Fatalf("DocCheckSensitive returned error: %v", err)
	}

	out := task.TaskOutput
	// "my-token" matches the 'token' keyword (hyphen is a Go \b boundary), so
	// the annotation wraps the matched substring: my-[!! token !!]. A fixed
	// multi-word keyword matches the full variant: [!! private-key !!].
	for _, want := range []string{"[!! SECRET !!]", "[!! secret !!]", "my-[!! token !!]", "[!! API_KEY !!]", "[!! private-key !!]"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing annotation %q, got:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "Detected 5 sensitive term occurrence(s):") {
		t.Fatalf("expected 5 occurrences, got:\n%s", out)
	}
	// Variants aggregate in the report: secret x2; token x1 (my-token);
	// api_key x1 (API_KEY); private key x1 (private-key).
	report := out[strings.Index(out, "--- doc.check_sensitive report ---"):]
	wantLines := []string{
		fmt.Sprintf("  %-14s : %d", "secret", 2),
		fmt.Sprintf("  %-14s : %d", "token", 1),
		fmt.Sprintf("  %-14s : %d", "api_key", 1),
		fmt.Sprintf("  %-14s : %d", "private key", 1),
	}
	for _, line := range wantLines {
		if !strings.Contains(report, line) {
			t.Fatalf("report missing %q, got:\n%s", line, report)
		}
	}
}
