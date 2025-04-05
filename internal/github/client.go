package github

import (
	"context"
	"time"

	"github.com/DragonAirDragon/GO/internal/models"
	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

type Client struct {
	client *github.Client
}

func NewClient(token string) (*Client, error) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	return &Client{
		client: client,
	}, nil
}

func (c *Client) GetRepositories(ctx context.Context, username string) ([]models.Repository, error) {
	opt := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
		Sort:        "created",
		Direction:   "desc",
	}

	var allRepos []models.Repository
	for {
		repos, resp, err := c.client.Repositories.List(ctx, username, opt)
		if err != nil {
			return nil, err
		}

		for _, repo := range repos {
			description := ""
			if repo.Description != nil {
				description = *repo.Description
			}

			createdAt := ""
			if repo.CreatedAt != nil {
				createdAt = repo.CreatedAt.Format(time.RFC3339)
			}

			repoURL := ""
			if repo.HTMLURL != nil {
				repoURL = *repo.HTMLURL
			}

			allRepos = append(allRepos, models.Repository{
				Name:        *repo.Name,
				Description: description,
				URL:         repoURL,
				CreatedAt:   createdAt,
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allRepos, nil
}

func (c *Client) GetLatestCommit(ctx context.Context, username, repo string) ([]models.Commit, error) {
	opt := &github.CommitsListOptions{
		ListOptions: github.ListOptions{PerPage: 5},
	}

	commits, _, err := c.client.Repositories.ListCommits(ctx, username, repo, opt)
	if err != nil {
		return nil, err
	}

	var result []models.Commit
	for _, commit := range commits {
		sha := ""
		if commit.SHA != nil {
			sha = *commit.SHA
		}

		message := ""
		if commit.Commit != nil && commit.Commit.Message != nil {
			message = *commit.Commit.Message
		}

		author := ""
		if commit.Author != nil && commit.Author.Login != nil {
			author = *commit.Author.Login
		} else if commit.Commit != nil && commit.Commit.Author != nil && commit.Commit.Author.Name != nil {
			author = *commit.Commit.Author.Name
		}

		date := ""
		if commit.Commit != nil && commit.Commit.Author != nil && commit.Commit.Author.Date != nil {
			date = commit.Commit.Author.Date.Format(time.RFC3339)
		}

		url := ""
		if commit.HTMLURL != nil {
			url = *commit.HTMLURL
		}

		result = append(result, models.Commit{
			SHA:     sha,
			Message: message,
			Author:  author,
			Date:    date,
			URL:     url,
		})
	}

	return result, nil
}
