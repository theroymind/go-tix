package yamlstore

import "go.yaml.in/yaml/v3"

func mappingRoot(document *yaml.Node) *yaml.Node {
	if document.Kind != yaml.DocumentNode || len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return nil
	}
	return document.Content[0]
}

func hasKey(mapping *yaml.Node) func(key string) bool {
	return func(key string) bool {
		return valueIndex(mapping, key) >= 0
	}
}

func valueIndex(mapping *yaml.Node, key string) int {
	if mapping == nil {
		return -1
	}
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			return index + 1
		}
	}
	return -1
}
