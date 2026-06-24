package tasks

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"log"
	"strings"

	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func CSVRead(task *models.WorkerTaskExecution) error {
	log.Printf("[CSVRead] Reading CSV document task execution: %s\n", task.ID)

	reader := csv.NewReader(strings.NewReader(task.TaskInput))
	records, err := reader.ReadAll()
	if err != nil {
		log.Printf("[CSVRead] Error parsing CSV: %v\n", err)
		return fmt.Errorf("failed to parse CSV: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("CSV parsed successfully. Total Rows: %d\n\n", len(records)))
	for i, record := range records {
		buf.WriteString(fmt.Sprintf("Row %d: %s\n", i+1, strings.Join(record, " | ")))
	}

	task.TaskOutput = buf.String()
	return nil
}
