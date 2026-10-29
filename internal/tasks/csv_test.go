package tasks

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func TestCSVReadSummarizesRows(t *testing.T) {
	input := "name,age\nalice,30\nbob,25\n"
	task := &models.WorkerTaskExecution{ID: "cr-1", TaskInput: input}
	if err := CSVRead(task); err != nil {
		t.Fatalf("CSVRead returned error: %v", err)
	}
	if !strings.HasPrefix(task.TaskOutput, "CSV parsed successfully. Total Rows: 3\n") {
		t.Fatalf("unexpected summary header, got:\n%s", task.TaskOutput)
	}
	if !strings.Contains(task.TaskOutput, "Row 2: alice | 30") {
		t.Fatalf("row rendering wrong, got:\n%s", task.TaskOutput)
	}
	if !strings.Contains(task.TaskOutput, "Row 3: bob | 25") {
		t.Fatalf("row rendering wrong, got:\n%s", task.TaskOutput)
	}
}

func TestCSVReadMalformedFails(t *testing.T) {
	// One too many fields on the second row: strict reader errors out.
	input := "a,b\n1,2,3\n"
	task := &models.WorkerTaskExecution{ID: "cr-2", TaskInput: input}
	if err := CSVRead(task); err == nil {
		t.Fatal("CSVRead should fail on a malformed CSV")
	}
}

func TestCSVValidatePassed(t *testing.T) {
	// encoding/csv skips truly blank lines; a whitespace-only line still
	// yields a one-field record and is counted as an empty row.
	task := &models.WorkerTaskExecution{ID: "cv-1", TaskInput: "a,b,c\n1,2,3\n   \n4,5,6\n"}
	if err := CSVValidate(task); err != nil {
		t.Fatalf("CSVValidate returned error: %v", err)
	}
	out := task.TaskOutput
	if !strings.Contains(out, "csv.validate: PASSED") {
		t.Fatalf("expected PASSED, got:\n%s", out)
	}
	if !strings.Contains(out, "rows: 4, empty rows: 1, columns: 3") {
		t.Fatalf("unexpected stats, got:\n%s", out)
	}
	if !strings.Contains(out, "no problems found") {
		t.Fatalf("expected no problems, got:\n%s", out)
	}
}

func TestCSVValidateFailedReportsNotFatal(t *testing.T) {
	// Row 3 has one extra field: validate reports FAILED but returns nil so
	// downstream tasks (e.g. csv.filter) still run.
	task := &models.WorkerTaskExecution{ID: "cv-2", TaskInput: "a,b\n1,2\n3,4,5\n"}
	if err := CSVValidate(task); err != nil {
		t.Fatalf("CSVValidate must not return an error on ragged rows: %v", err)
	}
	out := task.TaskOutput
	if !strings.Contains(out, "csv.validate: FAILED") {
		t.Fatalf("expected FAILED, got:\n%s", out)
	}
	if !strings.Contains(out, "row 3 has 3 field(s), expected 2") {
		t.Fatalf("missing problem line, got:\n%s", out)
	}
}

func TestCSVValidateEmptyInput(t *testing.T) {
	task := &models.WorkerTaskExecution{ID: "cv-3", TaskInput: ""}
	if err := CSVValidate(task); err != nil {
		t.Fatalf("CSVValidate returned error: %v", err)
	}
	if !strings.Contains(task.TaskOutput, "no records found") {
		t.Fatalf("expected empty-input report, got:\n%s", task.TaskOutput)
	}
}

func TestCSVFilterRemovesEmptyAndComments(t *testing.T) {
	input := "a,b\n# comment\n1,2\n   \n3,4\n"
	task := &models.WorkerTaskExecution{ID: "cf-1", TaskInput: input}
	if err := CSVFilter(task); err != nil {
		t.Fatalf("CSVFilter returned error: %v", err)
	}
	out := task.TaskOutput
	if !strings.HasPrefix(out, "# csv.filter: removed 1 empty row(s), 1 comment row(s); 3 row(s) kept\n") {
		t.Fatalf("unexpected summary line, got:\n%s", out)
	}
	if !strings.Contains(out, "a,b\n1,2\n3,4\n") {
		t.Fatalf("kept rows wrong, got:\n%s", out)
	}
	if strings.Contains(out, "# comment") && strings.Contains(out[1:], "# comment") {
		t.Fatalf("comment row must be removed, got:\n%s", out)
	}
}

func TestCSVFilterKeepsContentRowsOnly(t *testing.T) {
	input := "x\n   \n \nfoo\n"
	task := &models.WorkerTaskExecution{ID: "cf-2", TaskInput: input}
	if err := CSVFilter(task); err != nil {
		t.Fatalf("CSVFilter returned error: %v", err)
	}
	want := "# csv.filter: removed 2 empty row(s), 0 comment row(s); 2 row(s) kept\nx\nfoo\n"
	if task.TaskOutput != want {
		t.Fatalf("TaskOutput = %q, want %q", task.TaskOutput, want)
	}
}

func TestCSVAggregateStats(t *testing.T) {
	input := "name,score\nalice,30\nbob,25\ncarol,60\nbob,25\n"
	task := &models.WorkerTaskExecution{ID: "ca-1", TaskInput: input}
	if err := CSVAggregate(task); err != nil {
		t.Fatalf("CSVAggregate returned error: %v", err)
	}
	out := task.TaskOutput
	if !strings.Contains(out, "csv.aggregate: 4 data row(s), 2 column(s)") {
		t.Fatalf("unexpected summary, got:\n%s", out)
	}
	// name: 4 non-empty, 3 distinct (alice, bob, carol); not numeric.
	nameLine := fmt.Sprintf("  %-16s non-empty: %d, distinct: %d", "name", 4, 3)
	if !strings.Contains(out, nameLine+"\n") {
		t.Fatalf("name column stats wrong, got:\n%s", out)
	}
	// score: numeric stats min 25, max 60, mean (30+25+60+25)/4 = 35.
	scoreLine := fmt.Sprintf("  %-16s non-empty: %d, distinct: %d", "score", 4, 3) +
		fmt.Sprintf(", min: %g, max: %g, mean: %g", 25.0, 60.0, 35.0)
	if !strings.Contains(out, scoreLine+"\n") {
		t.Fatalf("score column stats wrong, got:\n%s", out)
	}
}

func TestCSVAggregateEmptyInput(t *testing.T) {
	task := &models.WorkerTaskExecution{ID: "ca-2", TaskInput: ""}
	if err := CSVAggregate(task); err != nil {
		t.Fatalf("CSVAggregate returned error: %v", err)
	}
	if !strings.Contains(task.TaskOutput, "no records found") {
		t.Fatalf("expected empty-input report, got:\n%s", task.TaskOutput)
	}
}
