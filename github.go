package ghsync

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"go.uber.org/zap"
)

const TracebackLimit = 100

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
		baseRef, _, err = f.cli.Git.GetRef(ctx, r.owner, r.repo, "heads/"+base)
		if err != nil {
			return nil, err
		}
	}

	r.baseRef = baseRef

	zap.L().Debug("base reference was fetched", zap.Any("ref", baseRef))

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

	zap.L().Debug("target content was fetched", zap.Any("content", f))

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
	baseCommit, err := r.findBaseCommit(ctx, cont, TracebackLimit)
	if err != nil {
		return err
	}

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

		zap.L().Debug("new base reference was created", zap.Any("ref", ref))
	}

	var tree *github.Tree

	switch cont := cont.(type) {
	case Submodule:
		path := cont.GetPath()
		mode := "160000"
		typ := "commit"
		sha := cont.GetSHA()
		var err error
		tree, _, err = r.cli.Git.CreateTree(ctx, r.owner, r.repo, baseCommit.GetSHA(), []github.TreeEntry{
			{Path: &path, Mode: &mode, Type: &typ, SHA: &sha},
		})
		if err != nil {
			return err
		}
		zap.L().Debug("new tree was created", zap.Any("tree", tree))
	default:
		return fmt.Errorf("unsupported content type: %T", cont)
	}

	err = r.createCommit(ctx, baseCommit, tree)
	if err != nil {
		return err
	}

	err = r.createPullRequestIfNeeded(ctx, cont)
	if err != nil {
		return err
	}

	return nil
}

func (r *githubContentRepositoryImpl) findBaseCommit(ctx context.Context, cont Content, tracebackLimit int) (*github.Commit, error) {
	baseRef, _, err := r.cli.Git.GetRef(ctx, r.owner, r.repo, "heads/"+r.base)
	if err != nil {
		return nil, err
	}

	commit, _, err := r.cli.Git.GetCommit(ctx, r.owner, r.repo, baseRef.GetObject().GetSHA())
	if err != nil {
		return nil, err
	}

	for i := 0; i < tracebackLimit; i++ {
		switch cont := cont.(type) {
		case Submodule:
			path := cont.GetPath()
			f, d, _, err := r.cli.Repositories.GetContents(ctx, r.owner, r.repo, path, &github.RepositoryContentGetOptions{
				Ref: commit.GetSHA(),
			})
			if err != nil {
				return nil, err
			}

			if d != nil || f.GetType() != "submodule" {
				return nil, fmt.Errorf("unsupported content type: %s", f.GetType())
			}

			compar, _, err := r.cli.Repositories.CompareCommits(ctx, r.originMeta.Owner, r.originMeta.Repo, f.GetSHA(), cont.GetSHA())
			if err != nil {
				return nil, err
			}
			if compar.GetBehindBy() == 0 {
				return commit, nil
			}
		default:
			return nil, fmt.Errorf("unsupported content type: %T", cont)
		}

		if len(commit.Parents) == 0 {
			break
		}

		nextCommit, _, err := r.cli.Git.GetCommit(ctx, r.owner, r.repo, commit.Parents[0].GetSHA())
		if err != nil {
			return nil, err
		}
		commit = nextCommit
	}

	return nil, fmt.Errorf("no appropriate base commit found")
}

func (r *githubContentRepositoryImpl) createCommit(ctx context.Context, baseCommit *github.Commit, tree *github.Tree) error {
	user, err := GetUser(ctx)
	if err != nil {
		author, _, err := r.cli.Users.Get(ctx, "")
		if err != nil {
			return err
		}
		user = &User{Name: author.GetName(), Email: author.GetEmail()}
	}

	date := time.Now()
	msg := strings.Join([]string{
		fmt.Sprintf("Use %s@%s", r.originMeta.GetSlug(), r.originMeta.SHA[:7]),
		"",
		"commit: " + r.originMeta.GetCommitURL(),
	}, "\n")
	commit, _, err := r.cli.Git.CreateCommit(ctx, r.owner, r.repo, &github.Commit{
		Message: &msg,
		Author: &github.CommitAuthor{
			Name:  &user.Name,
			Email: &user.Email,
			Date:  &date,
		},
		Parents: []github.Commit{{SHA: baseCommit.SHA}},
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
	}, true)
	if err != nil {
		return err
	}
	r.baseRef = ref

	zap.L().Info("a new commit was created",
		zap.String("sha", commit.GetSHA()),
		zap.String("url", commit.GetURL()),
		zap.String("html_url", commit.GetHTMLURL()),
		zap.String("message", commit.GetMessage()),
	)

	return nil
}

func (r *githubContentRepositoryImpl) createPullRequestIfNeeded(ctx context.Context, cont Content) error {
	if !r.originMeta.IsPR() {
		zap.L().Info("skip creating a pull request because this is not PR build")
		return nil
	}

	pulls, _, err := r.cli.PullRequests.List(ctx, r.owner, r.repo, &github.PullRequestListOptions{
		State: "open",
		Head:  r.originMeta.Owner + ":" + r.head,
	})
	if err != nil {
		return err
	}

	if len(pulls) > 0 {
		urls := make([]string, len(pulls))
		for i, p := range pulls {
			urls[i] = p.GetURL()
		}
		zap.L().Info("skip creating a pull request because already exist",
			zap.String("branch", r.originMeta.Owner+":"+r.head),
			zap.Strings("urls", urls),
		)
		return nil
	}

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

	msg := fmt.Sprintf("Created a new pull request!\n:point_right: %s", pull.GetHTMLURL())
	_, _, err = r.cli.Issues.CreateComment(ctx, r.originMeta.Owner, r.originMeta.Repo, r.originMeta.PR, &github.IssueComment{
		Body: &msg,
	})
	if err != nil {
		return err
	}

	zap.L().Info("a new pull request was craeted",
		zap.Int("number", pull.GetNumber()),
		zap.String("url", pull.GetURL()),
		zap.String("html_url", pull.GetHTMLURL()),
		zap.String("message", pull.GetTitle()),
	)

	return nil
}
