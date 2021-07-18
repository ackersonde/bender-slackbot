package structures

import (
	"encoding/base64"
	"os"
)

// RemoteConnectConfig provides structure for remote connections
type RemoteConnectConfig struct {
	User           string
	PrivateKeyPath string
	HostSSHKey     string
	HostPath       string
	HostName       string
}

// RemoteResult provides structure for stdout/err feedback
type RemoteResult struct {
	Err    error
	Stdout string
	Stderr string
}

// VPNPIRemoteConnectConfig connects to vpnpi
var VPNPIRemoteConnectConfig = &RemoteConnectConfig{
	User:           "ubuntu",
	PrivateKeyPath: "/root/.ssh/id_ed25519",
	HostSSHKey:     "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBCfXJ+mvHXs+t0+nF8JATxgMEwNngy6JCOVn1bEjsjsMylZsMejouArUrNKcnyPZ+vTvljlR7CaC6X9fbUtdxs0=",
	HostPath:       "/home/ubuntu/",
	HostName:       os.Getenv("SLAVE_HOSTNAME"),
}

// BlondeBomberRemoteConnectConfig connects to blonde-bomber
var BlondeBomberRemoteConnectConfig = &RemoteConnectConfig{
	User:           "ackersond",
	PrivateKeyPath: "/root/.ssh/id_ed25519",
	HostSSHKey:     "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBJBLGv3gAVIH2iM1I52Ckb2vnBKJtF+w1q3vVHxLY/J71v5edHrdr+ZpmegnpYdltJDsoJoVCD26MTXjWfJQbFg=",
	HostPath:       "/home/ackersond/",
	HostName:       os.Getenv("BUILD_HOSTNAME"),
}

// PI4RemoteConnectConfig connects to pi4
var PI4RemoteConnectConfig = &RemoteConnectConfig{
	User:           "ubuntu",
	PrivateKeyPath: "/root/.ssh/id_ed25519",
	HostSSHKey:     "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBC9wGQXT5zifmoWRaLeDrf/j98ShzZ29CilfVUVtSeKJp1k2uh8pMM/NTiG9FQQmitEIZXdwlcl2+Uj8YD21sAI=",
	HostPath:       "/home/ubuntu/",
	HostName:       os.Getenv("MASTER_HOSTNAME"),
}

var ackdeHostSSHKey, _ = base64.StdEncoding.DecodeString(os.Getenv("ACKDE_HOST_SSH_KEY_B64"))

//PrivateKeyPath: "/home/ackersond/go/src/github.com/ackersonde/bender-slackbot/tmp/id_ed25519_github_deploy",

// ACKDERemoteConnectConfig connects to ackerson.de
var ACKDERemoteConnectConfig = &RemoteConnectConfig{
	User:           "root",
	PrivateKeyPath: "/root/.ssh/id_ed25519",
	HostSSHKey:     "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBD4x8upKbzB81e9wtuv5vlHsC+jfzArRDRnhLTE6lCYXwq2Vfwo5Z/6iQe7bbfznfc+dLQ5pHImcTf4oON3SQ+s=",
	HostPath:       "/root",
	HostName:       "ackerson.de",
}
