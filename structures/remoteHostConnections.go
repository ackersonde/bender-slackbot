package structures

// RemoteConnectConfig provides structure for remote connections
type RemoteConnectConfig struct {
	User           string
	PrivateKeyPath string
	HostEndpoints  []string
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

// AndroidRCC connects to phone
var AndroidRCC = &RemoteConnectConfig{
	User:           "ackersond",
	PrivateKeyPath: "/root/.ssh/id_rsa_pix4x", // path must match K8S secret declaration in bender.yml
	HostEndpoints:  []string{"192.168.178.73:2222", "192.168.178.63:2222", "192.168.178.37:2222"},
	HostSSHKey:     "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBHFhojNPu3wLn4NrLlyCQnLBCBkdGtYGYTl7IBfOefr05BKmq4WqBFt3U+hRmE9ti4xtjJw7Sz60qDkbuvpPt3c=",
	HostPath:       "/storage/emulated/0/Download/",
}

// VPNPIRemoteConnectConfig connects to vpnpi
var VPNPIRemoteConnectConfig = &RemoteConnectConfig{
	User:           "ubuntu",
	PrivateKeyPath: "/Users/ackersond/.ssh/circleci_rsa",
	HostEndpoints:  []string{"192.168.178.59:22"},
	HostSSHKey:     "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBCfXJ+mvHXs+t0+nF8JATxgMEwNngy6JCOVn1bEjsjsMylZsMejouArUrNKcnyPZ+vTvljlR7CaC6X9fbUtdxs0=",
	HostPath:       "/home/ubuntu/",
	HostName:       "vpnpi.fritz.box",
}

// BlondeBomberRemoteConnectConfig connects to blonde-bomber
var BlondeBomberRemoteConnectConfig = &RemoteConnectConfig{
	User:           "ackersond",
	PrivateKeyPath: "/Users/ackersond/.ssh/circleci_rsa",
	HostEndpoints:  []string{"192.168.178.20:22"},
	HostSSHKey:     "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBJBLGv3gAVIH2iM1I52Ckb2vnBKJtF+w1q3vVHxLY/J71v5edHrdr+ZpmegnpYdltJDsoJoVCD26MTXjWfJQbFg=",
	HostPath:       "/home/ackersond/",
	HostName:       "blonde-bomber.fritz.box",
}

// BlackPearlRemoteConnectConfig connects to blackpearl
var BlackPearlRemoteConnectConfig = &RemoteConnectConfig{
	User:           "ubuntu",
	PrivateKeyPath: "/Users/ackersond/.ssh/circleci_rsa",
	HostEndpoints:  []string{"192.168.178.59:22"},
	HostSSHKey:     "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBD7p4FZyTPgywRBJ2ADL/i2igJ/N+3G8odFL3or3Ck77CVBnri8ZxO8+34/Rl/eGgt9qhp0vm7eTB4nE0C2m/Ro=",
	HostPath:       "/home/ubuntu/",
	HostName:       "blackpearl.fritz.box",
}

// PI4RemoteConnectConfig connects to pi4
var PI4RemoteConnectConfig = &RemoteConnectConfig{
	User:           "ubuntu",
	PrivateKeyPath: "/root/.ssh/id_ed25519",
	HostEndpoints:  []string{"192.168.178.29:22"},
	HostSSHKey:     "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBC9wGQXT5zifmoWRaLeDrf/j98ShzZ29CilfVUVtSeKJp1k2uh8pMM/NTiG9FQQmitEIZXdwlcl2+Uj8YD21sAI=",
	HostPath:       "/home/ubuntu/",
	HostName:       "pi4.fritz.box",
}
