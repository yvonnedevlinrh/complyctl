// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

// Fetcher abstracts registry access for testing without a live OCI registry.
type Fetcher interface {
	DefinitionVersion(ctx context.Context, modulePath string) (string, string, error)
}

// Client provides OCI registry access with credential-based auth via oras-go.
type Client struct {
	registryURL string
	credFunc    auth.CredentialFunc
	fetcher     Fetcher
	plainHTTP   bool
}

func NewClient(registryURL string, credFunc auth.CredentialFunc) *Client {
	return &Client{
		registryURL: registryURL,
		credFunc:    credFunc,
		fetcher:     nil,
		plainHTTP:   strings.HasPrefix(registryURL, "http://"),
	}
}

func NewClientWithFetcher(registryURL string, credFunc auth.CredentialFunc, fetcher Fetcher) *Client {
	return &Client{
		registryURL: registryURL,
		credFunc:    credFunc,
		fetcher:     fetcher,
		plainHTTP:   strings.HasPrefix(registryURL, "http://"),
	}
}

func (c *Client) DefinitionVersion(ctx context.Context, modulePath string) (string, string, error) {
	if modulePath == "" {
		return "", "", fmt.Errorf("module path cannot be empty")
	}

	if c.fetcher != nil {
		return c.fetcher.DefinitionVersion(ctx, modulePath)
	}

	ref := fmt.Sprintf("%s/%s:latest", c.registryHost(), modulePath)
	digest, version, err := c.fetchVersion(ctx, ref)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch version for %s: %w", modulePath, err)
	}

	return digest, version, nil
}

// NewRemoteRepository creates an authenticated oras-go remote.Repository
// for use as the source argument to oras.Copy().
func (c *Client) NewRemoteRepository(ctx context.Context, policyID string) (*remote.Repository, error) {
	repoName := fmt.Sprintf("%s/%s", c.registryHost(), policyID)
	repo, err := c.newRepository(repoName)
	if err != nil {
		return nil, err
	}
	return repo, nil
}

func (c *Client) registryHost() string {
	host := c.registryURL
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	return strings.TrimSuffix(host, "/")
}

func (c *Client) newRepository(repoName string) (*remote.Repository, error) {
	repo, err := remote.NewRepository(repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI repository client for %s: %w", repoName, err)
	}
	repo.PlainHTTP = c.plainHTTP
	if c.credFunc != nil {
		repo.Client = &auth.Client{
			Client:     http.DefaultClient,
			Credential: c.credFunc,
		}
	}
	return repo, nil
}

func (c *Client) fetchVersion(ctx context.Context, ref string) (string, string, error) {
	parsedRef, err := registry.ParseReference(ref)
	if err != nil {
		return "", "", fmt.Errorf("invalid OCI reference %s: %w", ref, err)
	}

	repoName := fmt.Sprintf("%s/%s", parsedRef.Registry, parsedRef.Repository)
	repo, err := c.newRepository(repoName)
	if err != nil {
		return "", "", err
	}

	tag := parsedRef.Reference
	if tag == "" {
		tag = "latest"
	}

	desc, err := repo.Resolve(ctx, tag)
	if err != nil {
		return "", "", fmt.Errorf("OCI version resolution failed for %s: %w", ref, err)
	}

	digest := desc.Digest.String()
	version := tag

	return digest, version, nil
}
