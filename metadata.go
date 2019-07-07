package ghsync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// OriginMetadata contains the current repository metadata.
type OriginMetadata struct {
	Owner  string
	Repo   string
	Branch string
	SHA    string
	URL    string
	PR     int

	ci ci
}

type ci int

const (
	ciNotUsed ci = iota
	ciUnknown
	ciTravis
	ciCircle
)

var (
	remoteURLPattern = regexp.MustCompile(`(?:https://github\.com/|git@github\.com:)([\w-]+)/([\w-]+)(?:\.git)?`)
)

// BuildMetadata collects data from a current environment.
func BuildMetadata(ctx context.Context) (*OriginMetadata, error) {
	md := new(OriginMetadata)
	if os.Getenv("CI") == "true" {
		switch {
		case os.Getenv("TRAVIS") == "true":
			md.ci = ciTravis
			chunks := strings.SplitN(os.Getenv("TRAVIS_REPO_SLUG"), "/", 2)
			md.Owner = chunks[0]
			md.Repo = chunks[1]
			md.SHA = os.Getenv("TRAVIS_PULL_REQUEST_SHA")
			if md.SHA == "" {
				md.SHA = os.Getenv("TRAVIS_COMMIT")
			}
			if pr, ok := os.LookupEnv("TRAVIS_PULL_REQUEST"); ok {
				md.PR, _ = strconv.Atoi(pr)
			}
			md.Branch = os.Getenv("TRAVIS_PULL_REQUEST_BRANCH")
			if md.Branch == "" {
				md.Branch = os.Getenv("TRAVIS_BRANCH")
			}
			md.URL = "https://github.com/" + md.GetSlug()
		case os.Getenv("CIRCLECI") == "true":
			md.ci = ciCircle
			md.Owner = os.Getenv("CIRCLE_PROJECT_USERNAME")
			md.Repo = os.Getenv("CIRCLE_PROJECT_REPONAME")
			md.SHA = os.Getenv("CIRCLE_SHA")
			if pr, ok := os.LookupEnv("CIRCLE_PR_NUMBER"); ok {
				md.PR, _ = strconv.Atoi(pr)
			}
			md.Branch = os.Getenv("CIRCLE_BRANCH")
			md.URL = os.Getenv("CIRCLE_REPOSITORY_URL")
		default:
			md.ci = ciUnknown
		}
	} else {
		out, err := exec.CommandContext(ctx, "git", "rev-parse", "HEAD").Output()
		if err != nil {
			return nil, err
		}
		md.SHA = strings.TrimSpace(string(out))
		out, err = exec.CommandContext(ctx, "git", "remote", "get-url", "origin").Output()
		if err != nil {
			return nil, err
		}
		ms := remoteURLPattern.FindStringSubmatch(strings.TrimSpace(string(out)))
		if len(ms) != 3 {
			return nil, errors.New("failed to parse git remote url")
		}
		md.Owner = ms[1]
		md.Repo = ms[2]
		md.URL = fmt.Sprintf("https://github.com/%s/%s", md.Owner, md.Repo)
		out, err = exec.CommandContext(ctx, "git", "symbolic-ref", "--short", "HEAD").Output()
		if err != nil {
			return nil, err
		}
		md.Branch = strings.TrimSpace(string(out))
	}
	return md, nil
}

func (m *OriginMetadata) GetPRURL() string {
	return fmt.Sprintf("https://github.com/%s/pull/%d", m.URL, m.PR)
}

func (m *OriginMetadata) GetCommitURL() string {
	return fmt.Sprintf("https://github.com/%s/commit/%s", m.URL, m.SHA)
}

func (m *OriginMetadata) GetSlug() string {
	return m.Owner + "/" + m.Repo
}

func (m *OriginMetadata) IsPR() bool {
	return m.PR > 0
}
