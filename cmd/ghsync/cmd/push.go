package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-github/github"
	"github.com/izumin5210/ghsync"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

func newPushCmd() *cobra.Command {
	fs := afero.NewOsFs()
	viper := viper.New()
	viper.SetFs(fs)

	var (
		target struct {
			Slug string
			Base string
			HEAD string
		}
	)

	cmd := &cobra.Command{
		Use:  "push <owner>/<repo> [<src>]:<dst>",
		Args: cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			ctx := context.Background()
			origin, err := ghsync.BuildMetadata(ctx)
			if err != nil {
				return err
			}

			target.Slug = args[0]
			if target.HEAD == "" {
				chunks := []string{"ghsync", origin.Owner, origin.Repo}
				if origin.IsPR() {
					chunks = append(chunks, "pull", fmt.Sprint(origin.PR))
				} else {
					chunks = append(chunks, "branch", origin.Branch)
				}
				target.HEAD = strings.Join(chunks, "/")
			}

			paths := strings.SplitN(args[1], ":", 2)
			src, dst := paths[0], paths[1]
			_ = src

			zap.L().Debug("parameters",
				zap.String("src", src),
				zap.String("dest", dst),
				zap.Any("origin", origin),
				zap.Any("target", target),
			)

			ts := tokenSource()
			hc := httpClient(ctx, ts)
			gc := githubClient(hc)

			factory := ghsync.NewGithubContentRepositoryFactory(gc)
			repo, err := factory.Create(ctx, target.Slug, target.Base, target.HEAD, origin)
			if err != nil {
				return err
			}

			cont, err := repo.Get(ctx, dst)
			if err != nil {
				return err
			}

			switch cont.(type) {
			case ghsync.Submodule:
				ok, err := cont.Update(&ghsync.LocalSubmodule{SHA: origin.SHA})
				if err != nil {
					return err
				}
				if ok {
					err = repo.Update(ctx, cont)
					if err != nil {
						return err
					}
				} else {
					zap.L().Info("the content has not been updated")
				}
			default:
				return errors.New("currently support only submodule mode")
			}

			// update and commit

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&target.Base, "base", "master", "Base branch")
	cmd.PersistentFlags().StringVar(&target.HEAD, "head", "", "HEAD branch")

	return cmd
}

func githubClient(hc *http.Client) *github.Client {
	return github.NewClient(hc)
}

func tokenSource() oauth2.TokenSource {
	token := os.Getenv("GITHUB_TOKEN")
	return oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
}

func httpClient(ctx context.Context, ts oauth2.TokenSource) *http.Client {
	return oauth2.NewClient(ctx, ts)
}
