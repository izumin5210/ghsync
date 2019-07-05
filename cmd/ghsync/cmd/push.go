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
	"golang.org/x/oauth2"
)

func newPushCmd() *cobra.Command {
	fs := afero.NewOsFs()
	viper := viper.New()
	viper.SetFs(fs)

	var (
		origin = ghsync.BuildMetadata()
		target struct {
			Slug      string
			Base      string
			HEAD      string
			Submodule string
		}
	)

	cmd := &cobra.Command{
		Use:  "push <owner>/<repo>",
		Args: cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			ctx := context.Background()
			ts := tokenSource()
			hc := httpClient(ctx, ts)
			gc := githubClient(hc)

			target.Slug = args[0]
			if target.HEAD == "" {
				chunks := []string{"ghsync", origin.Owner, origin.Repo}
				if origin.PR != 0 {
					chunks = append(chunks, fmt.Sprint(origin.PR))
				}
				target.HEAD = strings.Join(chunks, "/")
			}

			factory := ghsync.NewGithubContentRepositoryFactory(gc)
			repo, err := factory.Create(ctx, target.Slug, target.Base, target.HEAD, origin)
			if err != nil {
				return err
			}

			if target.Submodule != "" {
				cont, err := repo.Get(ctx, target.Submodule)
				if err != nil {
					return err
				}

				ok, err := cont.Update(&ghsync.LocalSubmodule{SHA: origin.SHA})
				if err != nil {
					return err
				}
				if ok {
					err = repo.Update(ctx, cont)
					if err != nil {
						return err
					}
				}
			} else {
				return errors.New("currently support only submodule mode")
			}

			// update and commit

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&target.Submodule, "submodule", "", "Destination path of git submodule")
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
