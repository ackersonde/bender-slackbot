package commands

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/ackersonde/bender-slackbot/filemanager"
	"github.com/rylio/ytdl"
)

func downloadYoutubeVideo(origURL string) bool {
	downloaded := false

	vid, err := ytdl.GetVideoInfo(context.Background(), origURL)
	if err == nil {
		client := ytdl.Client{
			HTTPClient: http.DefaultClient,
		}
		URI, err := client.GetDownloadURL(context.Background(), vid, vid.Formats[0])
		if err == nil {
			//Logger.Printf("preparing to download: %s\n", URI.String())

			uploadToPath := "/youtube/" + vid.Title + "." + vid.Formats[0].Extension
			tempPublicURL, err := filemanager.UploadInternetFileToDropbox(URI.String(), uploadToPath)
			if err != nil {
				Logger.Printf("%s %s\n", tempPublicURL, err.Error())
			} else {
				//Logger.Printf("Uploaded %s\n", tempPublicURL)
				tempPublicURL = strings.Replace(tempPublicURL, "dl=0", "dl=1", 1)
				icon := "https://emoji.slack-edge.com/T092UA8PR/youtube/a9a89483b7536f8a.png"
				smallIcon := "http://icons.iconarchive.com/icons/iconsmind/outline/16/Youtube-icon.png"

				filemanager.SendPayloadToJoinAPI(tempPublicURL, vid.Title, icon, smallIcon)
				downloaded = true
			}
		} else {
			Logger.Printf("ERR: %s\n", err.Error())
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
