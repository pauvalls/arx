package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ResolveIncludes processes YAML data and replaces !include tags with the content
// of the referenced files. Includes are resolved recursively, with cycle detection.
// The root parameter is the directory used to resolve relative include paths.
// Cycle detection uses a caller-provided seen set to track active resolution chains.
func ResolveIncludes(root string, data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing YAML for includes: %w", err)
	}

	seen := make(map[string]bool)
	if err := resolveNode(&doc, root, seen); err != nil {
		return nil, err
	}

	result, err := yaml.Marshal(&doc)
	if err != nil {
		return nil, fmt.Errorf("serializing YAML after includes: %w", err)
	}

	return result, nil
}

// resolveNode walks a yaml.Node tree and resolves any !include tags.
func resolveNode(node *yaml.Node, root string, seen map[string]bool) error {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case yaml.ScalarNode:
		if node.Tag == "!include" {
			return resolveInclude(node, root, seen)
		}
		return nil

	case yaml.SequenceNode:
		for i := range node.Content {
			if err := resolveNode(node.Content[i], root, seen); err != nil {
				return err
			}
		}

	case yaml.MappingNode:
		// Content is alternating key-value pairs
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			val := node.Content[i+1]

			// Check if the VALUE has !include tag
			if val.Tag == "!include" {
				if err := resolveInclude(val, root, seen); err != nil {
					return err
				}
				continue
			}

			// Recurse into value
			if err := resolveNode(val, root, seen); err != nil {
				return err
			}

			// Also recurse into key (rare but possible)
			if err := resolveNode(key, root, seen); err != nil {
				return err
			}
		}

	case yaml.DocumentNode:
		for i := range node.Content {
			if err := resolveNode(node.Content[i], root, seen); err != nil {
				return err
			}
		}

	case yaml.AliasNode:
		// Alias nodes reference other nodes — nothing to resolve
		return nil
	}

	return nil
}

// resolveInclude reads the file referenced by a !include node and replaces
// the node with the parsed YAML content of that file.
func resolveInclude(node *yaml.Node, root string, seen map[string]bool) error {
	includePath := node.Value
	if includePath == "" {
		return fmt.Errorf("!include tag with empty path")
	}

	// Resolve relative to root
	absPath := includePath
	if !filepath.IsAbs(includePath) {
		absPath = filepath.Join(root, includePath)
	}
	absPath = filepath.Clean(absPath)

	// Check for cycles
	if seen[absPath] {
		return fmt.Errorf("circular include detected: %s", absPath)
	}
	seen[absPath] = true
	defer delete(seen, absPath)

	// Read the included file
	includeData, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("reading include %q: %w", includePath, err)
	}

	// Resolve any nested includes first (recursive).
	// We parse the included file as a new YAML node tree and walk it with
	// the shared seen map for cycle detection.
	var includeDoc yaml.Node
	if err := yaml.Unmarshal(includeData, &includeDoc); err != nil {
		return fmt.Errorf("parsing include %q: %w", includePath, err)
	}

	// Walk the included document's tree to resolve nested includes
	if err := resolveNode(&includeDoc, filepath.Dir(absPath), seen); err != nil {
		return fmt.Errorf("resolving nested includes in %q: %w", includePath, err)
	}

	// Re-serialize the resolved include content to get clean YAML
	resolved, err := yaml.Marshal(&includeDoc)
	if err != nil {
		return fmt.Errorf("serializing resolved include %q: %w", includePath, err)
	}

	// Parse the resolved content and replace the !include node
	// Parse again to get the concrete content node (not the document wrapper)
	var contentDoc yaml.Node
	if err := yaml.Unmarshal(resolved, &contentDoc); err != nil {
		return fmt.Errorf("parsing resolved include %q: %w", includePath, err)
	}

	// Replace the !include node with the content of the included file
	if contentDoc.Kind == yaml.DocumentNode && len(contentDoc.Content) == 1 {
		included := contentDoc.Content[0]
		node.Kind = included.Kind
		node.Tag = included.Tag
		node.Value = included.Value
		node.Content = included.Content
		node.Alias = included.Alias
	} else {
		// Fallback: use resolved bytes directly as a string scalar
		node.Kind = yaml.ScalarNode
		node.Tag = "!!str"
		node.Value = string(resolved)
		node.Content = nil
	}

	return nil
}
