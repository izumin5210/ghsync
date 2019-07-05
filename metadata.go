package ghsync

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// OriginMetadata contains the current repository metadata.
type OriginMetadata struct {
	Owner string
	Repo  string
	SHA   string
	URL   string
	PR    int
}

// BuildMetadata collects data from a current environment.
func BuildMetadata() *OriginMetadata {
	md := new(OriginMetadata)
	if os.Getenv("CI") == "true" {
		if os.Getenv("TRAVIS") == "true" {
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
			md.URL = "https://github.com/" + md.GetSlug()
		}
		if os.Getenv("CIRCLECI") == "true" {
			md.Owner = os.Getenv("CIRCLE_PROJECT_USERNAME")
			md.Repo = os.Getenv("CIRCLE_PROJECT_REPONAME")
			md.SHA = os.Getenv("CIRCLE_SHA")
			if pr, ok := os.LookupEnv("CIRCLE_PR_NUMBER"); ok {
				md.PR, _ = strconv.Atoi(pr)
			}
			md.URL = os.Getenv("CIRCLE_REPOSITORY_URL")
		}
	}
	return md
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
