// Package ksm provides a wrapper around the Keeper Secrets Manager Go SDK.
package ksm

import (
	"fmt"
	"strings"

	ksm "github.com/keeper-security/secrets-manager-go/core"
)

// FolderTree represents the hierarchical structure of Keeper folders
type FolderTree struct {
	folders map[string]*FolderNode // UID -> Node mapping for fast lookup
	root    []*FolderNode           // Top-level folders (no parent)
}

// FolderNode represents a single folder in the hierarchy
type FolderNode struct {
	UID      string        // Folder UID
	Name     string        // Folder name
	Parent   *FolderNode   // Parent folder (nil for root folders)
	Children []*FolderNode // Child folders
}

// BuildFolderTree creates a hierarchical folder tree from a flat list of folders
func BuildFolderTree(folders []*ksm.KeeperFolder) *FolderTree {
	tree := &FolderTree{
		folders: make(map[string]*FolderNode),
		root:    []*FolderNode{},
	}

	// First pass: create all nodes
	for _, f := range folders {
		node := &FolderNode{
			UID:      f.FolderUid,
			Name:     f.Name,
			Children: []*FolderNode{},
		}
		tree.folders[f.FolderUid] = node
	}

	// Second pass: build parent-child relationships
	for _, f := range folders {
		node := tree.folders[f.FolderUid]
		if f.ParentUid == "" {
			// Root folder (no parent)
			tree.root = append(tree.root, node)
		} else if parent, ok := tree.folders[f.ParentUid]; ok {
			// Set parent relationship
			node.Parent = parent
			parent.Children = append(parent.Children, node)
		} else {
			// Parent not found - treat as root
			tree.root = append(tree.root, node)
		}
	}

	return tree
}

// ResolvePath converts a folder path (e.g., "Production/Databases") to a folder UID
// Returns the folder UID if found, or an error if the path doesn't exist
func (t *FolderTree) ResolvePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("folder path cannot be empty")
	}

	// Trim leading and trailing slashes and split into components
	path = strings.Trim(path, "/")
	rawParts := strings.Split(path, "/")

	// Filter out empty components
	var parts []string
	for _, p := range rawParts {
		if p != "" {
			parts = append(parts, p)
		}
	}

	if len(parts) == 0 {
		return "", fmt.Errorf("invalid folder path: %s", path)
	}

	// Start from root folders
	currentNodes := t.root

	// Traverse path components
	for i, part := range parts {

		// Find matching folder at this level
		var found *FolderNode
		for _, node := range currentNodes {
			if node.Name == part {
				found = node
				break
			}
		}

		if found == nil {
			// Build the path we were trying to find for error message
			attemptedPath := strings.Join(parts[:i+1], "/")
			return "", fmt.Errorf("folder not found at path '%s' (searching for '%s')", attemptedPath, part)
		}

		// Move to children for next iteration
		if i == len(parts)-1 {
			// This is the final component - return its UID
			return found.UID, nil
		}

		// Continue with children of this folder
		currentNodes = found.Children
	}

	return "", fmt.Errorf("folder path resolution failed: %s", path)
}

// GetPath returns the full path of a folder given its UID
// Returns empty string if UID not found
func (t *FolderTree) GetPath(folderUID string) string {
	node, ok := t.folders[folderUID]
	if !ok {
		return ""
	}

	// Build path by walking up to root
	var pathParts []string
	current := node
	for current != nil {
		pathParts = append([]string{current.Name}, pathParts...) // Prepend
		current = current.Parent
	}

	return strings.Join(pathParts, "/")
}

// GetNode returns the FolderNode for a given UID
func (t *FolderTree) GetNode(folderUID string) *FolderNode {
	return t.folders[folderUID]
}

// ListPaths returns all folder paths in the tree
func (t *FolderTree) ListPaths() []string {
	var paths []string
	for uid := range t.folders {
		path := t.GetPath(uid)
		if path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}
