# Generate docker-compose.yml for all Unraid Docker containers

## Experimental

This is an experimental program that hasn't been tested for vulnerabilities.

USE AT YOUR OWN RISK.

It invokes commands on the local machine based on docker container metadata.
While efforts have been made to quote / escape variables when invoked
as shell arguments, there is no guarantee that malicious container metadata
might not cause denial of service or remote code execution.

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
          brew install golang
          pip3 install runlike
          brew install npm
          npm install -g composerize
          ```

          Ensure the runlike command is available in the path.

3. Ensure the Unraid host is present in the ~/.ssh/known\_hosts file.
   If not, open a new session `ssh <user>@<host>`, and reply `y` when asked
   about authenticity.

4. `go run main.go -- -name <NAME>`

## Usage

```
go run main.go -- --help
```


