package main

import (
	"fmt"
	"os"

	"github.com/flant/werf/pkg/path_matcher"
	"github.com/flant/werf/pkg/true_git"
)

func main() {
	err := true_git.Init(true_git.Options{})
	if err != nil {
		panic(err)
	}

	f, err := os.Create("my-patch.diff")
	if err != nil {
		panic(err)
	}

	p, err := true_git.PatchWithSubmodules(f, os.Args[1], "my-work-tree", true_git.PatchOptions{
		FromCommit:  os.Args[2],
		ToCommit:    os.Args[3],
		PathMatcher: &path_matcher.GitMappingPathMatcher{},
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Patch paths=%v binary-paths=%v\n", p.Paths, p.BinaryPaths)
}
