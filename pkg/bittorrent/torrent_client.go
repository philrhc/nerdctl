package bittorrent

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

func get(magnet string) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", "http://localhost:8080/data?magnet="+magnet, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to download; status code: %v", resp.StatusCode)
	}

	return resp.Body, nil
}

func seed(r io.Reader, identifier string) (string, error) {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	contentType := mw.FormDataContentType()
	go func() {
		fw, err := mw.CreateFormFile("file", identifier)
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
