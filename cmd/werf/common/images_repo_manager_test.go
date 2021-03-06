package common

import (
	"fmt"
	"testing"
)

func TestGetImagesRepoManager(t *testing.T) {
	expectationsByRepoMode := map[string]struct {
		namelessImageRepo        string
		namelessImageRepoWithTag string
		imageRepo                string
		imageRepoWithTag         string
		isMonorepo               bool
	}{
		MonorepoImagesRepoMode: {
			namelessImageRepo:        "repo",
			namelessImageRepoWithTag: "repo:tag",
			imageRepo:                "repo",
			imageRepoWithTag:         fmt.Sprintf("repo:image%stag", MonorepoTagPartsSeparator),
			isMonorepo:               true,
		},
		MultirepoImagesRepoMode: {
			namelessImageRepo:        "repo",
			namelessImageRepoWithTag: "repo:tag",
			imageRepo:                "repo/image",
			imageRepoWithTag:         "repo/image:tag",
			isMonorepo:               false,
		},
	}

	for imagesRepoMode, expected := range expectationsByRepoMode {
		t.Run(imagesRepoMode, func(t *testing.T) {
			m, err := GetImagesRepoManager("repo", imagesRepoMode)
			if err != nil {
				t.Error(err)
			}

			namelessImageRepo := m.ImageRepo("")
			if expected.namelessImageRepo != namelessImageRepo {
				t.Errorf("\n[EXPECTED]: %q\n[GOT]: %q", expected.namelessImageRepo, namelessImageRepo)
			}

			namelessImageRepoWithTag := m.ImageRepoWithTag("", "tag")
			if expected.namelessImageRepoWithTag != namelessImageRepoWithTag {
				t.Errorf("\n[EXPECTED]: %q\n[GOT]: %q", expected.namelessImageRepoWithTag, namelessImageRepoWithTag)
			}

			imageRepo := m.ImageRepo("image")
			if expected.imageRepo != imageRepo {
				t.Errorf("\n[EXPECTED]: %q\n[GOT]: %q", expected.imageRepo, imageRepo)
			}

			imageRepoWithTag := m.ImageRepoWithTag("image", "tag")
			if expected.imageRepoWithTag != imageRepoWithTag {
				t.Errorf("\n[EXPECTED]: %q\n[GOT]: %q", expected.imageRepoWithTag, imageRepoWithTag)
			}

			isMonorepo := m.IsMonorepo()
			if expected.isMonorepo != isMonorepo {
				t.Errorf("\n[EXPECTED]: %v\n[GOT]: %v", expected.isMonorepo, isMonorepo)
			}
		})
	}
}
