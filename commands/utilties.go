package commands

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

// RemoteCmd for execution over SSH connection
type RemoteCmd struct {
	Host     string
	HostKey  string
	Username string
	Password string
	Cmd      string
}

// RemoteResult contains remote SSH execution
type RemoteResult struct {
	stdout string
	stderr string
}

func executeRemoteCmd(details RemoteCmd) RemoteResult {
	defer func() { //catch or finally
		if err := recover(); err != nil { //catch
			fmt.Fprintf(os.Stderr, "Exception: %v\n", err)
		}
	}()

	hostKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(details.HostKey))
	if err != nil {
		log.Printf("error parsing: %v", err)
	}

	config := &ssh.ClientConfig{
		User:            details.Username,
		Auth:            []ssh.AuthMethod{ssh.Password(details.Password)},
		HostKeyCallback: ssh.FixedHostKey(hostKey),
	}

	connectionString := fmt.Sprintf("%s:%s", details.Host, "22")
	conn, errConn := ssh.Dial("tcp", connectionString, config)
	if errConn != nil { //catch
		fmt.Fprintf(os.Stderr, "Exception: %v\n", errConn)
	}
	session, _ := conn.NewSession()
	defer session.Close()

	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	var stderrBuf bytes.Buffer
	session.Stderr = &stderrBuf
	session.Run(details.Cmd)

	errStr := ""
	if stderrBuf.String() != "" {
		errStr = strings.TrimSpace(stderrBuf.String())
	}

	return RemoteResult{stdoutBuf.String(), errStr}
}

func sendPayloadToJoinAPI(downloadFilename string, humanFilename string, icon string, smallIcon string) string {
	response := "Sorry, couldn't resend..."
	humanFilenameEnc := &url.URL{Path: humanFilename}
	humanFilenameEncoded := humanFilenameEnc.String()
	// NOW send this URL to the Join Push App API
	pushURL := "https://joinjoaomgcd.appspot.com/_ah/api/messaging/v1/sendPush"
	defaultParams := "?deviceId=d888b2e9a3a24a29a15178b2304a40b3&icon=" + icon + "&smallicon=" + smallIcon
	fileOnPhone := "&title=" + humanFilenameEncoded
	fileURL := spacesNamePublic + ".ams3.digitaloceanspaces.com/" + downloadFilename
	apiKey := "&apikey=" + joinAPIKey

	fileURLEnc := &url.URL{Path: fileURL}
	fileURL = fileURLEnc.String()
	completeURL := pushURL + defaultParams + apiKey + fileOnPhone + "&file=https://" + fileURL
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
