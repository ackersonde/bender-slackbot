package commands

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"golang.org/x/crypto/ssh"
)

// RemoteCmd for execution over SSH connection
type RemoteCmd struct {
	Host          string
	Cmd           string
	ConnectConfig *ssh.ClientConfig
}

// RemoteResult contains remote SSH execution
type RemoteResult struct {
	err    error
	stdout string
	stderr string
}

var connectConfig = remoteConnectionConfiguration(piHostKey, "pi")

func remoteConnectionConfiguration(unparsedHostKey string, username string) *ssh.ClientConfig {
	hostKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(unparsedHostKey))
	if err != nil {
		log.Printf("error parsing: %v", err)
	}

	key, err := ioutil.ReadFile("/root/.ssh/id_rsa")
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Printf("Unable to parse private key: %v", err)
	}

	return &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.FixedHostKey(hostKey),
	}
}

func executeRemoteCmd(details RemoteCmd) RemoteResult {
	defer func() { //catch or finally
		if err := recover(); err != nil { //catch
			fmt.Fprintf(os.Stderr, "Exception: %v\n", err)
		}
	}()

	connectionString := fmt.Sprintf("%s:%s", details.Host, "22")
	conn, errConn := ssh.Dial("tcp", connectionString, connectConfig)
	if errConn != nil { //catch
		return RemoteResult{nil, "", errConn.Error()}
	}
	session, _ := conn.NewSession()
	defer session.Close()

	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	var stderrBuf bytes.Buffer
	session.Stderr = &stderrBuf
	err := session.Run(details.Cmd)

	errStr := ""
	if stderrBuf.String() != "" {
		errStr = strings.TrimSpace(stderrBuf.String())
	}

	return RemoteResult{err, stdoutBuf.String(), errStr}
}

func sendPayloadToJoinAPI(fileURL string, humanFilename string, icon string, smallIcon string) string {
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
	log.Printf("joinPushURL: %s\n", completeURL)
	resp, err := http.Get(completeURL)
	if err != nil {
		log.Printf("ERR: unable to call Join Push\n")
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		log.Printf("successfully sent payload to Join!\n")
		response = "Success!"
	}

	return response
}

// Slack gets excited when it recognizes a string that might be a URL
// e.g. de-16.protonvpn.com or Terminator.Dark.Fate.2019.1080p.WEBRip.x264-[YTS.LT].mp4
// are sent to Bender as <http://de-16.protonvpn.com|de-16.protonvpn.com> or
// Terminator.Dark.Fate.2019.1080p.WEBRip.x264-\[<http://YTS.LT|YTS.LT>\].mp4
// respectively
func scrubParamOfHTTPMagicCrap(sourceString string) string {
	if strings.Contains(sourceString, "<http") {
		// strip out url tags leaving just the text
		re := regexp.MustCompile(`<http.*\|(.*)>`)
		sourceString = re.ReplaceAllString(sourceString, `$1`)
	}

	return sourceString
}
