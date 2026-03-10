package libapt

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SourceEntry represents a single APT source line (one-line or DEB822 format).
type SourceEntry struct {
	// Enabled is true when the line is active (not commented out).
	Enabled bool
	// Type is "deb" or "deb-src".
	Type string
	// Options holds bracketed options like [arch=amd64 signed-by=/path].
	Options string
	// URI is the repository URL.
	URI string
	// Suite is the distribution codename or path (e.g. "jammy", "stable").
	Suite string
	// Components lists the repository components (e.g. "main", "contrib").
	Components []string
	// FilePath is the file this entry was loaded from.
	FilePath string
	// LineNumber is the 1-based line number within the file (0 for DEB822).
	LineNumber int
	// RawLine is the original line text.
	RawLine string
	// IsDEB822 indicates this entry came from a .sources (DEB822) file.
	IsDEB822 bool
	// DEB822Block holds the raw text of the entire DEB822 stanza if applicable.
	DEB822Block string
	// DEB822Start is the starting line (1-based) of the stanza.
	DEB822Start int
	// DEB822End is the ending line (1-based) of the stanza.
	DEB822End int
}

// FormatOneLine returns the entry as a standard one-line sources.list line.
func (s *SourceEntry) FormatOneLine() string {
	var b strings.Builder
	if !s.Enabled {
		b.WriteString("# ")
	}
	b.WriteString(s.Type)
	if s.Options != "" {
		b.WriteString(" [")
		b.WriteString(s.Options)
		b.WriteString("]")
	}
	b.WriteString(" ")
	b.WriteString(s.URI)
	b.WriteString(" ")
	b.WriteString(s.Suite)
	if len(s.Components) > 0 {
		b.WriteString(" ")
		b.WriteString(strings.Join(s.Components, " "))
	}
	return b.String()
}

// DisplayString returns a concise human-readable representation.
func (s *SourceEntry) DisplayString() string {
	state := "●"
	if !s.Enabled {
		state = "○"
	}
	comps := ""
	if len(s.Components) > 0 {
		comps = " " + strings.Join(s.Components, " ")
	}
	opts := ""
	if s.Options != "" {
		opts = " [" + s.Options + "]"
	}
	return fmt.Sprintf("%s %s%s %s %s%s", state, s.Type, opts, s.URI, s.Suite, comps)
}

// sourcesDirs returns the directories to scan for source files.
func sourcesDirs() []string {
	return []string{
		"/etc/apt/sources.list",
		"/etc/apt/sources.list.d",
	}
}

// ListSources reads all APT source entries from the system.
func ListSources() ([]SourceEntry, error) {
	var entries []SourceEntry

	// 1. Parse /etc/apt/sources.list if it exists
	mainFile := "/etc/apt/sources.list"
	if info, err := os.Stat(mainFile); err == nil && !info.IsDir() {
		parsed, err := parseOneLineFile(mainFile)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", mainFile, err)
		}
		entries = append(entries, parsed...)
	}

	// 2. Parse /etc/apt/sources.list.d/*.list and *.sources
	listDir := "/etc/apt/sources.list.d"
	dirEntries, err := os.ReadDir(listDir)
	if err != nil {
		// directory might not exist; that's okay
		if os.IsNotExist(err) {
			return entries, nil
		}
		return entries, fmt.Errorf("reading %s: %w", listDir, err)
	}

	for _, de := range dirEntries {
		if de.IsDir() {
			continue
		}
		name := de.Name()
		fullPath := filepath.Join(listDir, name)

		if strings.HasSuffix(name, ".list") {
			parsed, err := parseOneLineFile(fullPath)
			if err != nil {
				continue // skip unreadable files
			}
			entries = append(entries, parsed...)
		} else if strings.HasSuffix(name, ".sources") {
			parsed, err := parseDEB822File(fullPath)
			if err != nil {
				continue
			}
			entries = append(entries, parsed...)
		}
	}

	return entries, nil
}

// parseOneLineFile reads a traditional sources.list file and returns entries.
func parseOneLineFile(path string) ([]SourceEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []SourceEntry
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)

		if trimmed == "" {
			continue
		}

		entry, ok := parseOneLineSrc(trimmed)
		if !ok {
			continue
		}
		entry.FilePath = path
		entry.LineNumber = lineNo
		entry.RawLine = raw
		entries = append(entries, entry)
	}

	return entries, scanner.Err()
}

// parseOneLineSrc parses a single one-line source entry (possibly commented).
func parseOneLineSrc(line string) (SourceEntry, bool) {
	var e SourceEntry
	work := line

	// Handle commented-out lines
	if strings.HasPrefix(work, "#") {
		work = strings.TrimSpace(strings.TrimPrefix(work, "#"))
		// might have multiple comment markers
		for strings.HasPrefix(work, "#") {
			work = strings.TrimSpace(strings.TrimPrefix(work, "#"))
		}
		e.Enabled = false
	} else {
		e.Enabled = true
	}

	// Must start with deb or deb-src
	if !strings.HasPrefix(work, "deb-src") && !strings.HasPrefix(work, "deb") {
		return e, false
	}

	// Extract type
	if strings.HasPrefix(work, "deb-src") {
		e.Type = "deb-src"
		work = strings.TrimSpace(work[7:])
	} else {
		e.Type = "deb"
		work = strings.TrimSpace(work[3:])
	}

	// Extract options in brackets [...]
	if strings.HasPrefix(work, "[") {
		end := strings.Index(work, "]")
		if end < 0 {
			return e, false
		}
		e.Options = strings.TrimSpace(work[1:end])
		work = strings.TrimSpace(work[end+1:])
	}

	fields := strings.Fields(work)
	if len(fields) < 2 {
		return e, false
	}

	e.URI = fields[0]
	e.Suite = fields[1]
	if len(fields) > 2 {
		e.Components = fields[2:]
	}

	return e, true
}

// parseDEB822File reads a DEB822-format .sources file and returns entries.
func parseDEB822File(path string) ([]SourceEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	var entries []SourceEntry
	var blockLines []string
	blockStart := 0

	flushBlock := func(endLine int) {
		if len(blockLines) == 0 {
			return
		}
		block := strings.Join(blockLines, "\n")
		entry := parseDEB822Block(block)
		entry.FilePath = path
		entry.IsDEB822 = true
		entry.DEB822Block = block
		entry.DEB822Start = blockStart
		entry.DEB822End = endLine
		if entry.Type != "" && entry.URI != "" {
			entries = append(entries, entry)
		}
		blockLines = nil
	}

	for i, line := range lines {
		lineNo := i + 1
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			flushBlock(lineNo - 1)
			blockStart = lineNo + 1
			continue
		}

		if len(blockLines) == 0 {
			blockStart = lineNo
		}
		blockLines = append(blockLines, line)
	}
	flushBlock(len(lines))

	return entries, nil
}

// parseDEB822Block parses a single DEB822 stanza into a SourceEntry.
func parseDEB822Block(block string) SourceEntry {
	var e SourceEntry
	e.Enabled = true

	fields := parseDEB822Fields(block)

	if v, ok := fields["Enabled"]; ok {
		e.Enabled = strings.ToLower(strings.TrimSpace(v)) != "no"
	}
	if v, ok := fields["Types"]; ok {
		e.Type = strings.TrimSpace(v)
	}
	if v, ok := fields["URIs"]; ok {
		e.URI = strings.TrimSpace(v)
	}
	if v, ok := fields["Suites"]; ok {
		e.Suite = strings.TrimSpace(v)
	}
	if v, ok := fields["Components"]; ok {
		comps := strings.Fields(strings.TrimSpace(v))
		if len(comps) > 0 {
			e.Components = comps
		}
	}

	// Collect options from known option fields
	var opts []string
	if v, ok := fields["Architectures"]; ok {
		opts = append(opts, "arch="+strings.TrimSpace(v))
	}
	if v, ok := fields["Signed-By"]; ok {
		opts = append(opts, "signed-by="+strings.TrimSpace(v))
	}
	if len(opts) > 0 {
		e.Options = strings.Join(opts, " ")
	}

	return e
}

// parseDEB822Fields parses key-value pairs from a DEB822 stanza.
func parseDEB822Fields(block string) map[string]string {
	fields := make(map[string]string)
	var currentKey string
	for _, line := range strings.Split(block, "\n") {
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			// Continuation line
			if currentKey != "" {
				fields[currentKey] += "\n" + line
			}
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		currentKey = strings.TrimSpace(line[:idx])
		fields[currentKey] = strings.TrimSpace(line[idx+1:])
	}
	return fields
}

// AddSource adds a new source entry to a file. If filePath is empty, it
// defaults to /etc/apt/sources.list.d/aether-custom.list.
func AddSource(entry SourceEntry) error {
	path := entry.FilePath
	if path == "" {
		path = "/etc/apt/sources.list.d/aether-custom.list"
	}

	// Make sure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	line := entry.FormatOneLine()

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	// Add a trailing newline if the file doesn't end with one
	info, _ := f.Stat()
	if info != nil && info.Size() > 0 {
		// Read last byte
		buf := make([]byte, 1)
		rf, rerr := os.Open(path)
		if rerr == nil {
			rf.Seek(-1, 2)
			rf.Read(buf)
			rf.Close()
			if buf[0] != '\n' {
				f.WriteString("\n")
			}
		}
	}

	_, err = f.WriteString(line + "\n")
	if err != nil {
		return fmt.Errorf("writing to %s: %w", path, err)
	}
	return nil
}

// EditSource replaces the source entry at the recorded file/line position
// with the new entry data.
func EditSource(old SourceEntry, updated SourceEntry) error {
	if old.IsDEB822 {
		return editDEB822Source(old, updated)
	}
	return editOneLineSource(old, updated)
}

// editOneLineSource replaces a single line in a .list file.
func editOneLineSource(old SourceEntry, updated SourceEntry) error {
	path := old.FilePath
	if path == "" {
		return fmt.Errorf("no file path for source entry")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	lines := strings.Split(string(data), "\n")
	lineIdx := old.LineNumber - 1
	if lineIdx < 0 || lineIdx >= len(lines) {
		return fmt.Errorf("line %d out of range in %s", old.LineNumber, path)
	}

	lines[lineIdx] = updated.FormatOneLine()

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

// editDEB822Source replaces a DEB822 stanza with an updated one-line version
// by commenting out the old block and appending a new .list file.
// This is conservative — DEB822 format is complex, so we convert to one-line.
func editDEB822Source(old SourceEntry, updated SourceEntry) error {
	path := old.FilePath

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	lines := strings.Split(string(data), "\n")

	// Comment out the old stanza lines
	start := old.DEB822Start - 1
	end := old.DEB822End - 1
	if start < 0 {
		start = 0
	}
	if end >= len(lines) {
		end = len(lines) - 1
	}

	for i := start; i <= end; i++ {
		if !strings.HasPrefix(lines[i], "#") {
			lines[i] = "# " + lines[i]
		}
	}

	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	// Append the updated entry as a one-line entry to a .list file
	updated.FilePath = ""
	return AddSource(updated)
}

// DeleteSource removes the source entry from its file.
func DeleteSource(entry SourceEntry) error {
	if entry.IsDEB822 {
		return deleteDEB822Source(entry)
	}
	return deleteOneLineSource(entry)
}

// deleteOneLineSource removes or comments out the line in a .list file.
func deleteOneLineSource(entry SourceEntry) error {
	path := entry.FilePath
	if path == "" {
		return fmt.Errorf("no file path for source entry")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	lines := strings.Split(string(data), "\n")
	lineIdx := entry.LineNumber - 1
	if lineIdx < 0 || lineIdx >= len(lines) {
		return fmt.Errorf("line %d out of range in %s", entry.LineNumber, path)
	}

	// Remove the line entirely
	lines = append(lines[:lineIdx], lines[lineIdx+1:]...)

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

// deleteDEB822Source comments out the entire stanza in a .sources file.
func deleteDEB822Source(entry SourceEntry) error {
	path := entry.FilePath

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	lines := strings.Split(string(data), "\n")
	start := entry.DEB822Start - 1
	end := entry.DEB822End - 1
	if start < 0 {
		start = 0
	}
	if end >= len(lines) {
		end = len(lines) - 1
	}

	for i := start; i <= end; i++ {
		if !strings.HasPrefix(lines[i], "#") && strings.TrimSpace(lines[i]) != "" {
			lines[i] = "# " + lines[i]
		}
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

// ToggleSource enables or disables a source entry in place.
func ToggleSource(entry SourceEntry) error {
	toggled := entry
	toggled.Enabled = !entry.Enabled

	if entry.IsDEB822 {
		return toggleDEB822Source(entry)
	}
	return EditSource(entry, toggled)
}

// toggleDEB822Source toggles the Enabled field in a DEB822 stanza.
func toggleDEB822Source(entry SourceEntry) error {
	path := entry.FilePath

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	lines := strings.Split(string(data), "\n")
	start := entry.DEB822Start - 1
	end := entry.DEB822End - 1
	if start < 0 {
		start = 0
	}
	if end >= len(lines) {
		end = len(lines) - 1
	}

	newEnabled := !entry.Enabled

	// Look for existing Enabled: line and update it
	foundEnabled := false
	for i := start; i <= end; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "Enabled:") {
			if newEnabled {
				lines[i] = "Enabled: yes"
			} else {
				lines[i] = "Enabled: no"
			}
			foundEnabled = true
			break
		}
	}

	// If no Enabled: line exists, insert one after the Types: line
	if !foundEnabled {
		val := "yes"
		if !newEnabled {
			val = "no"
		}
		for i := start; i <= end; i++ {
			if strings.HasPrefix(strings.TrimSpace(lines[i]), "Types:") {
				enabledLine := "Enabled: " + val
				// Insert after this line
				newLines := make([]string, 0, len(lines)+1)
				newLines = append(newLines, lines[:i+1]...)
				newLines = append(newLines, enabledLine)
				newLines = append(newLines, lines[i+1:]...)
				lines = newLines
				break
			}
		}
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

// ListSourceFiles returns the paths of all source files found on the system.
func ListSourceFiles() ([]string, error) {
	var files []string

	mainFile := "/etc/apt/sources.list"
	if info, err := os.Stat(mainFile); err == nil && !info.IsDir() {
		files = append(files, mainFile)
	}

	listDir := "/etc/apt/sources.list.d"
	dirEntries, err := os.ReadDir(listDir)
	if err != nil {
		if os.IsNotExist(err) {
			return files, nil
		}
		return files, err
	}

	for _, de := range dirEntries {
		if de.IsDir() {
			continue
		}
		name := de.Name()
		if strings.HasSuffix(name, ".list") || strings.HasSuffix(name, ".sources") {
			files = append(files, filepath.Join(listDir, name))
		}
	}

	return files, nil
}

// ReadSourceFile returns the raw content of a source file.
func ReadSourceFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteSourceFile writes content to a source file.
func WriteSourceFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
