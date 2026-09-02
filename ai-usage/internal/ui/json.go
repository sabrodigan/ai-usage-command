package ui

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"

	"ai-usage/internal/core"
)

// RenderJSON writes the indented JSON payload to writer.
func RenderJSON(w io.Writer, snapshot *core.UsageSnapshot) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(snapshot)
}

// RenderCSV writes a CSV export to writer.
func RenderCSV(w io.Writer, snapshot *core.UsageSnapshot) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	// Write header
	header := []string{"Rank", "Provider ID", "Display Name", "Model/Tier", "Consumed", "Unit", "Quota", "Remaining", "Percent Used", "Cost USD", "Status", "Error"}
	if err := cw.Write(header); err != nil {
		return err
	}

	for i, p := range snapshot.Providers {
		record := []string{
			fmt.Sprintf("%d", i+1),
			p.ProviderID,
			p.DisplayName,
			p.ModelOrTier,
			fmt.Sprintf("%.2f", p.Consumed),
			string(p.Unit),
			fmt.Sprintf("%.2f", p.Quota),
			fmt.Sprintf("%.2f", p.Remaining),
			fmt.Sprintf("%.2f", p.PercentUsed),
			fmt.Sprintf("%.2f", p.EstimatedCost),
			p.Status,
			p.ErrorMessage,
		}
		if err := cw.Write(record); err != nil {
			return err
		}
	}

	return nil
}
