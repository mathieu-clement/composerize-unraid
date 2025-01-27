# Generate docker-compose.yml from Unraid docker container

## Experimental

This is an experimental program that hasn't been tested for vulnerabilities.

USE AT YOUR OWN RISK.

It invokes commands on the local machine based on docker container metadata.
While efforts have been made to quote / escape variables when invoked
as shell arguments, there is no guarantee that malicious container metadata
might not cause denial of service or remote code execution.

Some of this escaping logic as it is pretty agressive results in invalid docker-compose.yml.

It is also not very flexible at this moment, with some details hardcoded
into the source code, such as SSH key exchange policies.

## Installation

1. Enable SSH access on Unraid:

    a. Unraid WebUI

    b. Settings

    c. Management Access

    d. User root: Manage

    e. On your local computer, copy the contents of all of the following
       files and append it to the "SSH authorized keys" text field on the
       Unraid Web UI, one per line:

         - ~/.ssh/id_rsa.pub (unsupported at this time)
         - ~/.ssh/id_ed25519.pub
         - ~/.ssh_ecdsa.pub (unsupported at this time)

       If you have neither, generate a new one with ssh-keygen.

2. Install **go** and dependencies, e.g. on macOS:

      ```
      brew install golang npm
      npm install -g composerize # or install only in current directory without -g flag
      pip3 install runlike
      ```

   or on Debian:

      ```
      sudo apt install golang npm pipx
      sudo npm install -g composerize # or install only in current directory without -g flag and sudo

      # Run as user that will execute composerize-unraid
      pipx install runlike
      pipx ensurepath # if needed, then re-login as prompted
      ```


   Ensure the runlike and composerize commands are available in the PATH:

     ```
     runlike --help
     composerize --help
     ```

3. Ensure the Unraid host is present in the ~/.ssh/known\_hosts file.
   If not, open a new session `ssh <user>@<host>`, and reply `y` when asked
   about authenticity.

4. `go build && ./composerize-unraid --name <NAME>`

## Usage

This will list the containers by name. We suppose Unraid is reachable at address 192.168.0.10
and we want to copy the configuraiton of 3 containers named duplicati, syslog-ng and watchtower
into a docker-compose.yml file in a folder bearing the name of the container inside of a new "output"
directory.
At the end, the tree command shows the created files.

```
./composerize-unraid --host 192.168.0.10 --list
mkdir output
for container in duplicati syslog-ng watchtower; do
  mkdir $container
  ./composerize-unraid --host 192.168.0.10 --name $container > ${container}/docker-compose.yml
done
tree output/
```

The YAML files should NOT be used as-is. They represent the exact configuration deployed on
the Unraid instance, but they're likely not what you want on another machine.

For example, the following parameters are set, which you probably want to change or remove altogether:

  - hostname
  - mac\_address
  - environment.HOST_*
  - network\_mode
  - working\_dir
  - logging
  - runtime
  - labels.`net.unraid.docker.manager`

Some of these parameters are set to default values, e.g. `network_mode` and `runtime`, 
so removing them will help readability.

Likewise the volume mappings will probably need to be adjusted since the host file system is different.

Finally you might want to group configurations together in a single docker-compose.yml for services
run as a group (e.g. an application and its database) and fine-tune the network configuration, including
removing ports needlessly exposed.

## How to help

If you want to help, there's lots to improve:

  - Fix bug of aggressive quoting / shell escaping resulting in invalid compose files
  - Support for other types of private keys
  - Go modules instead of single file
  - CI pipeline to create a distributable compiled version
  - Create a Homebrew tap
  - Pre-commit hook, including "go fmt".
  - Add flag to automatically remove some of the configuration above
