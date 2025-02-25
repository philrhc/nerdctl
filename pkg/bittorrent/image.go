package bittorrent

import (
	"context"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/nerdctl/v2/pkg/api/types"
	"github.com/containerd/nerdctl/v2/pkg/imgutil"
)

// EnsureImage pull the specified image from IPFS.
func EnsureImage(ctx context.Context, client *containerd.Client, scheme string, ref string, options types.ImagePullOptions) (*imgutil.EnsuredImage, error) {
	resolver, err := NewResolver(scheme)
	if err != nil {
		return nil, err
	}
	return imgutil.PullImage(ctx, client, resolver, ref, options)
}
