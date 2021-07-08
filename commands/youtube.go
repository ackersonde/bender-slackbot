package commands

import (
	"context"

	"github.com/kkdai/youtube/v2"
	"github.com/kkdai/youtube/v2/downloader"
)

var videoDownloader = func() (dl downloader.Downloader) {
	dl.OutputDir = syncthing
	dl.Debug = true
	return
}()

func downloadYoutubeVideo(videoID string) bool {
	downloaded := false

	client := youtube.Client{}

	video, err := client.GetVideo(videoID)
	if err != nil {
		panic(err)
	}

	err = videoDownloader.Download(context.Background(), video, &video.Formats[0], video.Title)
	if err == nil {
		downloaded = true
	}

	return downloaded
}
