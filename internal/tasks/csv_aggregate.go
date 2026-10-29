package tasks

import (
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/tanmayd-dev/TaskTide/internal/models"
)

// CSVAggregate computes summary statistics over the input CSV: row/column
// counts, per-column non-empty and distinct value counts, and (for columns
// whose non-empty values are all numeric) min/max/mean.
func CSVAggregate(task *models.WorkerTaskExecution) error {
	log.Printf("[CSVAggregate] Aggregating CSV task execution: %s\n", task.ID)

	reader := csv.NewReader(strings.NewReader(task.TaskInput))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		log.Printf("[CSVAggregate] Error parsing CSV: %v\n", err)
		return fmt.Errorf("failed to parse CSV: %w", err)
	}
	if len(records) == 0 {
		task.TaskOutput = "csv.aggregate: no records found (empty input)"
		return nil
	}

	columns := len(records[0])
	dataRows := records[1:] // treat the first record as a header
	if len(dataRows) == 0 {
		dataRows = records
	}

	type columnStats struct {
		nonEmpty   int
		distinct   map[string]struct{}
		numeric    bool
		sum        float64
		min, max   float64
		numericCnt int
	}
	stats := make([]*columnStats, columns)
	for i := range stats {
		stats[i] = &columnStats{
			distinct: make(map[string]struct{}),
			numeric:  true,
			min:      math.Inf(1),
			max:      math.Inf(-1),
		}
	}

	for _, row := range dataRows {
		for col := 0; col < columns && col < len(row); col++ {
			value := strings.TrimSpace(row[col])
			if value == "" {
				continue
			}
			s := stats[col]
			s.nonEmpty++
			s.distinct[value] = struct{}{}
			if s.numeric {
				if f, err := strconv.ParseFloat(value, 64); err == nil {
					s.numericCnt++
					s.sum += f
					if f < s.min {
						s.min = f
					}
					if f > s.max {
						s.max = f
					}
				} else {
					s.numeric = false
				}
			}
		}
	}

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("csv.aggregate: %d data row(s), %d column(s)\n", len(dataRows), columns))
	for col, s := range stats {
		var header string
		if col < len(records[0]) {
			header = strings.TrimSpace(records[0][col])
		}
		if header == "" {
			header = fmt.Sprintf("column_%d", col+1)
		}
		line := fmt.Sprintf("  %-16s non-empty: %d, distinct: %d", header, s.nonEmpty, len(s.distinct))
		if s.numeric && s.numericCnt > 0 {
			line += fmt.Sprintf(", min: %g, max: %g, mean: %g", s.min, s.max, s.sum/float64(s.numericCnt))
		}
		buf.WriteString(line + "\n")
	}

	task.TaskOutput = buf.String()
	return nil
}
