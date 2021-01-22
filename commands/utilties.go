package commands

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
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
		Logger.Printf("ERR: %s", err.Error())
	}

	return string(out)
}

// WifiAction controls Wifi radio signal toggling (protect your sleep)
func WifiAction(param string) string {
	response := ":fritzbox: :wifi:\n"

	var out []byte
	var err error
	if param == "1" { // only turn on 2G/5G bands (not Guest WLAN)
		out, err = createFritzCmd("WLAN_2G", param).Output()
		response += string(out)
		out, err = createFritzCmd("WLAN_5G", "0").Output()
	} else if param == "5" {
		out, err = createFritzCmd("WLAN_5G", "1").Output()
		response += string(out)
	} else {
		out, err = createFritzCmd("WLAN", param).Output()
	}
	// TODO: implement `wgg` to enable guest wifi (""" WLAN_GUEST 1)

	if err != nil {
		Logger.Printf("ERR: %s", err.Error())
		response += "failed w/ " + err.Error()
	} else {
		response += string(out)
	}

	return response
}

func createFritzCmd(action string, param string) *exec.Cmd {
	return exec.Command("/app/fritzBoxShell.sh",
		"--boxip", os.Getenv("FRITZ_BOX_HOST"),
		"--boxuser", os.Getenv("FRITZ_BOX_USER"),
		"--boxpw", os.Getenv("FRITZ_BOX_PASS"),
		action, param)
}

var baseWireGuardCmd = "sudo kubectl exec -it $(sudo kubectl get po | grep wireguard | awk '{print $1; exit}' | tr -d \\n) -- bash -c"

func wireguardAction(action string) string {
	response := ":wireguard: "
	cmd := fmt.Sprintf("%s 'wg-quick %s wg0'", baseWireGuardCmd, action)
	Logger.Printf("cmd: %s", cmd)
	remoteResult := executeRemoteCmd(cmd, structures.PI4RemoteConnectConfig)

	if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
		response += remoteResult.Stderr
	} else {
		response += remoteResult.Stdout
	}

	return response
}

func wireguardShow() string {
	response := ":wireguard: "
	cmd := fmt.Sprintf("%s 'wg --version; wg'", baseWireGuardCmd)
	Logger.Printf("cmd: %s", cmd)
	remoteResult := executeRemoteCmd(cmd, structures.PI4RemoteConnectConfig)

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
// e.g. de-16.protonvpn.com or Big.Buck.Bunny.2007.1080p.x264-[opensrc.org].mp4
// are sent to Bender as <http://de-16.protonvpn.com|de-16.protonvpn.com> or
// Big.Buck.Bunny.2007.1080p.x264-\[<http://opensrc.org|opensrc.org>\].mp4
func scrubParamOfHTTPMagicCrap(sourceString string) string {
	if strings.Contains(sourceString, "<http") {
		// strip out url tags leaving just the text
		re := regexp.MustCompile(`<http.*\|(.*)>`)
		sourceString = re.ReplaceAllString(sourceString, `$1`)
	}

	return sourceString
}

func raspberryPIChecks() string {
	response := measureCPUTemp()
	response += getAppVersions()

	return response
}

func getAppVersions() string {
	result := "\n*APPs* :martial_arts_uniform:\n"

	hosts := []structures.RemoteConnectConfig{
		*structures.VPNPIRemoteConnectConfig,
		*structures.PI4RemoteConnectConfig}

	for _, host := range hosts {
		remoteResult := executeRemoteCmd("k3s --version | head -n 1", &host)

		result += "_" + host.HostName + "_: "
		if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
			result += remoteResult.Stderr
		} else {
			result += remoteResult.Stdout
			if strings.HasPrefix(host.HostName, "vpnpi") {
				result = strings.TrimRight(result, "\n") + ", "
				remoteResult := executeRemoteCmd("sudo docker --version", &host)
				result += remoteResult.Stdout
			}
		}
	}
	return result + "\n"
}

func measureCPUTemp() string {
	hosts := []structures.RemoteConnectConfig{
		*structures.BlondeBomberRemoteConnectConfig,
		*structures.VPNPIRemoteConnectConfig,
		*structures.PI4RemoteConnectConfig}

	result := "*CPUs* :thermometer:\n"
	for _, host := range hosts {
		measureCPUTempCmd := "((TEMP=`cat /sys/class/thermal/thermal_zone0/temp`/1000)); echo \"$TEMP\"C"
		if strings.HasPrefix(host.HostName, "blonde") {
			measureCPUTempCmd = "/usr/bin/sensors | grep Tctl | cut -d+ -f2"
		}

		remoteResult := executeRemoteCmd(measureCPUTempCmd, &host)
		if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
			result += "_" + host.HostName + "_: " + remoteResult.Stderr + "\n"
		} else {
			result += "_" + host.HostName + "_: *" + strings.TrimSuffix(remoteResult.Stdout, "\n") + "*\n"
		}
	}

	return result
}
