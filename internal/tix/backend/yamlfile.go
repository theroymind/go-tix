package backend

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"go.yaml.in/yaml/v3"
)

// ParseYAML parses a single YAML document, rejecting duplicate keys, and names path in its errors.
func ParseYAML(text []byte, path string) (*yaml.Node, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(text))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil && !errors.Is(err, io.EOF) {
		return nil, invalidYAML(path, err)
	}
	var next yaml.Node
	if err := decoder.Decode(&next); !errors.Is(err, io.EOF) {
		if err == nil {
			err = ErrMultipleDocuments
		}
		return nil, invalidYAML(path, err)
	}
	if err := checkDuplicateKeys(&document); err != nil {
		return nil, invalidYAML(path, err)
	}
	return &document, nil
}

// checkDuplicateKeys exists because the decoder only reports duplicate keys when decoding into a
// map or struct, and a document is parsed into a node first.
func checkDuplicateKeys(node *yaml.Node) error {
	if node.Kind == yaml.MappingNode {
		lines := map[string]int{}
		for index := 0; index+1 < len(node.Content); index += 2 {
			key := node.Content[index]
			if firstLine, exists := lines[key.Value]; exists && key.Kind == yaml.ScalarNode {
				return fmt.Errorf("line %d: %w %q, first defined at line %d", key.Line, ErrDuplicateKey, key.Value, firstLine)
			}
			lines[key.Value] = key.Line
		}
	}
	for _, child := range node.Content {
		if err := checkDuplicateKeys(child); err != nil {
			return err
		}
	}
	return nil
}

// DecodeYAML decodes a parsed document into target, reporting a type mismatch as an invalid file at path.
func DecodeYAML(document *yaml.Node, target any, path string) error {
	if document.Kind == 0 {
		return nil
	}
	err := document.Decode(target)
	if typeError := (*yaml.TypeError)(nil); errors.As(err, &typeError) {
		return InvalidFile(path, err)
	}
	return err
}

// EncodeYAML writes value as YAML with the two-space indent tix v1 files use.
func EncodeYAML(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(yamlIndent)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
