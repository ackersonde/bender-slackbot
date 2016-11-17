package commands

import (
	"fmt"
	"os"
	"os/exec"
)

func vpnTunnelChecks() bool {
	/* TODO: ensure vpn tunnel on RaspberryPI is up and working properly
		    0. `ssh pi@raspberrypi`
			  1. `curl ipinfo.io` (if this doesn't work, just `curl icanhazip.com`)
			  {
			  "ip": "85.159.233.103",
			  "hostname": "No Hostname",
			  "city": "",
			  "region": "",
			  "country": "NL",
			  "loc": "52.3824,4.8995",
			  "org": "AS43350 NForce Entertainment B.V."
			  }
			  -- verify "ip" != home.ackerson.de
			  -- verify "country" == "NL" (not possible if icanhazip request)

			  2. `sudo iptables -L OUTPUT -v --line-numbers | grep all`
			1    1030K  225M ACCEPT     all  --  any    tun0    anywhere             anywhere
			3    2288K 3237M ACCEPT     all  --  any    eth0    anywhere             192.168.178.0/24
			10   1381K  251M DROP       all  --  any    eth0    anywhere             anywhere
			  -- verify ACCEPT all -> any tun0 any -> any as *first* line
			  -- verify ACCEPT all -> any eth0 any -> 192.168.178.0/24 as *middle* line
			  -- verify DROP   all -> any eth0 any -> any as *last* line

	      3. if either of these checks fail, shutdown transmission daemon and send RED ALERT msg!
	*/

	return false
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
