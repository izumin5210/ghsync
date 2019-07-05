package ghsync

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/github"
)

func NewGithubContentRepositoryFactory(
	cli *github.Client,
) ContentRepositoryFactory {
	return &githubContentRepositoryFactoryImpl{
		cli: cli,
	}
}

type githubContentRepositoryFactoryImpl struct {
	cli *github.Client
}

func (f *githubContentRepositoryFactoryImpl) Create(ctx context.Context, slug, base, head string, omd *OriginMetadata) (ContentRepository, error) {
	ownerAndRepo := strings.SplitN(slug, "/", 2)
	r := &githubContentRepositoryImpl{
		owner:      ownerAndRepo[0],
		repo:       ownerAndRepo[1],
		base:       base,
		head:       head,
		cli:        f.cli,
		originMeta: omd,
	}

	baseRef, resp, err := f.cli.Git.GetRef(ctx, r.owner, r.repo, "heads/"+head)
	if err != nil {
		if resp == nil || resp.StatusCode != 404 {
			return nil, err
		}
	}

	if baseRef == nil {
		baseRef, _, err = f.cli.Git.GetRef(ctx, r.owner, r.repo, "heads/"+head)
		if err != nil {
			return nil, err
		}
	}

	r.baseRef = baseRef

	return r, nil
}

type githubContentRepositoryImpl struct {
	owner, repo string
	head, base  string
	baseRef     *github.Reference
	cli         *github.Client
	originMeta  *OriginMetadata
}

func (r *githubContentRepositoryImpl) Get(ctx context.Context, path string) (Content, error) {
	f, d, _, err := r.cli.Repositories.GetContents(ctx, r.owner, r.repo, path, &github.RepositoryContentGetOptions{
		Ref: r.baseRef.GetRef(),
	})
	if err != nil {
		return nil, err
	}
	if d != nil {
		// TODO: not yet implemented
		return nil, fmt.Errorf("unsupported content type: %s", f.GetType())
	}
	switch f.GetType() {
	case "submodule":
		return &GithubContentSubmodule{cont: f}, nil
	case "file", "symlink":
		// TODO: not yet implemented
		return nil, fmt.Errorf("unsupported content type: %s", f.GetType())
	default:
		return nil, fmt.Errorf("unknown content type: %s", f.GetType())
	}
}

func (r *githubContentRepositoryImpl) Update(ctx context.Context, cont Content) error {
	if strings.TrimPrefix(r.baseRef.GetRef(), "refs/heads/") != r.head {
		refStr := "heads/" + r.head
		ref, _, err := r.cli.Git.CreateRef(ctx, r.owner, r.repo, &github.Reference{
			Ref:    &refStr,
			Object: r.baseRef.GetObject(),
		})
		if err != nil {
			return err
		}
		r.baseRef = ref
	}

	var tree *github.Tree

	switch cont := cont.(type) {
	case Submodule:
		path := cont.GetPath()
		mode := "160000"
		typ := "commit"
		sha := cont.GetSHA()
		var err error
		tree, _, err = r.cli.Git.CreateTree(ctx, r.owner, r.repo, r.baseRef.GetObject().GetSHA(), []github.TreeEntry{
			{Path: &path, Mode: &mode, Type: &typ, SHA: &sha},
		})
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported content type: %T", cont)
	}

	err := r.createCommit(ctx, tree)
	if err != nil {
		return err
	}

	err = r.createPullRequestIfNeeded(ctx, cont)
	if err != nil {
		return err
	}

	return nil
}

func (r *githubContentRepositoryImpl) createCommit(ctx context.Context, tree *github.Tree) error {
	author, _, err := r.cli.Users.Get(ctx, "")
	if err != nil {
		return err
	}

	date := time.Now()
	msg := strings.Join([]string{
		fmt.Sprintf("Use %s@%s", r.originMeta.GetSlug(), r.originMeta.SHA[:7]),
		"",
		"pull request: " + r.originMeta.GetPRURL(),
		"commit: " + r.originMeta.GetCommitURL(),
	}, "\n")
	commit, _, err := r.cli.Git.CreateCommit(ctx, r.owner, r.repo, &github.Commit{
		Message: &msg,
		Author: &github.CommitAuthor{
			Name:  author.Name,
			Email: author.Email,
			Date:  &date,
		},
		Parents: []github.Commit{{SHA: r.baseRef.GetObject().SHA}},
		Tree:    tree,
	})
	if err != nil {
		return err
	}

	ref, _, err := r.cli.Git.UpdateRef(ctx, r.owner, r.repo, &github.Reference{
		Ref: r.baseRef.Ref,
		Object: &github.GitObject{
			SHA: commit.SHA,
		},
	}, false)
	if err != nil {
		return err
	}
	r.baseRef = ref

	return nil
}

func (r *githubContentRepositoryImpl) createPullRequestIfNeeded(ctx context.Context, cont Content) error {
	pulls, _, err := r.cli.PullRequests.List(ctx, r.owner, r.repo, &github.PullRequestListOptions{
		State: "all",
		Head:  r.head,
	})
	if err != nil {
		return err
	}

	if len(pulls) == 0 {
		title := fmt.Sprintf("Update %s", cont.GetPath())
		body := fmt.Sprintf("from %s", r.originMeta.GetPRURL())
		pull, _, err := r.cli.PullRequests.Create(ctx, r.owner, r.repo, &github.NewPullRequest{
			Base:  &r.base,
			Head:  &r.head,
			Title: &title,
			Body:  &body,
		})
		if err != nil {
			return err
		}

		msg := fmt.Sprintf("Created a new pull request!\n:point_right: %s", pull.GetURL())
		_, _, err = r.cli.Issues.CreateComment(ctx, r.originMeta.Owner, r.originMeta.Repo, r.originMeta.PR, &github.IssueComment{
			Body: &msg,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
