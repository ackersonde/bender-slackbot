package commands

import (
	"bytes"
	"fmt"
	"log"
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
