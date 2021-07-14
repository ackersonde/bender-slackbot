package commands

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/ackersonde/bender-slackbot/structures"
	"golang.org/x/crypto/ssh"
)

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

	cert, _ := ioutil.ReadFile(privateKeyPath + "-cert.pub")
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
		out, _ = createFritzCmd("WLAN_2G", param).Output()
		response += string(out)
		out, err = createFritzCmd("WLAN_5G", "0").Output()
	} else if param == "5" {
		out, err = createFritzCmd("WLAN_5G", "1").Output()
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

// CheckFirewallRules does a cross check of SSH access between
// digital ocean instance and home networks, ensuring minimal connectivity
func CheckFirewallRules() string {
	homeIPv6Prefix := fetchHomeIPv6Prefix()
	extras := fetchExtraDOsshFirewallRules(homeIPv6Prefix)

	response := ":do_droplet: "
	if len(extras) > 0 {
		response += "<https://cloud.digitalocean.com/networking/firewalls/" +
			os.Getenv("CTX_DIGITALOCEAN_FIREWALL") + "/rules|open to> -> " +
			strings.Join(extras, ", ") + " :rotating_light:"
	} else {
		response += "secured for " + homeIPv6Prefix + " :white_check_mark:"
	}

	response += "\n\n:house: "

	cmd := "nslookup -type=aaaa ackerson.de | grep Address | tail -n +2"
	domainIPv6Bytes, _ := exec.Command("/bin/sh", "-c", cmd).Output()

	domainIPv6 := string(bytes.Trim(domainIPv6Bytes, "\n"))
	domainIPv6 = strings.TrimPrefix(domainIPv6, "Address: ")
	homeFirewallRules := checkHomeFirewallSettings(domainIPv6)
	if len(homeFirewallRules) > 0 {
		response += "opened on -> " + strings.Join(homeFirewallRules, "\n") + " :rotating_light:"
	} else {
		response += "secured for " + domainIPv6 + " :white_check_mark:"
	}

	return response
}

func contains(arr [2]string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}

// checkHomeFirewallSettings returns addresses of ackerson.de
// and internal network has inbound SSH access
func checkHomeFirewallSettings(domainIPv6 string) []string {
	authorizedIPs := [2]string{domainIPv6, "192.168.178.0/24"}

	return retrieveHomeFirewallRules(authorizedIPs)
}

func fetchHomeIPv6Prefix() string {
	response := ""

	cmd := []string{"/home/ubuntu/fritzBoxShell.sh",
		"--boxip", os.Getenv("FRITZ_BOX_HOST"),
		"--boxuser", os.Getenv("FRITZ_BOX_USER"),
		"--boxpw", os.Getenv("FRITZ_BOX_PASS"),
		"IGDIP", "STATE"}

	remoteResult := executeRemoteCmd(strings.Join(cmd, " "), structures.PI4RemoteConnectConfig)
	if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
		response = remoteResult.Stderr
		Logger.Printf("ERR: %s", response)
	} else {
		re := regexp.MustCompile(`(.*)NewIPv6Prefix (?P<prefix>.*)\nNewPrefixLength (?P<length>.*)\n(.*)`)
		matches := re.FindAllStringSubmatch(remoteResult.Stdout, -1)
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
	}

	return response
}

func dockerInfo(application string) string {
	response := ""
	cmd := "docker logs -n 100 " + application
	if application == "" {
		cmd = "docker ps -a --format 'table {{.Names}}\t{{.Status}}'"
	}

	remoteResult := executeRemoteCmd(cmd, structures.ACKDERemoteConnectConfig)
	if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
		response += remoteResult.Stderr
	} else {
		response += remoteResult.Stdout
	}

	return response
}

func createFritzCmd(action string, param string) *exec.Cmd {
	return exec.Command("/home/ubuntu/fritzBoxShell.sh",
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
		*structures.BlondeBomberRemoteConnectConfig,
		*structures.VPNPIRemoteConnectConfig,
		*structures.PI4RemoteConnectConfig}

	for _, host := range hosts {
		result += "_" + host.HostName + "_: "
		remoteResult := executeRemoteCmd("sudo docker --version", &host)
		result += remoteResult.Stdout
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

func retrieveHomeFirewallRules(authorizedIPs [2]string) []string {
	hosts := []structures.RemoteConnectConfig{
		*structures.BlondeBomberRemoteConnectConfig,
		*structures.VPNPIRemoteConnectConfig,
		*structures.PI4RemoteConnectConfig}

	var result []string
	for _, host := range hosts {
		cmd := "sudo ufw status | grep 22 | awk '{print $3}'"
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
