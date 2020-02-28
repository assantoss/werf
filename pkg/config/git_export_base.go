package config

import (
	"path/filepath"
	"strings"
)

type GitExportBase struct {
	*GitExport
	StageDependencies *StageDependencies
}

func (c *ExportBase) GitMappingAdd() string {
	if c.Add == "/" {
		return ""
	}
	return filepath.FromSlash(strings.TrimPrefix(c.Add, "/"))
}

func (c *ExportBase) GitMappingTo() string {
	return c.To
}

func (c *ExportBase) GitMappingIncludePaths() []string {
	return gitMappingPaths(c.IncludePaths)
}

func (c *ExportBase) GitMappingExcludePath() []string {
	return gitMappingPaths(c.ExcludePaths)
}

func (c *GitExportBase) GitMappingStageDependencies() *StageDependencies {
	s := &StageDependencies{}
	s.Install = gitMappingPaths(c.StageDependencies.Install)
	s.BeforeSetup = gitMappingPaths(c.StageDependencies.BeforeSetup)
	s.Setup = gitMappingPaths(c.StageDependencies.Setup)
	return s
}

func gitMappingPaths(paths []string) []string {
	var newPaths []string
	for _, path := range paths {
		newPaths = append(newPaths, filepath.FromSlash(path))
	}

	return newPaths
}
