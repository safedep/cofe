package vet

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v54/github"
	"github.com/safedep/dry/utils"
	"github.com/safedep/vet/pkg/common/logger"
	"golang.org/x/oauth2"
)

func GetGithubClient() (*github.Client, error) {
	github_token := os.Getenv("GITHUB_TOKEN")
	if !utils.IsEmptyString(github_token) {
		logger.Debugf("Found GITHUB_TOKEN env variable, using it to access Github.")
		return nil, fmt.Errorf("GITHUB_TOKEN is not set")
	}

	if utils.IsEmptyString(github_token) {
		rateLimitedClient, err := githubRateLimitedClient(http.DefaultTransport)
		if err != nil {
			return nil, err
		}

		logger.Debugf("Creating a Github client without credential")
		return github.NewClient(rateLimitedClient), nil
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: github_token,
	})

	baseClient := oauth2.NewClient(context.Background(), tokenSource)
	rateLimitedClient, err := githubRateLimitedClient(baseClient.Transport)
	if err != nil {
		return nil, err
	}

	logger.Debugf("Created a new Github client with credential")
	return github.NewClient(rateLimitedClient), nil
}

// This is currently effective only for Github secondary rate limits
// https://docs.github.com/en/rest/overview/rate-limits-for-the-rest-api
func githubRateLimitedClient(transport http.RoundTripper) (*http.Client, error) {
	var options []github_ratelimit.Option

	if !githubClientRateLimitBlockDisabled() {
		logger.Debugf("Adding Github rate limit callbacks to client")

		options = append(options, github_ratelimit.WithLimitDetectedCallback(func(cc *github_ratelimit.CallbackContext) {
			logger.Infof("Github rate limit detected, sleep until: %s", cc.SleepUntil)
		}))
	}

	rateLimitedClient, err := github_ratelimit.NewRateLimitWaiterClient(transport, options...)
	if err != nil {
		return nil, err
	}

	return rateLimitedClient, err
}

// We implement this as an internal feature i.e. without a config or an UI option because
// we want this to be the default behaviour *always* unless user want to explicitly disable it
func githubClientRateLimitBlockDisabled() bool {
	ret, err := strconv.ParseBool(os.Getenv("VET_GITHUB_DISABLE_RATE_LIMIT_BLOCKING"))
	if err != nil {
		return false
	}

	return ret
}
