package internal

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Document struct {
	Path    string
	Content string
}

func ensureVault(path string) error {
	return os.MkdirAll(path, 0o755)
}

func notePath(vault, note string) string {
	if strings.TrimSpace(note) == "" {
		note = "untitled.md"
	}
	if !strings.HasSuffix(strings.ToLower(note), ".md") {
		note += ".md"
	}
	return filepath.Join(vault, note)
}

func loadDocument(vault, note string) (Document, error) {
	if err := ensureVault(vault); err != nil {
		return Document{}, err
	}
	path := notePath(vault, note)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			initial := "# Untitled\n\n"
			if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
				return Document{}, err
			}
			return Document{Path: path, Content: initial}, nil
		}
		return Document{}, err
	}
	return Document{Path: path, Content: string(data)}, nil
}

func createTodayNote(vault string) (Document, error) {
	name := time.Now().Format("2006-01-02") + ".md"
	return loadDocument(vault, name)
}

func saveDocument(doc Document) error {
	return os.WriteFile(doc.Path, []byte(doc.Content), 0o644)
}
