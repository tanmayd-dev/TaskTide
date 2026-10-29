package tasks

import (
	"encoding/csv"
	"fmt"
	"log"
	"strings"

	"github.com/tanmayd-dev/TaskTide/internal/models"
)

// CSVValidate checks CSV integrity: consistency of column counts across
// rows and completely empty rows. It reports problems but does not treat
// them as fatal unless parsing itself fails, so partially-broken files can
// still flow to downstream repair tasks (e.g. csv.filter).
func CSVValidate(task *models.WorkerTaskExecution) error {
	log.Printf("[CSVValidate] Validating CSV task execution: %s\n", task.ID)

	reader := csv.NewReader(strings.NewReader(task.TaskInput))
	reader.FieldsPerRecord = -1 // report per-row shapes instead of failing early
	records, err := reader.ReadAll()
	if err != nil {
		log.Printf("[CSVValidate] Error parsing CSV: %v\n", err)
		return fmt.Errorf("failed to parse CSV: %w", err)
	}
	if len(records) == 0 {
		task.TaskOutput = "csv.validate: FAILED — no records found (empty input)\n"
		return nil
	}

	columns := len(records[0])
	var problems []string
	emptyRows := 0
	for i, record := range records {
		if len(record) == 1 && strings.TrimSpace(record[0]) == "" {
			emptyRows++
			continue
		}
		if len(record) != columns {
			problems = append(problems,
				fmt.Sprintf("row %d has %d field(s), expected %d", i+1, len(record), columns))
		}
	}

	var buf strings.Builder
	status := "PASSED"
	if len(problems) > 0 {
		status = "FAILED"
	}
	buf.WriteString(fmt.Sprintf("csv.validate: %s\n", status))
	buf.WriteString(fmt.Sprintf("rows: %d, empty rows: %d, columns: %d\n", len(records), emptyRows, columns))
	if len(problems) > 0 {
		buf.WriteString(fmt.Sprintf("%d problem(s):\n", len(problems)))
		for _, p := range problems {
			buf.WriteString("  - " + p + "\n")
		}
	} else {
		buf.WriteString("no problems found\n")
	}

	task.TaskOutput = buf.String()
	return nil
}
