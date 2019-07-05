package ghsync

import (
	"context"
	"fmt"

	"github.com/google/go-github/github"
)

type ContentRepository interface {
	Get(ctx context.Context, path string) (Content, error)
	Update(ctx context.Context, cont Content) error
}

type ContentRepositoryFactory interface {
	Create(ctx context.Context, slug, base, head string, md *OriginMetadata) (ContentRepository, error)
}

type Content interface {
	GetPath() string
	Update(Content) (updated bool, err error)
}

type Submodule interface {
	Content
	GetSHA() string
}

var (
	_ Submodule = (*GithubContentSubmodule)(nil)
)

type LocalSubmodule struct {
	SHA string
}

func (c *LocalSubmodule) GetPath() string {
	return ""
}

func (c *LocalSubmodule) GetSHA() string {
	return c.SHA
}

func (c *LocalSubmodule) Update(other Content) (bool, error) {
	return false, fmt.Errorf("%T.Update is unsupported", c)
}

type GithubContentSubmodule struct {
	cont *github.RepositoryContent
}

func (c *GithubContentSubmodule) GetPath() string {
	return *c.cont.Path
}

func (c *GithubContentSubmodule) GetSHA() string {
	return *c.cont.SHA
}

func (c *GithubContentSubmodule) Update(other Content) (bool, error) {
	c2, ok := other.(Submodule)
	if !ok {
		return false, fmt.Errorf("type mismatch: want %T, got %T", c, other)
	}

	if c.GetSHA() == c2.GetSHA() {
		return false, nil
	}

	sha := c2.GetSHA()
	c.cont.SHA = &sha
	return true, nil
}
