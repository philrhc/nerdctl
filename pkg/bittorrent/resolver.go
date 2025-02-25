package bittorrent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/containerd/containerd/v2/core/remotes"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type resolver struct {
	scheme string
}

func NewResolver(scheme string) (remotes.Resolver, error) {
	return &resolver{
		scheme: scheme,
	}, nil
}

func (r *resolver) Resolve(ctx context.Context, magnetLink string) (name string, desc ocispec.Descriptor, err error) {
	rc, err := get(magnetLink)
	if err != nil {
		return "", ocispec.Descriptor{}, err
	}
	defer rc.Close()
	if err := json.NewDecoder(rc).Decode(&desc); err != nil {
		return "", ocispec.Descriptor{}, fmt.Errorf("failed to decode descriptor: %w", err)
	}
	return magnetLink, desc, nil
}

func (r *resolver) Fetcher(ctx context.Context, ref string) (remotes.Fetcher, error) {
	return &fetcher{r}, nil
}

func (r *resolver) Pusher(ctx context.Context, ref string) (remotes.Pusher, error) {
	return nil, fmt.Errorf("immutable remote")
}

type fetcher struct {
	r *resolver
}

func (f *fetcher) Fetch(ctx context.Context, desc ocispec.Descriptor) (io.ReadCloser, error) {
	magnetLink, err := getMagnetLink(desc)
	if err != nil {
		return nil, err
	}
	return get(magnetLink)
}

func getMagnetLink(desc ocispec.Descriptor) (string, error) {
	for _, u := range desc.URLs {
		if strings.HasPrefix(u, "magnet:") {
			return u, nil
		}
	}
	return "", fmt.Errorf("no CID is recorded")
}