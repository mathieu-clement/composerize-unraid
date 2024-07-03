package docker

import (
	"encoding/json"
    "fmt"
    "io"
    "log"
	"os/exec"
    "sort"
    "strings"

    "github.com/mathieu-clement/composerize-unraid/internal/ssh"
)

type Container struct {
	id      string
	name    string
	inspect string
}

type DockerInstance struct {
    Session ssh.Session
}

func (instance DockerInstance) Inspect(containerIdOrName string) Container {
	container := &Container{}
	container.inspect = instance.Session.Run("docker inspect -f json " + ssh.QuoteArg(containerIdOrName))

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

func (instance DockerInstance) ListById() {
	ids := instance.GetContainerIds()

	for _, id := range ids {
		fmt.Println(id)
	}
}

func (instance DockerInstance) ListByName() {
	var names []string

	output := instance.Session.Run("docker ps --format json")

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

func (instance DockerInstance) GetContainerIds() []string {
	return Lines(instance.Session.Run("docker ps -q -a"))
}

func (instance DockerInstance) ComposerizeByIdOrName(idOrName string) {
	container := instance.Inspect(idOrName)
	dockerRunCommand := container.Runlike()
	composeFile := Composerize(dockerRunCommand)

	fmt.Println(composeFile)
}


// Execute "runlike" locally in a subprocess
// Returns docker run command (stdout as string)
func (container Container) Runlike() string {
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
func Composerize(dockerRunCommand string) string {
	var sb strings.Builder
    sb.WriteString("# ")
    sb.WriteString(dockerRunCommand)
    sb.WriteString("\n\n")
	
    cmd := exec.Command("composerize", strings.Split(dockerRunCommand, " ")...)

	out, err := cmd.Output()
	if err != nil {
		log.Fatalf("Error composerizing: %v", err)
	}
	
    lines := Lines(string(out))

	for _, line := range lines {
		if !strings.Contains(line, "<your project name>") {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func Lines(in string) []string {
	return strings.Split(strings.TrimSuffix(in, "\n"), "\n")
}

