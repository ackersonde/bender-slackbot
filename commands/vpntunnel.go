package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

func executeRemoteCmd(command string, config *ssh.ClientConfig) string {
	defer func() { //catch or finally
		if err := recover(); err != nil { //catch
			fmt.Fprintf(os.Stderr, "Exception: %v\n", err)
		}
	}()

	connectionString := fmt.Sprintf("%s:%s", raspberryPIIP, "22")
	fmt.Println("SSH to " + connectionString)
	conn, errConn := ssh.Dial("tcp", connectionString, config)
	if errConn != nil { //catch
		fmt.Fprintf(os.Stderr, "Exception: %v\n", errConn)
	}
	session, _ := conn.NewSession()
	defer session.Close()

	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Run(command)

	return stdoutBuf.String()
}

// ensure the PrivateTunnel vpn connection on PI is up and working properly
func raspberryPIPrivateTunnelChecks() string {
	tunnelUp := ""
	piPass := os.Getenv("piPass")
	piUser := os.Getenv("piUser")

	sshConfig := &ssh.ClientConfig{
		User: piUser,
		Auth: []ssh.AuthMethod{ssh.Password(piPass)},
	}

	// `curl ipinfo.io` (if this doesn't work, just `curl icanhazip.com`)
	results := make(chan string, 10)
	timeout := time.After(2 * time.Second)
	go func() {
		results <- executeRemoteCmd("curl ipinfo.io", sshConfig)
	}()

	type IPInfoResponse struct {
		IP      string
		Country string
	}
	var jsonRes IPInfoResponse

	select {
	case res := <-results:
		if res != "" {
			err := json.Unmarshal([]byte(res), &jsonRes)
			if err != nil {
				fmt.Printf("unable to parse JSON string %s\n", res)
			}
			if jsonRes.Country == "NL" {
				resultsDig := make(chan string, 10)
				timeoutDig := time.After(2 * time.Second)
				// ensure home.ackerson.de is DIFFERENT than PI IP address!
				go func() {
					resultsDig <- executeRemoteCmd("dig +short home.ackerson.de | tail -n1", sshConfig)
				}()
				select {
				case resComp := <-resultsDig:
					if resComp != jsonRes.IP {
						tunnelUp = jsonRes.IP
					}
				case <-timeoutDig:
					fmt.Println("Timed out on dig home.ackerson.de!")
				}
			}
		}
	case <-timeout:
		fmt.Println("Timed out on curl ipinfo.io!")
	}

	// Tunnel should be OK. Now double check iptables to ensure that
	// ALL Internet requests are running over OpenVPN!
	if tunnelUp != "" {
		resultsIPTables := make(chan string, 10)
		timeoutIPTables := time.After(2 * time.Second)
		// ensure home.ackerson.de is DIFFERENT than PI IP address!
		go func() {
			resultsIPTables <- executeRemoteCmd("sudo iptables -L OUTPUT -v --line-numbers | grep all", sshConfig)
		}()
		select {
		case resIPTables := <-resultsIPTables:
			lines := strings.Split(resIPTables, "\n")

			for idx, oneLine := range lines {
				switch idx {
				case 0:
					if !strings.Contains(oneLine, "ACCEPT     all  --  any    tun0    anywhere") {
						tunnelUp = ""
					}
				case 1:
					if !strings.Contains(oneLine, "ACCEPT     all  --  any    eth0    anywhere             192.168.178.0") {
						tunnelUp = ""
					}
				case 2:
					if !strings.Contains(oneLine, "DROP       all  --  any    eth0    anywhere") {
						tunnelUp = ""
					}
				}
			}
		case <-timeoutIPTables:
			fmt.Println("Timed out on `iptables -L OUTPUT`!")
		}
		//  TODO if tunnelUp = "" shutdown transmission daemon, restart VPN and send RED ALERT msg!
	}

	return tunnelUp
}

func vpnTunnelCmds(command ...string) string {
	if command[0] != "status" {
		cmd := exec.Command(command[0])

		args := len(command)
		if args > 1 {
			cmd = exec.Command(command[0], command[1])
		}

		errStart := cmd.Start()
		if errStart != nil {
			os.Stderr.WriteString(errStart.Error())
		}

		if errWait := cmd.Wait(); errWait != nil {
			fmt.Println(errWait)
		}
	}

	/* Here's the next cmd to get setup
			# ip a show tun0
			9: tun0: <POINTOPOINT,MULTICAST,NOARP,UP,LOWER_UP> mtu 1024 qdisc pfifo_fast state UNKNOWN group default qlen 500 link/none
	    		inet 192.168.178.201/32 scope global tun0
	       	valid_lft forever preferred_lft forever
			# vpnc-disconnect
				Terminating vpnc daemon (pid: 174)
			# ip a show tun0
				Device "tun0" does not exist.
	*/
	tun0StatusCmd := "/sbin/ip a show tun0 | /bin/grep tun0 | /usr/bin/tail -1"
	tunnel, err := exec.Command("/bin/bash", "-c", tun0StatusCmd).Output()
	if err != nil {
		fmt.Printf("Failed to execute command: %s", tun0StatusCmd)
	}

	tunnelStatus := string(tunnel)
	if len(tunnelStatus) == 0 {
		tunnelStatus = "Tunnel offline."
	}

	return ":closed_lock_with_key: " + tunnelStatus
}
