package torrent

import (
	"context"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hekmon/transmissionrpc/v3"
	"golift.io/starr/debuglog"
)

func waitForCompletion(torrentId int64) (io.ReadCloser, error) {
	transmissionClient := client()
	for {
		torrents, err := transmissionClient.TorrentGet(context.Background(), []string{"percentDone", "files"}, []int64{torrentId})
		if err != nil {
			panic(err)
		}

		//TODO: better way of ensuring that only one torrent returned
		retrievedTorrent := torrents[0]

		//isFinished returns false for some reason
		percentDone := *retrievedTorrent.PercentDone

		if percentDone == 1 {
			filename := retrievedTorrent.Files[0].Name
			return os.Open("/var/tmp/docker_transmission_mount/downloads/complete/" + filename)
		} else {
			time.Sleep(1 * time.Second)
		}
	}
}

func get(magnetLink string) (_ io.ReadCloser, retErr error) {
	transmissionClient := client()

	torrentAddPayload := transmissionrpc.TorrentAddPayload{
		Filename: &magnetLink,
	}
	addedTorrent, err := transmissionClient.TorrentAdd(context.Background(), torrentAddPayload)
	if err != nil {
		panic(err)
	}

	return waitForCompletion(*addedTorrent.ID)

}

func client() *transmissionrpc.Client {
	endpoint, err := url.Parse("http://127.0.0.1:9091/transmission/rpc")
	if err != nil {
		panic(err)
	}

	httpClient := cleanhttp.DefaultPooledClient()
	httpClient.Transport = debuglog.NewLoggingRoundTripper(debuglog.Config{
		Redact: []string{endpoint.User.String()},
	}, httpClient.Transport)

	tbt, err := transmissionrpc.New(endpoint, &transmissionrpc.Config{
		CustomClient: httpClient,
	})
	if err != nil {
		panic(err)
	}
	return tbt
}
