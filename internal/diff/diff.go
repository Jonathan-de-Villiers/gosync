package diff

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type DiffEngine struct {
	diffCommand string
}

type DiffResult struct {
	HasChanges bool
	Output     string
	Error      error
}

func New(diffCommand string) *DiffEngine {
	if diffCommand == "" {
		diffCommand = "diff -u"
	}
	return &DiffEngine{
		diffCommand: diffCommand,
	}
}

func (d *DiffEngine) HasChanges(file1, file2 string) bool {
	result := d.runDiff(file1, file2, true)
	return result.HasChanges
}

func (d *DiffEngine) ShowDiff(file1, file2 string) error {
	result := d.runDiff(file1, file2, false)
	if result.Error != nil {
		return result.Error
	}

	if result.Output != "" {
		fmt.Println(result.Output)
	}

	return nil
}

func (d *DiffEngine) GetDiff(file1, file2 string) (DiffResult, error) {
	return d.runDiff(file1, file2, false), nil
}

func (d *DiffEngine) runDiff(file1, file2 string, quiet bool) DiffResult {
	result := DiffResult{}

	// Split command and arguments
	parts := strings.Fields(d.diffCommand)
	if len(parts) == 0 {
		result.Error = fmt.Errorf("no diff command specified")
		return result
	}

	cmd := exec.Command(parts[0], append(parts[1:], file1, file2)...)
	
	if quiet {
		// Use -q flag if available to just check for differences
		if !contains(parts, "-q") {
			cmd.Args = append([]string{parts[0], "-q"}, append(parts[1:], file1, file2)...)
		}
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		// diff returns exit code 1 when files differ, which is not an error for us
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			result.HasChanges = true
			result.Output = string(output)
			return result
		}
		result.Error = fmt.Errorf("diff command failed: %w", err)
		return result
	}

	if len(output) > 0 {
		result.HasChanges = true
		result.Output = string(output)
	}

	return result
}

func (d *DiffEngine) ShowFileSummary(file1, file2 string) error {
	info1, err1 := os.Stat(file1)
	info2, err2 := os.Stat(file2)

	fmt.Printf("File comparison:\n")
	
	if err1 == nil {
		fmt.Printf("  File 1: %s (modified: %s, size: %d bytes)\n", 
			filepath.Base(file1), 
			info1.ModTime().Format("2006-01-02 15:04:05"), 
			info1.Size())
	} else {
		fmt.Printf("  File 1: %s (does not exist)\n", filepath.Base(file1))
	}

	if err2 == nil {
		fmt.Printf("  File 2: %s (modified: %s, size: %d bytes)\n", 
			filepath.Base(file2), 
			info2.ModTime().Format("2006-01-02 15:04:05"), 
			info2.Size())
	} else {
		fmt.Printf("  File 2: %s (does not exist)\n", filepath.Base(file2))
	}

	return nil
}

func (d *DiffEngine) ShowUnifiedDiff(file1, file2 string, contextLines int) error {
	// Create a custom unified diff with context
	f1, err := os.Open(file1)
	if err != nil {
		return fmt.Errorf("failed to open file1: %w", err)
	}
	defer f1.Close()

	f2, err := os.Open(file2)
	if err != nil {
		return fmt.Errorf("failed to open file2: %w", err)
	}
	defer f2.Close()

	lines1, err := d.readLines(f1)
	if err != nil {
		return fmt.Errorf("failed to read file1: %w", err)
	}

	lines2, err := d.readLines(f2)
	if err != nil {
		return fmt.Errorf("failed to read file2: %w", err)
	}

	diff := d.unifiedDiff(lines1, lines2, filepath.Base(file1), filepath.Base(file2), contextLines)
	fmt.Print(diff)

	return nil
}

func (d *DiffEngine) readLines(file *os.File) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func (d *DiffEngine) unifiedDiff(lines1, lines2 []string, file1, file2 string, context int) string {
	// Simple unified diff implementation
	var result strings.Builder
	
	result.WriteString(fmt.Sprintf("--- %s\n", file1))
	result.WriteString(fmt.Sprintf("+++ %s\n", file2))

	// Find differences using a simple algorithm
	diffBlocks := d.findDiffBlocks(lines1, lines2, context)
	
	for _, block := range diffBlocks {
		if block.Type == "context" {
			for _, line := range block.Lines {
				result.WriteString(fmt.Sprintf(" %s\n", line))
			}
		} else if block.Type == "removed" {
			for _, line := range block.Lines {
				result.WriteString(fmt.Sprintf("-%s\n", line))
			}
		} else if block.Type == "added" {
			for _, line := range block.Lines {
				result.WriteString(fmt.Sprintf("+%s\n", line))
			}
		}
	}

	return result.String()
}

type DiffBlock struct {
	Type  string // "context", "added", "removed"
	Lines []string
}

func (d *DiffEngine) findDiffBlocks(lines1, lines2 []string, context int) []DiffBlock {
	var blocks []DiffBlock
	i, j := 0, 0
	len1, len2 := len(lines1), len(lines2)

	for i < len1 || j < len2 {
		// Find next matching line
		matchI, matchJ := -1, -1
		
		// Look ahead for matches
		for di := 0; di < len1-i; di++ {
			for dj := 0; dj < len2-j; dj++ {
				if i+di < len1 && j+dj < len2 && lines1[i+di] == lines2[j+dj] {
					matchI, matchJ = i+di, j+dj
					break
				}
			}
			if matchI != -1 {
				break
			}
		}

		if matchI == -1 && matchJ == -1 {
			// No more matches, process remaining lines
			if i < len1 {
				// Remaining lines are removed
				block := DiffBlock{Type: "removed", Lines: lines1[i:]}
				blocks = append(blocks, block)
			}
			if j < len2 {
				// Remaining lines are added
				block := DiffBlock{Type: "added", Lines: lines2[j:]}
				blocks = append(blocks, block)
			}
			break
		}

		// Add context before changes
		if matchI > i || matchJ > j {
			contextStart := max(0, i-context)
			contextEnd := min(len1, matchI)
			
			if contextEnd > contextStart {
				block := DiffBlock{Type: "context", Lines: lines1[contextStart:contextEnd]}
				blocks = append(blocks, block)
			}

			// Add removed lines
			if matchI > i {
				block := DiffBlock{Type: "removed", Lines: lines1[i:matchI]}
				blocks = append(blocks, block)
			}

			// Add added lines
			if matchJ > j {
				block := DiffBlock{Type: "added", Lines: lines2[j:matchJ]}
				blocks = append(blocks, block)
			}
		}

		// Move to match position
		i, j = matchI, matchJ
	}

	return blocks
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
