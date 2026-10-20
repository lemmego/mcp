package mcp

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// defaultLogDir is where lemmego/log writes, and where a scaffolded project's
// LOG_FILE points.
const defaultLogDir = "storage/logs"

// maxLogScan bounds how much of a file is read.
//
// A log is append-only and unbounded, and the interesting end is the last one.
// Without a bound, "read the log" on a long-running application would try to
// load a gigabyte into a tool result.
const maxLogScan = 4096

// --- read-log-entries -------------------------------------------------------

type readLogInput struct {
	Lines    int    `json:"lines,omitempty" jsonschema:"how many of the most recent lines to return (default 50)"`
	Contains string `json:"contains,omitempty" jsonschema:"only return lines containing this text"`
	File     string `json:"file,omitempty" jsonschema:"a specific log file; omit for the most recently written one"`
}

type readLogOutput struct {
	File    string   `json:"file"`
	Entries []string `json:"entries"`
	Scanned int      `json:"lines_scanned"`
}

func (s *Server) addReadLogEntries(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name: "read-log-entries",
		Description: "The most recent lines from this application's log, newest last, " +
			"optionally filtered. Reads the most recently written file in storage/logs unless " +
			"told otherwise. Note that queued work is logged by the worker process, not the " +
			"web process, so a missing record may mean no worker is running.",
	}, s.readLogEntries)
}

func (s *Server) readLogEntries(_ context.Context, _ *sdk.CallToolRequest, in readLogInput) (*sdk.CallToolResult, readLogOutput, error) {
	path, err := s.resolveLogFile(in.File)
	if err != nil {
		return failure("%v", err), readLogOutput{}, nil
	}
	limit := in.Lines
	if limit <= 0 {
		limit = 50
	}

	lines, scanned, err := tailLines(path, limit, in.Contains)
	if err != nil {
		return failure("reading %s: %v", path, err), readLogOutput{}, nil
	}
	out := readLogOutput{File: path, Entries: lines, Scanned: scanned}

	var b strings.Builder
	fmt.Fprintf(&b, "%s — %d of the last %d lines", path, len(lines), scanned)
	if in.Contains != "" {
		fmt.Fprintf(&b, " matching %q", in.Contains)
	}
	b.WriteString("\n\n")
	for _, line := range lines {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return text(b.String()), out, nil
}

// --- last-error -------------------------------------------------------------

type lastErrorOutput struct {
	File  string `json:"file,omitempty"`
	Entry string `json:"entry,omitempty"`
	Found bool   `json:"found"`
}

func (s *Server) addLastError(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name: "last-error",
		Description: "The most recent error-level entry in this application's log. " +
			"Start a debugging loop here rather than reading the whole log: it is the one " +
			"line that says what actually broke.",
	}, s.lastError)
}

func (s *Server) lastError(_ context.Context, _ *sdk.CallToolRequest, _ emptyInput) (*sdk.CallToolResult, lastErrorOutput, error) {
	path, err := s.resolveLogFile("")
	if err != nil {
		return failure("%v", err), lastErrorOutput{}, nil
	}

	// Matched case-insensitively against the level as both the console and the
	// JSON handler write it: "ERROR" from the text handler, "level":"ERROR"
	// from the JSON one.
	lines, _, err := tailLines(path, maxLogScan, "")
	if err != nil {
		return failure("reading %s: %v", path, err), lastErrorOutput{}, nil
	}
	for i := len(lines) - 1; i >= 0; i-- {
		lower := strings.ToLower(lines[i])
		if strings.Contains(lower, "error") || strings.Contains(lower, "fatal") || strings.Contains(lower, "panic") {
			out := lastErrorOutput{File: path, Entry: lines[i], Found: true}
			return text(fmt.Sprintf("%s\n\n%s\n", path, lines[i])), out, nil
		}
	}
	return text(fmt.Sprintf("no error-level entry in the last %d lines of %s", len(lines), path)),
		lastErrorOutput{File: path}, nil
}

// --- helpers ----------------------------------------------------------------

// resolveLogFile picks the file to read.
//
// The most recently modified file rather than a fixed name, because
// lemmego/log rotates on size and on date, so the current file's name contains
// a date and is not predictable from configuration alone.
func (s *Server) resolveLogFile(requested string) (string, error) {
	dir := asString(s.app.Config().Get("log.dir"))
	if dir == "" {
		dir = defaultLogDir
	}

	if requested != "" {
		// Resolved within the log directory, and refused if it escapes:
		// otherwise "file" is an arbitrary-file-read tool wearing a log
		// reader's name.
		clean := filepath.Clean(filepath.Join(dir, filepath.Base(requested)))
		if _, err := os.Stat(clean); err != nil {
			return "", fmt.Errorf("no log file %q in %s", requested, dir)
		}
		return clean, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("no log directory at %s; this application may be logging to the console only", dir)
	}
	type candidate struct {
		path    string
		modTime int64
	}
	var candidates []candidate
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		candidates = append(candidates, candidate{filepath.Join(dir, entry.Name()), info.ModTime().UnixNano()})
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no log files in %s", dir)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].modTime > candidates[j].modTime })
	return candidates[0].path, nil
}

// tailLines returns the last limit lines of path that match contains, with the
// number of lines it looked at.
//
// The whole file is streamed and a ring of the last limit lines kept, rather
// than seeking backwards. Seeking is faster on a large file but has to handle
// a partial line at the seek point, and getting that wrong silently drops or
// splits the one entry somebody is looking for.
func tailLines(path string, limit int, contains string) ([]string, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	if limit > maxLogScan {
		limit = maxLogScan
	}
	ring := make([]string, 0, limit)
	scanned := 0

	scanner := bufio.NewScanner(file)
	// A JSON log line holds a stack trace and overruns the default 64KB.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if contains != "" && !strings.Contains(line, contains) {
			continue
		}
		scanned++
		if len(ring) == limit {
			ring = ring[1:]
		}
		ring = append(ring, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, scanned, err
	}
	return ring, scanned, nil
}
