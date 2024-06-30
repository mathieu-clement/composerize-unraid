package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/alessio/shellescape"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type SshHost struct {
	User           string
	Host           string
	Port           uint16
	PrivateKeyPath string
	KnownHostsPath string
}

type DockerContainer struct {
	id      string
	name    string
	inspect string
}

type SshSession struct {
	client *ssh.Client
}

func ComposerizeByIdOrName(host *SshHost, idOrName string) {
	client := connectViaSsh(host)
	defer client.Close()

	container := client.Inspect(idOrName)
	dockerRunCommand := runlike(container)
	composeFile := composerize(dockerRunCommand)

	fmt.Println(composeFile)
}

func ListById(host *SshHost) {
	client := connectViaSsh(host)
	defer client.Close()

	ids := client.GetContainerIds()

	for _, id := range ids {
		fmt.Println(id)
	}
}

func ListByName(host *SshHost) {
	client := connectViaSsh(host)
	defer client.Close()

	var names []string

	output := client.run("docker ps --format json")

	for _, line := range strings.Split(strings.TrimSuffix(output, "\n"), "\n") {
		var container map[string]interface{}
		err := json.Unmarshal([]byte(line), &container)
		if err != nil {
			log.Fatalf("Error unmarshalling JSON from ps (to list container names): %v", err)
		}

		name := container["Names"].(string)
		names = append(names, name)
	}

	sort.Slice(names, func(i, j int) bool {
		return strings.ToLower(names[i]) < strings.ToLower(names[j])
	})

	for _, name := range names {
		fmt.Println(name)
	}
}

func connectViaSsh(remote *SshHost) SshSession {
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

	return SshSession{client}
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
 * Methods of SshSession
 */

func (session SshSession) NewSession() *ssh.Session {
	_session, err := session.client.NewSession()
	if err != nil {
		log.Fatalf("unable to create SSH session: %v", err)
	}
	return _session
}

func (session SshSession) GetContainerIds() []string {
	return lines(session.run("docker ps -q -a"))
}

// Runs command
// Caller responsible to sanitize input
// Returns stdout as string
func (session SshSession) run(cmd string) string {
	_session := session.NewSession()
	defer _session.Close()
	output, err := _session.Output(cmd)
	if err != nil {
		log.Fatalf("unable to run command: %v", err)
	}
	return string(output)
}

func (session SshSession) Inspect(containerIdOrName string) DockerContainer {
	container := &DockerContainer{}
	container.inspect = session.run("docker inspect -f json " + shellescape.Quote(containerIdOrName))

	var results []map[string]interface{}
	err := json.Unmarshal([]byte(container.inspect), &results)
	if err != nil {
		log.Fatalf("Error unmarshalling JSON from docker inspect: %v", err)
	}
	result := results[0]
	container.id = result["Id"].(string)
	container.name = strings.TrimPrefix(result["Name"].(string), "/")

	return *container
}

func (session SshSession) Close() {
	session.client.Close()
}

// Execute "runlike" locally in a subprocess
// Returns docker run command (stdout as string)
func runlike(container DockerContainer) string {
	cmd := exec.Command("runlike", "--no-name", "--stdin")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatalf("error piping runlike: %v", err)
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, container.inspect)
	}()

	out, err := cmd.Output()
	if err != nil {
		log.Fatalf("error runliking: %v", err)
	}

	return string(out)
}

// Transform docker run command into docker-compose.yml
// using composerize in a subprocess
// First line will be a comment followed by the docker run command.
// Caller sanitizes input
// returns stdout as stirng
func composerize(dockerRunCommand string) string {
	cmd := exec.Command("composerize", strings.Split(dockerRunCommand, " ")...)

	out, err := cmd.Output()
	if err != nil {
		log.Fatalf("Error composerizing: %v", err)
	}

	lines := lines(string(out))
	var sb strings.Builder

	for _, line := range lines {
		if !strings.Contains(line, "<your project name>") {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func lines(in string) []string {
	return strings.Split(strings.TrimSuffix(in, "\n"), "\n")
}

func ParseFlags(host *SshHost) {
	// Flag options

	// --list           Lists containers by name
	// --ids            Lists containers by ID
	//
	// --id <ID>        Outputs docker-compose.yml contents to stdout,
	//                  for container with short identifier <ID>
	// --name <NAME>    Outputs docker-compose.yml contents to stdout,
	//                  for container with name <NAME>

	var list_by_name bool
	var list_by_id bool
	var id string
	var name string

	flag.BoolVar(&list_by_name, "list", false, "List containers by name")
	flag.BoolVar(&list_by_id, "ids", false, "List containers by id")
	flag.StringVar(&id, "id", "", "Output docker-compose.yml contents to stdout for container with id <ID>")
	flag.StringVar(&name, "name", "", "Output docker-compose.yml contents to stdout for container with name <NAME>")

	flag.Usage = PrintUsage

	// TODO add env vars or flags for Unraid IP addr, SSH details, etc.

	flag.Parse()

	// Mutually exclusive options
	if list_by_name && list_by_id {
		log.Fatal("Cannot list by both names (--names) and IDs (--ids)")
	}
	if id != "" && name != "" {
		log.Fatal("Use only one of: --id, --name")
	}

	switch {
	case list_by_name:
		ListByName(host)
	case list_by_id:
		ListById(host)
	case id != "":
		ComposerizeByIdOrName(host, id)
	case name != "":
		ComposerizeByIdOrName(host, name)
	default:
		fmt.Println("No flags provided, printing usage:")
		fmt.Println()
		flag.Usage()
	}
}

func PrintUsage() {
	fmt.Fprintln(os.Stderr, "Usage: composerize-unraid [options]\n")
	flag.PrintDefaults()
}

func main() {

	user := os.Getenv("USERNAME")
	if user == "" {
		user = "root"
	}

	host := os.Getenv("HOST")
	if host == "" {
		log.Fatal("HOST environment variable is required. Set to Unraid hostname or IP address.")
	}

	var port uint16 = 22
	envPort := os.Getenv("PORT")
	if envPort != "" {
		envPortInt, err := strconv.ParseUint(envPort, 10, 16)
		if err != nil {
			log.Fatal("error parsing PORT environment variable (allowed values 1-65,535)")
		}
		port = uint16(envPortInt)
	}

	sshHost := SshHost{
		User:           user,
		Host:           host,
		Port:           port,
		PrivateKeyPath: os.Getenv("HOME") + "/.ssh/id_ed25519",
		KnownHostsPath: os.Getenv("HOME") + "/.ssh/known_hosts",
	}

	ParseFlags(&sshHost)
}
