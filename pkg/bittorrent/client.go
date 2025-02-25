package bittorrent

import (
	"io"

	"github.com/anacrolix/torrent"
)

func get(magnet string) (io.ReadCloser, error) {
	c, _ := torrent.NewClient(nil)
	t, err := c.AddMagnet(magnet)
	if err != nil {
		return nil, err
	}
	<-t.GotInfo()
	r := t.Files()[0].NewReader()
	return r, nil
}

