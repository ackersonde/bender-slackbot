package filemanager

import (
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox"
	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox/files"
)

var joinAPIKey = os.Getenv("CTX_JOIN_API_KEY")

// Logger to give senseful settings
var Logger = log.New(os.Stdout, "", log.LstdFlags)

var dropboxAccessToken = os.Getenv("CTX_DROPBOX_ACCESS_TOKEN")
var chunkSize = int64(1 << 27) // ~138 MB

// UploadInternetFileToDropbox sends contents of url to public dropbox folder
func UploadInternetFileToDropbox(
	downloadFromURL string,
	uploadToPath string) (tempPublicURL string, err error) {

	config := dropbox.Config{
		Token: dropboxAccessToken,
	}
	dbx := files.New(config)

	// https://www.dropbox.com/developers/documentation/http/documentation
	// https://github.com/dropbox/dropbox-sdk-go-unofficial/blob/master/dropbox/files/types.go

	res, err := http.Head(downloadFromURL)
	if err != nil {
		return downloadFromURL, err
	} else if res.ContentLength <= 0 {
		return downloadFromURL, errors.New("<= cowardly refusing to transfer empty file")
	}

	Logger.Printf("File size: %s bytes\n", strconv.FormatInt(res.ContentLength, 10))

	resp, err := http.Get(downloadFromURL)
	if err != nil {
		Logger.Printf("ERR: %s\n", err.Error())
	}
	defer resp.Body.Close()

	err = nil
	// if video is > 1<<27 ()
	if res.ContentLength > chunkSize {
		commitInfo := files.NewCommitInfo(uploadToPath)
		err = uploadChunked(dbx, resp.Body, commitInfo, res.ContentLength)
	} else if res.ContentLength > 0 {
		commitInfo := files.NewCommitInfo(uploadToPath)
		_, err = dbx.Upload(commitInfo, resp.Body)
	}

	if err == nil {
		filesMetaData, err := dbx.GetTemporaryLink(
			files.NewGetTemporaryLinkArg(uploadToPath))
		return filesMetaData.Link, err
	}

	return downloadFromURL, err
}

// https://github.com/mschneider82/sharecmd/blob/master/provider/dropbox/dropbox.go
func uploadChunked(dbx files.Client, r io.Reader, commitInfo *files.CommitInfo, sizeTotal int64) (err error) {
	res, err := dbx.UploadSessionStart(files.NewUploadSessionStartArg(),
		&io.LimitedReader{R: r, N: chunkSize})
	if err != nil {
		return err
	}

	written := chunkSize

	for (sizeTotal - written) > chunkSize {
		cursor := files.NewUploadSessionCursor(res.SessionId, uint64(written))
		args := files.NewUploadSessionAppendArg(cursor)

		err = dbx.UploadSessionAppendV2(args, &io.LimitedReader{R: r, N: chunkSize})
		if err != nil {
			return err
		}
		written += chunkSize
	}

	cursor := files.NewUploadSessionCursor(res.SessionId, uint64(written))
	args := files.NewUploadSessionFinishArg(cursor, commitInfo)

	if _, err = dbx.UploadSessionFinish(args, r); err != nil {
		return err
	}

	return nil
}

// DownloadFileToPhone accepts a URL and filename to upload to a public dropbox
// folder and then invoke JoinAPI to download to mobile
func DownloadFileToPhone(url string, filename string) bool {
	result := false

	tempPublicURL, err := UploadInternetFileToDropbox(url, "/other/"+filename)
	if err != nil {
		Logger.Printf("%s %s\n", tempPublicURL, err.Error())
	} else {
		//Logger.Printf("Uploaded %s\n", tempPublicURL)
		tempPublicURL = strings.Replace(tempPublicURL, "dl=0", "dl=1", 1)
		icon := "https://upload.wikimedia.org/wikipedia/commons/9/99/1328101811_Download.png"
		smallIcon := "https://upload.wikimedia.org/wikipedia/commons/thumb/1/11/Breathe-folder-download.svg/128px-Breathe-folder-download.svg.png"

		SendPayloadToJoinAPI(tempPublicURL, filename, icon, smallIcon)
		result = true
	}

	return result
}

// SendPayloadToJoinAPI sends publically accessible file url to Join which pushes to mobile
func SendPayloadToJoinAPI(fileURL string, humanFilename string, icon string, smallIcon string) string {
	response := "Sorry, couldn't resend..."
	humanFilenameEnc := &url.URL{Path: humanFilename}
	humanFilenameEncoded := humanFilenameEnc.String()
	// NOW send this URL to the Join Push App API
	pushURL := "https://joinjoaomgcd.appspot.com/_ah/api/messaging/v1/sendPush"
	defaultParams := "?deviceId=d888b2e9a3a24a29a15178b2304a40b3&icon=" + icon + "&smallicon=" + smallIcon
	fileOnPhone := "&title=" + humanFilenameEncoded
	apiKey := "&apikey=" + joinAPIKey

	completeURL := pushURL + defaultParams + apiKey + fileOnPhone + "&file=" + fileURL
	// Get the data
	resp, err := http.Get(completeURL)
	if err != nil {
		Logger.Printf("ERR: unable to call Join Push\n")
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		Logger.Printf("successfully sent payload to Join!\n")
		response = "Success!"
	}

	return response
}
