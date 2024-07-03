package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/mathieu-clement/composerize-unraid/internal/docker"
	"github.com/mathieu-clement/composerize-unraid/internal/ssh"
)

func ParseFlags() {
	// Flag options

	// --list           Lists containers by name
	// --ids            Lists containers by ID
	//
	// --id <ID>        Outputs docker-compose.yml contents to stdout,
	//                  for container with short identifier <ID>
	// --name <NAME>    Outputs docker-compose.yml contents to stdout,
	//                  for container with name <NAME>
	// --host <HOST>    Unraid hostname or IP address

	var list_by_name bool
	var list_by_id bool
	var id string
	var name string
	var hostname string

	flag.BoolVar(&list_by_name, "list", false, "List containers by name")
	flag.BoolVar(&list_by_id, "ids", false, "List containers by id")
	flag.StringVar(&id, "id", "", "Output docker-compose.yml contents to stdout for container with id <ID>")
	flag.StringVar(&name, "name", "", "Output docker-compose.yml contents to stdout for container with name <NAME>")
	flag.StringVar(&hostname, "host", os.Getenv("HOST"), "Unraid hostname or IP address. Overrides HOST environment variable.")

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

	// SSH host config
	user := os.Getenv("USERNAME")
	if user == "" {
		user = "root"
	}

	if hostname == "" {
		log.Fatal("--host parameter or HOST environment variable is required.")
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

	host := ssh.Host{
		User:           user,
		Host:           hostname,
		Port:           port,
		PrivateKeyPath: os.Getenv("HOME") + "/.ssh/id_ed25519",
		KnownHostsPath: os.Getenv("HOME") + "/.ssh/known_hosts",
	}

	session := ssh.Connect(&host)
	defer session.Close()

	dockerInstance := docker.DockerInstance{session}

	switch {
	case list_by_name:
		dockerInstance.ListByName()
	case list_by_id:
		dockerInstance.ListById()
	case id != "":
		dockerInstance.ComposerizeByIdOrName(id)
	case name != "":
		dockerInstance.ComposerizeByIdOrName(name)
	default:
		flag.Usage()
		fmt.Println()
		fmt.Println("No flags provided (typically --list or --id <CONTAINER_ID>).")
		os.Exit(1)
	}
}

func PrintUsage() {
	fmt.Fprintln(os.Stderr, "USAGE")
	fmt.Fprintln(os.Stderr, "  composerize-unraid [options]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "OPTIONS")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "ENVIRONMENT VARIABLES")
	fmt.Fprintln(os.Stderr, "To configure the SSH connection to the Unraid instance:")
	fmt.Fprintln(os.Stderr, "  - HOST (overridden by --host parameter)")
	fmt.Fprintln(os.Stderr, "  - PORT (defaults to 22 if unset)")
	fmt.Fprintln(os.Stderr, "  - USERNAME (defaults to 'root' if unset)")
}

func main() {
	ParseFlags()
}
