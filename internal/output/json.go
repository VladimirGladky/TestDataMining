package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"TestDataMining/internal/lenta"
)

type Report struct {
	GeneratedAt string                 `json:"generated_at"`
	Source      string                 `json:"source"`
	Categories  []lenta.CategoryResult `json:"categories"`
	TotalCount  int                    `json:"total_count"`
}

func WriteJSON(path string, categories []lenta.CategoryResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	total := 0
	for _, c := range categories {
		total += len(c.Products)
	}

	report := Report{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Source:      "lenta.com",
		Categories:  categories,
		TotalCount:  total,
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
