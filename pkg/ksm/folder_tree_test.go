package ksm

import (
	"testing"

	ksm "github.com/keeper-security/secrets-manager-go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create test folders
func createTestFolders() []*ksm.KeeperFolder {
	return []*ksm.KeeperFolder{
		{FolderUid: "root1", ParentUid: "", Name: "Production"},
		{FolderUid: "root2", ParentUid: "", Name: "Development"},
		{FolderUid: "root3", ParentUid: "", Name: "QA"},
		{FolderUid: "child1", ParentUid: "root1", Name: "Databases"},
		{FolderUid: "child2", ParentUid: "root1", Name: "APIs"},
		{FolderUid: "child3", ParentUid: "root2", Name: "Apps"},
		{FolderUid: "grandchild1", ParentUid: "child1", Name: "MySQL"},
		{FolderUid: "grandchild2", ParentUid: "child1", Name: "PostgreSQL"},
	}
}

func TestBuildFolderTree(t *testing.T) {
	folders := createTestFolders()
	tree := BuildFolderTree(folders)

	// Verify all folders are in the map
	assert.Len(t, tree.folders, 8, "should have 8 folders")

	// Verify root folders
	assert.Len(t, tree.root, 3, "should have 3 root folders")

	// Verify hierarchy
	prodNode := tree.folders["root1"]
	require.NotNil(t, prodNode, "Production folder should exist")
	assert.Equal(t, "Production", prodNode.Name)
	assert.Nil(t, prodNode.Parent, "Production should have no parent")
	assert.Len(t, prodNode.Children, 2, "Production should have 2 children")

	// Verify child relationships
	dbNode := tree.folders["child1"]
	require.NotNil(t, dbNode, "Databases folder should exist")
	assert.Equal(t, "Databases", dbNode.Name)
	assert.Equal(t, prodNode, dbNode.Parent, "Databases parent should be Production")
	assert.Len(t, dbNode.Children, 2, "Databases should have 2 children")
}

func TestResolvePath_Simple(t *testing.T) {
	folders := createTestFolders()
	tree := BuildFolderTree(folders)

	// Test single-level path
	uid, err := tree.ResolvePath("Production")
	require.NoError(t, err)
	assert.Equal(t, "root1", uid)

	uid, err = tree.ResolvePath("Development")
	require.NoError(t, err)
	assert.Equal(t, "root2", uid)

	uid, err = tree.ResolvePath("QA")
	require.NoError(t, err)
	assert.Equal(t, "root3", uid)
}

func TestResolvePath_Nested(t *testing.T) {
	folders := createTestFolders()
	tree := BuildFolderTree(folders)

	// Test two-level path
	uid, err := tree.ResolvePath("Production/Databases")
	require.NoError(t, err)
	assert.Equal(t, "child1", uid)

	uid, err = tree.ResolvePath("Production/APIs")
	require.NoError(t, err)
	assert.Equal(t, "child2", uid)

	uid, err = tree.ResolvePath("Development/Apps")
	require.NoError(t, err)
	assert.Equal(t, "child3", uid)

	// Test three-level path
	uid, err = tree.ResolvePath("Production/Databases/MySQL")
	require.NoError(t, err)
	assert.Equal(t, "grandchild1", uid)

	uid, err = tree.ResolvePath("Production/Databases/PostgreSQL")
	require.NoError(t, err)
	assert.Equal(t, "grandchild2", uid)
}

func TestResolvePath_WithLeadingTrailingSlashes(t *testing.T) {
	folders := createTestFolders()
	tree := BuildFolderTree(folders)

	// Leading slash should be ignored
	uid, err := tree.ResolvePath("/Production/Databases")
	require.NoError(t, err)
	assert.Equal(t, "child1", uid)

	// Trailing slash should be ignored
	uid, err = tree.ResolvePath("Production/Databases/")
	require.NoError(t, err)
	assert.Equal(t, "child1", uid)

	// Both leading and trailing
	uid, err = tree.ResolvePath("/Production/Databases/")
	require.NoError(t, err)
	assert.Equal(t, "child1", uid)
}

func TestResolvePath_NotFound(t *testing.T) {
	folders := createTestFolders()
	tree := BuildFolderTree(folders)

	// Non-existent root folder
	_, err := tree.ResolvePath("NonExistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "folder not found")

	// Non-existent child folder
	_, err = tree.ResolvePath("Production/NonExistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "folder not found")

	// Non-existent grandchild
	_, err = tree.ResolvePath("Production/Databases/NonExistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "folder not found")

	// Wrong parent path
	_, err = tree.ResolvePath("Development/Databases")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "folder not found")
}

func TestResolvePath_EmptyPath(t *testing.T) {
	folders := createTestFolders()
	tree := BuildFolderTree(folders)

	_, err := tree.ResolvePath("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestResolvePath_CaseSensitive(t *testing.T) {
	folders := createTestFolders()
	tree := BuildFolderTree(folders)

	// Correct case should work
	_, err := tree.ResolvePath("Production")
	require.NoError(t, err)

	// Wrong case should fail
	_, err = tree.ResolvePath("production")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "folder not found")

	_, err = tree.ResolvePath("PRODUCTION")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "folder not found")
}

func TestGetPath(t *testing.T) {
	folders := createTestFolders()
	tree := BuildFolderTree(folders)

	// Test root folder
	path := tree.GetPath("root1")
	assert.Equal(t, "Production", path)

	// Test child folder
	path = tree.GetPath("child1")
	assert.Equal(t, "Production/Databases", path)

	// Test grandchild folder
	path = tree.GetPath("grandchild1")
	assert.Equal(t, "Production/Databases/MySQL", path)

	// Test non-existent UID
	path = tree.GetPath("nonexistent")
	assert.Equal(t, "", path)
}

func TestGetNode(t *testing.T) {
	folders := createTestFolders()
	tree := BuildFolderTree(folders)

	// Get existing node
	node := tree.GetNode("root1")
	require.NotNil(t, node)
	assert.Equal(t, "Production", node.Name)
	assert.Equal(t, "root1", node.UID)

	// Get non-existent node
	node = tree.GetNode("nonexistent")
	assert.Nil(t, node)
}

func TestListPaths(t *testing.T) {
	folders := createTestFolders()
	tree := BuildFolderTree(folders)

	paths := tree.ListPaths()
	assert.Len(t, paths, 8, "should have 8 paths")

	// Check that some expected paths are present
	assert.Contains(t, paths, "Production")
	assert.Contains(t, paths, "Production/Databases")
	assert.Contains(t, paths, "Production/Databases/MySQL")
	assert.Contains(t, paths, "Development/Apps")
}

func TestBuildFolderTree_EmptyList(t *testing.T) {
	tree := BuildFolderTree([]*ksm.KeeperFolder{})
	assert.NotNil(t, tree)
	assert.Len(t, tree.folders, 0)
	assert.Len(t, tree.root, 0)
}

func TestBuildFolderTree_OrphanedFolders(t *testing.T) {
	// Folder with non-existent parent should be treated as root
	folders := []*ksm.KeeperFolder{
		{FolderUid: "orphan1", ParentUid: "nonexistent", Name: "OrphanFolder"},
		{FolderUid: "root1", ParentUid: "", Name: "RootFolder"},
	}

	tree := BuildFolderTree(folders)
	assert.Len(t, tree.root, 2, "orphaned folder should be treated as root")

	orphan := tree.GetNode("orphan1")
	require.NotNil(t, orphan)
	assert.Nil(t, orphan.Parent, "orphaned folder should have no parent")
}

func TestResolvePath_SpecialCharacters(t *testing.T) {
	// Test folders with special characters in names
	folders := []*ksm.KeeperFolder{
		{FolderUid: "root1", ParentUid: "", Name: "Prod-2024"},
		{FolderUid: "child1", ParentUid: "root1", Name: "DB_MySQL"},
		{FolderUid: "child2", ParentUid: "root1", Name: "API (v2)"},
	}

	tree := BuildFolderTree(folders)

	// Test dash
	uid, err := tree.ResolvePath("Prod-2024")
	require.NoError(t, err)
	assert.Equal(t, "root1", uid)

	// Test underscore
	uid, err = tree.ResolvePath("Prod-2024/DB_MySQL")
	require.NoError(t, err)
	assert.Equal(t, "child1", uid)

	// Test parentheses and spaces
	uid, err = tree.ResolvePath("Prod-2024/API (v2)")
	require.NoError(t, err)
	assert.Equal(t, "child2", uid)
}
