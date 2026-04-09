package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kostya/peyote-designer/internal/model"
)

type Store struct{}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) Load(path string) (*model.Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var doc model.Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("decode peyote file: %w", err)
	}
	if err := doc.Validate(); err != nil {
		return nil, fmt.Errorf("validate peyote file: %w", err)
	}
	return &doc, nil
}

func (s *Store) Save(path string, document *model.Document) error {
	if filepath.Ext(path) != ".pey" {
		path += ".pey"
	}
	document.Touch()
	document.Metadata.AppName = model.AppName

	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("encode peyote file: %w", err)
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && !strings.EqualFold(filepath.Dir(path), ".") {
		return fmt.Errorf("ensure directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write peyote file: %w", err)
	}
	return nil
}
