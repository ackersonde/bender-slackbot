package commands

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/bramvdbogaerde/go-scp"
	"github.com/bramvdbogaerde/go-scp/auth"
	"golang.org/x/crypto/ssh"
)

// RemoteCmd for execution over SSH connection
type RemoteCmd struct {
	Host            string
	Cmd             string
	SourcePath      string
	DestinationPath string
	ConnectConfig   *ssh.ClientConfig
}

// RemoteConnectConfig for SSH activities
type RemoteConnectConfig struct {
	User           string
	PrivateKeyPath string
	HostEndpoints  []string
	HostSSHKey     string
	HostPath       string
	HostName       string
}

// RemoteResult contains remote SSH execution
type RemoteResult struct {
	err    error
	stdout string
	stderr string
}

var androidRCC = &RemoteConnectConfig{
	User:           "ackersond",
	PrivateKeyPath: "/root/.ssh/id_rsa_pix4x", // path must match K8S secret declaration in bender.yml
	HostEndpoints:  []string{"192.168.178.37:2222", "192.168.178.61:2222", "192.168.178.62:2222"},
	HostSSHKey:     "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHFhojNPu3wLn4NrLlyCQnLBCBkdGtYGYTl7IBfOefr05BKmq4WqBFt3U+hRmE9ti4xtjJw7Sz60qDkbuvpPt3c=",
	HostPath:       "/storage/emulated/0/Download/",
}

var vpnPIRemoteConnectConfig = &RemoteConnectConfig{
	User:           "ubuntu",
	PrivateKeyPath: "/Users/ackersond/.ssh/circleci_rsa",
	HostEndpoints:  []string{"192.168.178.59:22"},
	HostSSHKey:     "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBCfXJ+mvHXs+t0+nF8JATxgMEwNngy6JCOVn1bEjsjsMylZsMejouArUrNKcnyPZ+vTvljlR7CaC6X9fbUtdxs0=",
	HostPath:       "/home/ubuntu/",
	HostName:       "vpnpi.fritz.box",
}

var blackPearlRemoteConnectConfig = &RemoteConnectConfig{
	User:           "ubuntu",
	PrivateKeyPath: "/Users/ackersond/.ssh/circleci_rsa",
	HostEndpoints:  []string{"192.168.178.59:22"},
	HostSSHKey:     "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBD7p4FZyTPgywRBJ2ADL/i2igJ/N+3G8odFL3or3Ck77CVBnri8ZxO8+34/Rl/eGgt9qhp0vm7eTB4nE0C2m/Ro=",
	HostPath:       "/home/ubuntu/",
	HostName:       "blackpearl.fritz.box",
}

// SCPRemoteConnectionConfiguration returns scp client connection
func SCPRemoteConnectionConfiguration(config *RemoteConnectConfig) scp.Client {
	var client scp.Client

	clientConfig := retrieveClientConfig(config)
	if clientConfig != nil {
		// loop thru hostEndpoints until successful SCP connection
		for _, hostEndpoint := range config.HostEndpoints {
			client = scp.NewClient(hostEndpoint, clientConfig)
			// Connect to the remote server
			err := client.Connect()
			if err != nil {
				Logger.Printf("Couldn't establish a connection: %s", err)
			} else {
				break
			}
		}
	}
	return client
}

func retrieveClientConfig(config *RemoteConnectConfig) *ssh.ClientConfig {
	var clientConfig ssh.ClientConfig
	hostKey, _, _, _, err := ssh.ParseAuthorizedKey(
		[]byte(config.HostSSHKey))
	if err != nil {
		Logger.Printf("ERR: unable to parse HostKey -> %s", err)
		return &clientConfig
	}

	clientConfig, _ = auth.PrivateKey(
		config.User,
		config.PrivateKeyPath,
		ssh.FixedHostKey(hostKey))
	clientConfig.Timeout = 10 * time.Second // time to find SSH endpoint

	return &clientConfig
}

func remoteConnectionConfiguration(unparsedHostKey string, username string) *ssh.ClientConfig {
	hostKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(unparsedHostKey))
	if err != nil {
		Logger.Printf("error parsing: %v", err)
	}

	key, err := ioutil.ReadFile("/root/.ssh/id_rsa")
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		Logger.Printf("Unable to parse private key: %v", err)
	}

	return &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.FixedHostKey(hostKey),
	}
}

func wireguardAct(action string) string {
	response := ":wireguard: "
	cmd := fmt.Sprintf("sudo wg-quick %s wg0", action)
	Logger.Printf("cmd: %s", cmd)
	remoteResult := executeRemoteCmd(cmd, blackPearlRemoteConnectConfig)

	if remoteResult.stdout == "" && remoteResult.stderr != "" {
		response += remoteResult.stderr
	} else {
		response += remoteResult.stdout
	}

	return response
}

func wireguardShow() string {
	response := ":wireguard: "
	cmd := fmt.Sprintf("sudo wg show")
	Logger.Printf("cmd: %s", cmd)
	remoteResult := executeRemoteCmd(cmd, blackPearlRemoteConnectConfig)

	if remoteResult.stdout == "" && remoteResult.stderr != "" {
		response += remoteResult.stderr
	} else {
		response += remoteResult.stdout
	}

	return response
}

func executeRemoteCmd(cmd string, config *RemoteConnectConfig) RemoteResult {
	defer func() { //catch or finally
		if err := recover(); err != nil { //catch
			Logger.Printf("Exception: %v\n", err)
		}
	}()

	remoteConfig := remoteConnectionConfiguration(config.HostSSHKey, config.User)
	if config.HostName != "" {
		sshClient := initialDialOut(config.HostName, remoteConfig)

		session, _ := sshClient.NewSession()
		defer session.Close()

		var stdoutBuf bytes.Buffer
		session.Stdout = &stdoutBuf
		var stderrBuf bytes.Buffer
		session.Stderr = &stderrBuf
		err := session.Run(cmd)

		errStr := ""
		if stderrBuf.String() != "" {
			errStr = strings.TrimSpace(stderrBuf.String())
		}

		return RemoteResult{err, stdoutBuf.String(), errStr}
	}

	return RemoteResult{}
}

func initialDialOut(hostname string, remoteConfig *ssh.ClientConfig) *ssh.Client {
	connectionString := fmt.Sprintf("%s:%s", hostname, "22")
	sshClient, errConn := ssh.Dial("tcp", connectionString, remoteConfig)
	if errConn != nil { //catch
		Logger.Printf(errConn.Error())
	}

	return sshClient
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
	Logger.Printf("joinPushURL: %s\n", completeURL)
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
