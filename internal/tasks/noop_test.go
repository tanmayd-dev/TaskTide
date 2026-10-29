package tasks

import (
	"testing"

	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func TestNoopAppendsText(t *testing.T) {
	task := &models.WorkerTaskExecution{ID: "noop-1", TaskInput: "Hello TaskTide!"}
	if err := Noop(task); err != nil {
		t.Fatalf("Noop returned error: %v", err)
	}
	want := "Hello TaskTide!\nNOOP appended text from noop"
	if task.TaskOutput != want {
		t.Fatalf("TaskOutput = %q, want %q", task.TaskOutput, want)
	}
}

func TestNoopDefaultZeroDelay(t *testing.T) {
	// Without NOOP_TIME, Noop must behave like a 0s sleep, not error.
	t.Setenv("NOOP_TIME", "")
	task := &models.WorkerTaskExecution{ID: "noop-2", TaskInput: "x"}
	if err := Noop(task); err != nil {
		t.Fatalf("Noop with unset NOOP_TIME returned error: %v", err)
	}
}

func TestNoopInvalidDelay(t *testing.T) {
	t.Setenv("NOOP_TIME", "not-a-number")
	task := &models.WorkerTaskExecution{ID: "noop-3", TaskInput: "x"}
	if err := Noop(task); err == nil {
		t.Fatal("Noop with invalid NOOP_TIME should fail")
	}
}

func TestNoopSimulatedPermanentFailure(t *testing.T) {
	t.Setenv("ENABLE_TEST_SIMULATORS", "true")
	task := &models.WorkerTaskExecution{ID: "noop-4", TaskInput: "FAIL"}
	if err := Noop(task); err == nil {
		t.Fatal("Noop with FAIL input should fail when simulators are enabled")
	}
}

func TestNoopSimulatedTransientFailureOnce(t *testing.T) {
	t.Setenv("ENABLE_TEST_SIMULATORS", "true")
	resetFailedTasks()
	id := models.TaskExecutionID("noop-5")

	first := &models.WorkerTaskExecution{ID: id, TaskInput: "FAIL_ONCE"}
	if err := Noop(first); err == nil {
		t.Fatal("first FAIL_ONCE attempt should fail")
	}

	second := &models.WorkerTaskExecution{ID: id, TaskInput: "FAIL_ONCE"}
	if err := Noop(second); err != nil {
		t.Fatalf("second FAIL_ONCE attempt should succeed, got: %v", err)
	}
	want := "FAIL_ONCE\nNOOP appended text from noop"
	if second.TaskOutput != want {
		t.Fatalf("TaskOutput = %q, want %q", second.TaskOutput, want)
	}
}

func TestNoopSimulatorsDisabledWithoutEnv(t *testing.T) {
	t.Setenv("ENABLE_TEST_SIMULATORS", "")
	task := &models.WorkerTaskExecution{ID: "noop-6", TaskInput: "FAIL"}
	if err := Noop(task); err != nil {
		t.Fatalf("FAIL input must be inert when simulators disabled: %v", err)
	}
}

func TestOutputAppendsText(t *testing.T) {
	task := &models.WorkerTaskExecution{ID: "out-1", TaskInput: "final"}
	if err := Output(task); err != nil {
		t.Fatalf("Output returned error: %v", err)
	}
	want := "final\nNOOP appended text from output"
	if task.TaskOutput != want {
		t.Fatalf("TaskOutput = %q, want %q", task.TaskOutput, want)
	}
}

func TestOutputDefaultZeroDelay(t *testing.T) {
	// Regression: Output used to fail when NOOP_TIME was unset.
	t.Setenv("NOOP_TIME", "")
	task := &models.WorkerTaskExecution{ID: "out-2", TaskInput: "x"}
	if err := Output(task); err != nil {
		t.Fatalf("Output with unset NOOP_TIME returned error: %v", err)
	}
}

func TestOutputInvalidDelay(t *testing.T) {
	t.Setenv("NOOP_TIME", "not-a-number")
	task := &models.WorkerTaskExecution{ID: "out-3", TaskInput: "x"}
	if err := Output(task); err == nil {
		t.Fatal("Output with invalid NOOP_TIME should fail")
	}
}

func TestDocReadEmptyInput(t *testing.T) {
	task := &models.WorkerTaskExecution{ID: "doc-2"}
	if err := DocRead(task); err != nil {
		t.Fatalf("DocRead returned error: %v", err)
	}
	if task.TaskOutput != "" {
		t.Fatalf("TaskOutput = %q, want empty passthrough", task.TaskOutput)
	}
}

func resetFailedTasks() {
	failedMu.Lock()
	defer failedMu.Unlock()
	failedTasks = make(map[string]bool)
}
