package ssh

import (
    "fmt"
	"io/ioutil"
    "log"

	"github.com/alessio/shellescape"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type Host struct {
	User           string
	Host           string
	Port           uint16
	PrivateKeyPath string
	KnownHostsPath string
}

type Session struct {
	client *ssh.Client
}

func Connect(remote *Host) Session {
	signer := getPrivateKeySigner(remote.PrivateKeyPath)

	config := &ssh.ClientConfig{
		User: remote.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyAlgorithms: []string{ssh.KeyAlgoED25519},
		HostKeyCallback:   getKnownHostKeyCallback(remote.KnownHostsPath),
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", remote.Host, remote.Port), config)
	if err != nil {
		log.Fatalf("unable to connect to SSH server: %v.\nDid you set USERNAME? (defaults to root) Did you configure the local SSH public key on Unraid?", err)
	}

	return Session{client}
}

func getPrivateKeySigner(privateKeyPath string) ssh.Signer {
	key, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		log.Fatalf("unable to read SSH private key: %v", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatalf("unable to parse SSH private key: %v", err)
	}
	return signer
}

func getKnownHostKeyCallback(knownHostsPath string) ssh.HostKeyCallback {
	callback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		log.Fatalf("unable to read known_hosts file: %v", err)
	}
	return callback
}

/*
 * Methods of Session
 */

func (session Session) NewSession() *ssh.Session {
	_session, err := session.client.NewSession()
	if err != nil {
		log.Fatalf("unable to create SSH session: %v", err)
	}
	return _session
}

// Runs command
// Caller responsible to sanitize input
// Returns stdout as string
func (session Session) Run(cmd string) string {
	_session := session.NewSession()
	defer _session.Close()
	output, err := _session.Output(cmd)
	if err != nil {
		log.Fatalf("unable to run command: %v", err)
	}
	return string(output)
}

func QuoteArg(argument string) string {
    return shellescape.Quote(argument)
}

func (session Session) Close() {
	session.client.Close()
}
