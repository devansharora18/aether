package tui

import (
	"strings"

	"github.com/rivo/tview"
)

func centered(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 1, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)
}

type packageResult struct {
	name        string
	repo        string
	version     string
	arch        string
	description string
}

func formatSearchResults(query, raw string) string {
	// kept for backward compatibility with the CLI search; not used by TUI
	res := parseSearchResults(raw)
	if len(res) == 0 {
		return "No results"
	}
	var b strings.Builder
	for _, r := range res {
		b.WriteString(r.name + " - " + r.description + "\n")
	}
	return b.String()
}

func parseSearchResults(raw string) []packageResult {
	lines := strings.Split(raw, "\n")
	out := make([]packageResult, 0)
	current := -1

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "WARNING:") ||
			trimmed == "Sorting..." ||
			trimmed == "Full Text Search..." ||
			strings.HasPrefix(trimmed, "N: ") {
			continue
		}

		if strings.HasPrefix(line, "  ") {
			if current >= 0 {
				if out[current].description == "" {
					out[current].description = trimmed
				} else {
					out[current].description += " " + trimmed
				}
			}
			continue
		}

		fields := strings.Fields(trimmed)
		if len(fields) < 2 || !strings.Contains(fields[0], "/") {
			continue
		}

		nameRepo := strings.SplitN(fields[0], "/", 2)
		repo := ""
		if len(nameRepo) == 2 {
			repo = nameRepo[1]
			if comma := strings.Index(repo, ","); comma != -1 {
				repo = repo[:comma]
			}
		}

		version := fields[1]
		arch := ""
		if len(fields) > 2 {
			arch = fields[2]
		}

		out = append(out, packageResult{
			name:    nameRepo[0],
			repo:    repo,
			version: version,
			arch:    arch,
		})
		current = len(out) - 1
	}

	return out
}
