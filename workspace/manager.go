package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Manager handles reading and writing files in the project workspace.
type Manager struct {
	Root string
}

// New creates a workspace Manager for the given project directory.
func New(projectPath string) (*Manager, error) {
	abs, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, fmt.Errorf("invalid project path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			if mkErr := os.MkdirAll(abs, 0o755); mkErr != nil {
				return nil, fmt.Errorf("failed to create project path: %s", abs)
			}
		} else {
			return nil, fmt.Errorf("project path error: %s", abs)
		}
	} else if !info.IsDir() {
		return nil, fmt.Errorf("project path is not a directory: %s", abs)
	}
	return &Manager{Root: abs}, nil
}

// ReadFile reads a file from the workspace.
func (m *Manager) ReadFile(relPath string) (string, error) {
	full := filepath.Join(m.Root, relPath)
	data, err := os.ReadFile(full)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", relPath, err)
	}
	return string(data), nil
}

// WriteFile writes content to a file in the workspace, creating
// parent directories as needed.
func (m *Manager) WriteFile(relPath string, content string) error {
	full := filepath.Join(m.Root, relPath)
	dir := filepath.Dir(full)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	return os.WriteFile(full, []byte(content), 0o644)
}

// ListFiles returns all files in the workspace (relative paths).
// Skips hidden directories (starting with .).
func (m *Manager) ListFiles() ([]string, error) {
	var files []string
	err := filepath.Walk(m.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}
		if !info.IsDir() {
			rel, _ := filepath.Rel(m.Root, path)
			files = append(files, rel)
		}
		return nil
	})
	return files, err
}

// GetSummary returns a formatted tree of files in the workspace.
func (m *Manager) GetSummary() (string, error) {
	files, err := m.ListFiles()
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Project: %s\n", m.Root))
	sb.WriteString(fmt.Sprintf("Files (%d):\n", len(files)))
	for _, f := range files {
		sb.WriteString(fmt.Sprintf("  %s\n", f))
	}
	return sb.String(), nil
}
