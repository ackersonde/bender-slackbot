package commands

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Andreychik32/ytdl"
)

func downloadYoutubeVideo(origURL string) bool {
	downloaded := false

	vid, err := ytdl.GetVideoInfo(context.Background(), origURL)
	if err == nil {
		client := ytdl.Client{
			HTTPClient: http.DefaultClient,
		}

		filepath := syncthing + vid.Title + "." + vid.Formats[0].Extension
		f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			Logger.Printf("ERR: unable to write %s\n", filepath)
		} else {
			defer f.Close()
			err := client.Download(context.Background(), vid, vid.Formats[0], f)
			if err != nil {
				Logger.Printf("ERR: %s\n", err.Error())
			} else {
				downloaded = true
			}
		}
	} else {
		Logger.Printf("ERR: %s\n", err.Error())
	}

	return downloaded
}

func findVideoOnYoutube(fetchURL *url.URL) (*url.URL, string) {
	vid, err := ytdl.GetVideoInfo(context.Background(), fetchURL)
	if err != nil {
		Logger.Printf("ERR: ytdl GetVideoInfo: %s", err.Error())
	}
	youtubeClient := ytdl.Client{
		HTTPClient: http.DefaultClient,
	}
	foundURL, errB := youtubeClient.GetDownloadURL(context.Background(), vid, vid.Formats[0])
	if errB != nil {
		Logger.Printf("ERR: ytdl GetDownloadURL %s", errB.Error())
	}

	destination := strings.ReplaceAll(vid.Title+"."+vid.Formats[0].Extension, " ", "_")
	destination = strings.Replace(destination, "(", "", -1)
	destination = strings.Replace(destination, ")", "", -1)
	destination = strings.Replace(destination, ",", "", -1)
	destination = strings.Replace(destination, "..", ".", -1)

	return foundURL, destination
}
