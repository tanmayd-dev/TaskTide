package tasks

import (
	"encoding/csv"
	"fmt"
	"log"
	"strings"

	"github.com/tanmayd-dev/TaskTide/internal/models"
)

// CSVFilter removes empty rows and '#' comment lines from the input CSV and
// writes a clean CSV (joined with commas) as its output.
func CSVFilter(task *models.WorkerTaskExecution) error {
	log.Printf("[CSVFilter] Filtering CSV task execution: %s\n", task.ID)

	reader := csv.NewReader(strings.NewReader(task.TaskInput))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		log.Printf("[CSVFilter] Error parsing CSV: %v\n", err)
		return fmt.Errorf("failed to parse CSV: %w", err)
	}

	kept := make([][]string, 0, len(records))
	removedEmpty, removedComments := 0, 0
	for _, record := range records {
		if len(record) == 1 && strings.TrimSpace(record[0]) == "" {
			removedEmpty++
			continue
		}
		if len(record) > 0 && strings.HasPrefix(strings.TrimSpace(record[0]), "#") {
			removedComments++
			continue
		}
		kept = append(kept, record)
	}

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("# csv.filter: removed %d empty row(s), %d comment row(s); %d row(s) kept\n",
		removedEmpty, removedComments, len(kept)))
	for _, record := range kept {
		buf.WriteString(strings.Join(record, ","))
		buf.WriteString("\n")
	}

	task.TaskOutput = buf.String()
	return nil
}
