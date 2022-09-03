package commands

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"

	"github.com/ackersonde/bender-slackbot/structures"
	"golang.org/x/crypto/ssh"
)

func remoteConnectionConfiguration(unparsedHostKey string, username string) *ssh.ClientConfig {
	privateKeyPath := "/root/.ssh/id_ed25519"
	hostKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(unparsedHostKey))
	if err != nil {
		Logger.Printf("error parsing: %v", err)
	}

	signer := GetPublicCertificate(privateKeyPath)

	if username == "root" { // TODO : figure out a better way to distinguish
		key, err := io.ReadFile(privateKeyPath)
		if err != nil {
			Logger.Printf("ROOT: unable to read private key: %v", err)
			return nil
		}

		// Create the Signer for this private key.
		signer, err = ssh.ParsePrivateKey(key)
		if err != nil {
			Logger.Printf("unable to parse private key: %v", err)
			return nil
		}
	}

	return &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.FixedHostKey(hostKey),
	}
}

// GetPublicCertificate retrieves it from the given privateKeyPath param
func GetPublicCertificate(privateKeyPath string) ssh.Signer {
	key, err := io.ReadFile(privateKeyPath)
	if err != nil {
		Logger.Printf("unable to read private key file: %v", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		Logger.Printf("Unable to parse private key: %v", err)
	}

	cert, _ := io.ReadFile(privateKeyPath + "-cert.pub")
	pk, _, _, _, err := ssh.ParseAuthorizedKey(cert)
	if err != nil {
		Logger.Printf("unable to parse CA public key: %v", err)
		return nil
	}

	certSigner, err := ssh.NewCertSigner(pk.(*ssh.Certificate), signer)
	if err != nil {
		Logger.Printf("failed to create cert signer: %v", err)
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

	if param == "1" || param == "0" {
		response += execFritzCmd("WLAN", param)
	} else {
		response += execFritzCmd("WLAN", "STATE")
	}

	return response
}

func getIPv6forHostname(hostname string) string {
	cmd := "nslookup -type=aaaa " + hostname + " | grep Address | tail -n +2"
	domainIPv6Bytes, _ := exec.Command("/bin/sh", "-c", cmd).Output()

	domainIPv6 := string(bytes.Trim(domainIPv6Bytes, "\n"))
	domainIPv6 = strings.TrimPrefix(domainIPv6, "Address: ")

	return domainIPv6
}

func contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}

// checkHomeFirewallSettings returns addresses of ackerson.de
// and internal network has inbound SSH access
func checkHomeFirewallSettings(domainIPv6 string, homeIPv6Prefix string) []string {
	authorizedIPs := []string{domainIPv6, homeIPv6Prefix, "192.168.178.0/24"}

	return retrieveHomeFirewallRules(authorizedIPs)
}

func fetchHomeIPv6Prefix() string {
	response := ""
	remoteResult := execFritzCmd("IGDIP", "STATE")

	re := regexp.MustCompile(`(.*)NewIPv6Prefix (?P<prefix>.*)\nNewPrefixLength (?P<length>.*)\n(.*)`)
	matches := re.FindAllStringSubmatch(remoteResult, -1)
	names := re.SubexpNames()

	m := map[string]string{}
	if len(matches) > 0 {
		for i, n := range matches[0] {
			m[names[i]] = n
		}
		if len(m) > 1 {
			response = m["prefix"] + "/" + m["length"]
		}
	}

	return response
}

func dockerInfo(application string) string {
	response := ""
	cmd := "ssh homepage \"docker  logs -n 100 " + application + "\""
	if application == "" {
		cmd = "ssh homepage \"docker ps -a --format 'table {{.Names}}\t{{.Status}}'\""
	}

	remoteResult := executeRemoteCmd(cmd, structures.PI4RemoteConnectConfig)
	if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
		response += remoteResult.Stderr
	} else {
		response += remoteResult.Stdout
	}

	return response
}

func execFritzCmd(action string, param string) string {
	response := ""

	cmd := fmt.Sprintf(
		"/home/ubuntu/fritzBoxShell.sh %s %s", action, param)

	remoteResult := executeRemoteCmd(cmd, structures.PI4RemoteConnectConfig)
	if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
		response = remoteResult.Stderr
	} else {
		response = remoteResult.Stdout
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
	sshClient, errConn := ssh.Dial("tcp6", connectionString, remoteConfig)
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
		result += "_" + host.HostName + "_: "
		remoteResult := executeRemoteCmd("docker --version", &host)
		result += remoteResult.Stdout
	}
	return result + "\n"
}

func measureCPUTemp() string {
	hosts := []structures.RemoteConnectConfig{
		*structures.VPNPIRemoteConnectConfig,
		*structures.PI4RemoteConnectConfig}

	result := "*CPUs* :thermometer:\n"
	for _, host := range hosts {
		// raspberryPIs
		measureCPUTempCmd := "((TEMP=`cat /sys/class/thermal/thermal_zone0/temp`/1000)); echo \"$TEMP\"C"
		if strings.HasPrefix(host.HostName, "build") {
			// AMD desktop
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

func retrieveHomeFirewallRules(authorizedIPs []string) []string {
	hosts := []structures.RemoteConnectConfig{
		*structures.VPNPIRemoteConnectConfig,
		*structures.PI4RemoteConnectConfig}

	var result []string
	for _, host := range hosts {
		cmd := "sudo ip6tables -L -n | grep 22 | awk '{print $3}'"
		res := executeRemoteCmd(cmd, &host)
		if res.Stdout == "" {
			result = append(result, "Firewall is NOT enabled for "+host.HostName+"!")
		} else {
			scanner := bufio.NewScanner(strings.NewReader(res.Stdout))
			interim := ""
			for scanner.Scan() {
				if !contains(authorizedIPs, scanner.Text()) {
					interim += scanner.Text() + "\t"
				}
			}
			if interim != "" {
				result = append(result, host.HostName+": "+interim)
			}
		}
	}

	return result
}
