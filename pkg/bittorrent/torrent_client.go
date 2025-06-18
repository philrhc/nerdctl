package bittorrent

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/anacrolix/torrent"
)

type Client struct {
	Client *torrent.Client
}

func newClientConfig() *torrent.ClientConfig {
	cfg := torrent.NewDefaultClientConfig()
	cfg.ListenPort = 0
	cfg.NoDHT = true
	cfg.DisablePEX = true
	cfg.NoDefaultPortForwarding = true
	cfg.Seed = true
	cfg.Debug = true
	cfg.AcceptPeerConnections = true
	cfg.AlwaysWantConns = true
	cfg.DisableTrackers = true
	cfg.LocalServiceDiscovery = true
	return cfg
}

func newClient() (*Client, error) {
	clientConfig := newClientConfig()
	client, err := torrent.NewClient(clientConfig)
	if err != nil {
		return nil, err
	}
	return &Client{Client: client}, nil
}

func (c Client) get(magnet string) (io.ReadCloser, error) {
	t, err := c.Client.AddMagnet(magnet)
	if err != nil {
		return nil, err
	}
	<-t.GotInfo()
	r := t.Files()[0].NewReader()
	return r, nil
}

func (c Client) seed(r io.Reader) (string, error) {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	contentType := mw.FormDataContentType()
	go func() {
		fw, err := mw.CreateFormFile("file", "file")
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		if _, err := io.Copy(fw, r); err != nil {
			pw.CloseWithError(err)
			return
		}
		if err := mw.Close(); err != nil {
			pw.CloseWithError(err)
			return
		}
		pw.Close()
	}()
	req, err := http.NewRequest("POST", "http://localhost:8080/create", pr)
	if err != nil {
		fmt.Println("Could not create request:", err.Error())
		return "", err
	}
	req.Header.Add("Content-Type", contentType)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error making POST request:", err.Error())
		return "", err
	}

	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()
	fmt.Println("Response Status:", resp.Status)
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("failed to add; status code: %v", resp.StatusCode)
	}
/* 	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	// Print the raw body directly to the console
	fmt.Println("Received JSON Body:", string(body)) */

	var rs struct {
		Hash string `json:"Hash"`
		Magnet string `json:Magnet`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rs); err != nil {
		return "", err
	}
	if rs.Magnet == "" {
		return "", fmt.Errorf("got empty magnet")
	}

	return rs.Magnet, nil

}
