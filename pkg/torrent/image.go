package torrent

import (
	"context"
	"encoding/json"
	"log"
	"net/url"
	"os"

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/content"
	"github.com/containerd/containerd/v2/core/images/converter"
	"github.com/containerd/nerdctl/v2/pkg/api/types"
	"github.com/containerd/nerdctl/v2/pkg/imgutil"
	"github.com/containerd/nerdctl/v2/pkg/platformutil"
	"github.com/containerd/nerdctl/v2/pkg/referenceutil"

	"github.com/hekmon/transmissionrpc/v3"
)

// EnsureImage pull the specified image from IPFS.
func EnsureImage(ctx context.Context, client *containerd.Client, scheme, ref string, options types.ImagePullOptions) (*imgutil.EnsuredImage, error) {
	r, err := NewResolver(scheme)
	if err != nil {
		return nil, err
	}
	return imgutil.PullImage(ctx, client, r, ref, options)
}

// Push pushes the specified image to IPFS.
func Push(ctx context.Context, client *containerd.Client, rawRef string, layerConvert converter.ConvertFunc, allPlatforms bool, platform []string) (string, error) {
	platformMC, err := platformutil.NewMatchComparer(allPlatforms, platform)
	if err != nil {
		return "", err
	}

	ref, err := referenceutil.ParseAny(rawRef)
	if err != nil {
		return "", err
	}

	//TODO: ensure image contents are fully downloaded

	//TODO: look up torrent daemon path?

	ctx, done, err := client.WithLease(ctx)
	if err != nil {
		return "", err
	}
	defer done(ctx)
	img, err := client.ImageService().Get(ctx, ref.String())
	if err != nil {
		return "", err
	}

	//TODO: create folder

	desc, err := converter.IndexConvertFuncWithHook(layerConvert, true, platformMC, converter.ConvertHooks{
		PostConvertHook: storeBlobHook(),
	})(ctx, client.ContentStore(), img.Target)
	if err != nil {
		return "", err
	}

	root, err := json.Marshal(desc)
	if err != nil {
		return "", err
	}

	magnet := store("manifest", root)

	return magnet, nil
}

func store(name string, root []byte) string {
	os.WriteFile("/var/tmp/docker_transmission_mount/downloads/complete/"+name, root, 0755)
	magnet := serve(name)
	return magnet
}

func storeBlobHook() converter.ConvertHookFunc {
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

		buffer := make([]byte, ra.Size()) // Adjust size as needed
		_, err = ra.ReadAt(buffer, 0)
		if err != nil {
			return nil, err
		}

		err = os.WriteFile("/var/tmp/docker_transmission_mount/downloads/complete/"+resultDesc.Digest.Encoded(), buffer, 0755)
		if err != nil {
			return nil, err
		}

		magnet := serve(resultDesc.Digest.Encoded())

		resultDesc.URLs = []string{magnet}

		return resultDesc, nil
	}
}

func serve(filename string) string {
	file, err := os.Open("/var/tmp/docker_transmission_mount/downloads/complete/" + filename)
	if err != nil {
		log.Fatal(err)
	}
	fi, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}

	pieceLength := fi.Size()
	info := metainfo.Info{
		PieceLength: pieceLength,
	}
	err = info.BuildFromFilePath("/var/tmp/docker_transmission_mount/downloads/complete/" + filename)
	if err != nil {
		panic(err)
	}

	mi := metainfo.MetaInfo{
		InfoBytes: bencode.MustMarshal(info),
	}

	mmi := bencode.MustMarshal(mi)

	err = os.WriteFile("/var/tmp/docker_transmission_mount/downloads/complete/"+filename+".torrent", mmi, 0755)
	if err != nil {
		panic(err)
	}

	endpoint, err := url.Parse("http://127.0.0.1:9091/transmission/rpc")
	if err != nil {
		panic(err)
	}

	tbt, err := transmissionrpc.New(endpoint, nil)
	if err != nil {
		panic(err)
	}

	torrent_filepath := "/downloads/complete/" + filename + ".torrent"
	torrentAddPayload := transmissionrpc.TorrentAddPayload{
		Filename: &torrent_filepath,
	}
	torrent, err := tbt.TorrentAdd(context.Background(), torrentAddPayload)
	if err != nil {
		panic(err)
	}

	addedTorrent, err := tbt.TorrentGet(context.Background(), []string{"magnetLink", "status"}, []int64{*torrent.ID})
	if err != nil {
		panic(err)
	}

	return *addedTorrent[0].MagnetLink
}
