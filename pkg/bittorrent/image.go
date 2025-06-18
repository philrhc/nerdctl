package bittorrent

import (
	"bytes"
	"context"
	"encoding/json"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/content"
	"github.com/containerd/containerd/v2/core/images/converter"
	"github.com/containerd/nerdctl/v2/pkg/api/types"
	"github.com/containerd/nerdctl/v2/pkg/imgutil"
	"github.com/containerd/nerdctl/v2/pkg/platformutil"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// EnsureImage pull the specified image from IPFS.
func EnsureImage(ctx context.Context, client *containerd.Client, scheme string, ref string, options types.ImagePullOptions) (*imgutil.EnsuredImage, error) {
	resolver, err := newResolver(scheme)
	if err != nil {
		return nil, err
	}
	return imgutil.PullImage(ctx, client, resolver, ref, options)
}

func Push(ctx context.Context, client *containerd.Client, ref string, layerConvert converter.ConvertFunc, allPlatforms bool, platform []string) (string, error) {
	platformMC, err := platformutil.NewMatchComparer(allPlatforms, platform)
	if err != nil {
		return "", err
	}

	ctx, done, err := client.WithLease(ctx)
	if err != nil {
		return "", err
	}

	defer done(ctx)
	// imgs, err := client.ImageService().List(ctx)
	// fmt.Printf(imgs[0].Name)
	// if err != nil {
	// 	return "", err
	// }
	img, err := client.ImageService().Get(ctx, ref)
	
	if err != nil {
		return "", err
	}

	c, err := newClient()
	if err != nil {
		return "", err
	}

	desc, err := converter.IndexConvertFuncWithHook(layerConvert, true, platformMC, converter.ConvertHooks{
		PostConvertHook: pushBlobHook(c),
	})(ctx, client.ContentStore(), img.Target)
	if err != nil {
		return "", err
	}

	root, err := json.Marshal(desc)
	if err != nil {
		return "", err
	}
	return c.seed(bytes.NewReader(root))
}

func pushBlobHook(client *Client) converter.ConvertHookFunc {
	return func(ctx context.Context, cs content.Store, desc ocispec.Descriptor, newDesc *ocispec.Descriptor) (*ocispec.Descriptor, error) {
		resultDesc := newDesc
		if resultDesc == nil {
			descCopy := desc
			resultDesc = &descCopy
		}
		ra, err := cs.ReaderAt(ctx, *resultDesc)
		if err != nil {
			return nil, err
		}
		magnetLink, err := client.seed(content.NewReader(ra))
		if err != nil {
			return nil, err
		}
		resultDesc.URLs = []string{magnetLink}
		return resultDesc, nil
	}
}

