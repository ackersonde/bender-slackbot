package commands

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/ackersonde/bender-slackbot/structures"
	"github.com/bramvdbogaerde/go-scp"
	"github.com/bramvdbogaerde/go-scp/auth"
	"golang.org/x/crypto/ssh"
)

func scpRemoteConnectionConfiguration(config *structures.RemoteConnectConfig) scp.Client {
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

func retrieveClientConfig(config *structures.RemoteConnectConfig) *ssh.ClientConfig {
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

	certSigner := GetPublicCertificate("/root/.ssh/id_ed25519")

	return &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(certSigner)},
		HostKeyCallback: ssh.FixedHostKey(hostKey),
	}
}

// GetPublicCertificate retrieves it from the given privateKeyPath param
func GetPublicCertificate(privateKeyPath string) ssh.Signer {
	key, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		log.Printf("unable to read private key file: %v", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Printf("Unable to parse private key: %v", err)
	}

	cert, err := ioutil.ReadFile(privateKeyPath + "-cert.pub")
	pk, _, _, _, err := ssh.ParseAuthorizedKey(cert)
	if err != nil {
		log.Printf("unable to parse CA public key: %v", err)
		return nil
	}

	certSigner, err := ssh.NewCertSigner(pk.(*ssh.Certificate), signer)
	if err != nil {
		log.Printf("failed to create cert signer: %v", err)
		return nil
	}

	return certSigner
}

func getDeployFingerprint(deployCertFilePath string) string {
	out, err := exec.Command("/usr/bin/ssh-keygen", "-Lf", deployCertFilePath).Output()
	if err != nil {
		log.Println(err)
	}

	return string(out)
}

func wireguardAction(action string) string {
	response := ":wireguard: "
	cmd := fmt.Sprintf("sudo wg-quick %s wg0", action)
	Logger.Printf("cmd: %s", cmd)
	remoteResult := executeRemoteCmd(cmd, structures.BlondeBomberRemoteConnectConfig)

	if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
		response += remoteResult.Stderr
	} else {
		response += remoteResult.Stdout
	}

	return response
}

func wireguardShow() string {
	response := ":wireguard: "
	cmd := fmt.Sprintf("sudo wg")
	Logger.Printf("cmd: %s", cmd)
	remoteResult := executeRemoteCmd(cmd, structures.BlondeBomberRemoteConnectConfig)

	if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
		response += remoteResult.Stderr
	} else {
		response += remoteResult.Stdout
	}

	return response
}

func executeRemoteCmd(cmd string, config *structures.RemoteConnectConfig) structures.RemoteResult {
	defer func() { //catch or finally
		if err := recover(); err != nil { //catch
			Logger.Printf("Exception: %v\n", err)
		}
	}()

	remoteConfig := remoteConnectionConfiguration(config.HostSSHKey, config.User)
	if remoteConfig != nil && config.HostName != "" {
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

		return structures.RemoteResult{Err: err, Stdout: stdoutBuf.String(), Stderr: errStr}
	}

	return structures.RemoteResult{}
}

func initialDialOut(hostname string, remoteConfig *ssh.ClientConfig) *ssh.Client {
	connectionString := fmt.Sprintf("%s:%s", hostname, "22")
	sshClient, errConn := ssh.Dial("tcp", connectionString, remoteConfig)
	if errConn != nil { //catch
		Logger.Printf(errConn.Error())
	}

	return sshClient
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
	hosts := []structures.RemoteConnectConfig{
		*structures.BlondeBomberRemoteConnectConfig,
		*structures.VPNPIRemoteConnectConfig,
		*structures.PI4RemoteConnectConfig}

	response = measureCPUTemp(&hosts)
	apps := []string{"wg", "ipsec", "k3s"} // app order MUST match host above --^
	response += getAppVersions(apps, &hosts)

	return response
}

func getAppVersions(apps []string, hosts *[]structures.RemoteConnectConfig) string {
	result := "\n*APPs* :martial_arts_uniform:\n"
	for i, host := range *hosts {
		remoteResult := executeRemoteCmd(apps[i]+" --version", &host)
		if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
			result += host.HostName + ": " + remoteResult.Stderr
		} else {
			result += "_" + host.HostName + "_: " + remoteResult.Stdout + "\n"
		}
	}
	return result
}
func measureCPUTemp(hosts *[]structures.RemoteConnectConfig) string {
	measureCPUTempCmd := "((TEMP=`cat /sys/class/thermal/thermal_zone0/temp`/1000)); echo \"$TEMP\"C"

	result := "*CPUs* :thermometer:\n"
	for _, host := range *hosts {
		remoteResult := executeRemoteCmd(measureCPUTempCmd, &host)
		if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
			result += host.HostName + ": " + remoteResult.Stderr
		} else {
			if strings.TrimSpace(remoteResult.Stdout) == "C" {
				remoteResult = executeRemoteCmd("sensors | grep Tctl | awk '{print $2}'", &host)
			}
			result += "_" + host.HostName + "_: *" + strings.TrimSuffix(remoteResult.Stdout, "\n") + "*\n"
		}
	}

	return result
}
