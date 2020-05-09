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

func scpRemoteConnectionConfiguration(config *remoteConnectConfig) scp.Client {
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

func retrieveClientConfig(config *remoteConnectConfig) *ssh.ClientConfig {
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

func wireguardAction(action string) string {
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

func executeRemoteCmd(cmd string, config *remoteConnectConfig) remoteResult {
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

		return remoteResult{err, stdoutBuf.String(), errStr}
	}

	return remoteResult{}
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

func raspberryPIChecks() string {
	response := ""
	hosts := []remoteConnectConfig{*blackPearlRemoteConnectConfig, *vpnPIRemoteConnectConfig, *pi4RemoteConnectConfig}

	response = measureCPUTemp(&hosts)
	apps := []string{"wg", "ipsec", "k3s"} // app order MUST match host above --^
	response += getAppVersions(apps, &hosts)

	return response
}

func getAppVersions(apps []string, hosts *[]remoteConnectConfig) string {
	result := "\n*APPs* :martial_arts_uniform:\n"
	for i, host := range *hosts {
		remoteResult := executeRemoteCmd(apps[i]+" --version", &host)
		if remoteResult.stdout == "" && remoteResult.stderr != "" {
			result += host.HostName + ": " + remoteResult.stderr
		} else {
			result += host.HostName + ": " + remoteResult.stdout + "\n"
		}
	}
	return result
}
func measureCPUTemp(hosts *[]remoteConnectConfig) string {
	measureCPUTempCmd := "((TEMP=`cat /sys/class/thermal/thermal_zone0/temp`/1000)); echo \"$TEMP\"C"

	result := "*CPUs* :thermometer:\n"
	for _, host := range *hosts {
		remoteResult := executeRemoteCmd(measureCPUTempCmd, &host)
		if remoteResult.stdout == "" && remoteResult.stderr != "" {
			result += host.HostName + ": " + remoteResult.stderr
		} else {
			result += host.HostName + ": *" + strings.TrimSuffix(remoteResult.stdout, "\n") + "*\n"
		}
	}

	return result
}
