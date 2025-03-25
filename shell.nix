let
    nixpkgs = fetchTarball "https://github.com/NixOS/nixpkgs/tarball/nixos-24.11";
    pkgs = import nixpkgs { config = {}; overlays = []; };
    isMacOS = pkgs.stdenv.hostPlatform.system == "darwin";
    remoteDockerHost = "ssh://ubuntu@192.168.0.30"; ## set this if you want to use remote docker, else set it to ""
in

pkgs.mkShellNoCC {
    packages = with pkgs; [
        # Optional packages
        pkgs.tree

        ## Added for golang project
        pkgs.go #version 1.23.6
        pkgs.delve # debugger for Go
        pkgs.air # hot reload for Go
        pkgs.clang # added for vscode Go extension to work
        pkgs.cyrus_sasl # added for vscode Go extension to work
        pkgs.direnv # direnv for project/shell specific env variables(see .envrc file)
        pkgs.commitlint # to validate git commit message

        ## Golang Tools
        pkgs.golangci-lint # https://golangci-lint.run/usage/linters/
        pkgs.lefthook # tool to run tasks on git hooks

        ## Added for golang testing
        pkgs.docker

    ] ++ (if pkgs.stdenv.isDarwin then [ 
        pkgs.darwin.iproute2mac pkgs.darwin.apple_sdk.frameworks.CoreFoundation 
    ] else []);

    shellHook = ''
        ## Add environment variables ##

        ### direnv ###
        eval "$(direnv hook bash)"
        if [ -f .envrc ]; then
            export DIRENV_LOG_FORMAT="" #for disabling log output from direnv
            echo ".envrc found. Allowing direnv..."
            direnv allow .
        fi

        ### Conditional set CGO_LDFLAGS ###
        if $isMacOS; then
            export CGO_CFLAGS="-mmacosx-version-min=13.0"
            export CGO_LDFLAGS="-mmacosx-version-min=13.0"
        fi

        if [ "${remoteDockerHost}" != "" ]; then
            if [[ "${remoteDockerHost}" == ssh://* ]]; then
                echo "Using remote Docker and setting ssh for the same"
                
                ## Extract username and IP from remoteDockerHost
                sshUser=$(echo "${remoteDockerHost}" | sed -E 's|ssh://([^@]+)@.*|\1|')
                sshIp=$(echo "${remoteDockerHost}" | sed -E 's|ssh://[^@]+@(.*)|\1|')
                
                ## Check if SSH access is possible to remotehost
                if ssh -o BatchMode=yes -o ConnectTimeout=5 -q $sshUser@$sshIp exit; then
                    echo "SSH access to $sshIp@$sshIp verified successfully."
                else
                    echo "Error: Unable to SSH into $username@$ip. Please check your SSH configuration and access permissions."
                    exit 1
                fi

                ## Start SSH agent if not already running
                if [ -z "$SSH_AGENT_PID" ]; then
                    eval $(ssh-agent -s)
                    echo "SSH agent started with PID $SSH_AGENT_PID"
                fi
                ## Add host ssh keys
                if $isMacOS; then
                    ssh-add --apple-use-keychain ~/.ssh/id_ed25519
                else
                    ssh-add ~/.ssh/id_ed25519
                fi

                ## Two options to connect to remote docker(both requires ssh access rights to remote docker)
                
                ### option1 : SSH tunneling to remote host for using its docker instance 
                rm -f /tmp/remote-docker-gotest.sock # remove remote-docker-gotest.sock if it already exists 
                ssh -f -N -L /tmp/remote-docker-gotest.sock:/var/run/docker.sock $sshUser@$sshIp # do the ssh tunneling
                export DOCKER_HOST=unix:///tmp/remote-docker-gotest.sock

                ### option2 : Set DOCKER_HOST using ssh url
                # make sure your remote docker has tcp enabled(potential security risk) because programs such as testcontainers is still using tcp to check docker status 
                #export DOCKER_HOST=${remoteDockerHost}
                
                ## Set testcontainers env variables
                export TESTCONTAINERS_HOST_OVERRIDE=$sshIp
                export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock
                #export TESTCONTAINERS_RYUK_DISABLED=true # disabling ryuk for now. Reason: ryuk port binding is failing for some reason
                
            else
                echo "Invalid remoteDockerHost format. Expected 'ssh://<username>@<ip>'."
                exit 1
            fi

        else
            echo "Using local Docker"
        fi
        # Ensure Docker is running(required for testcontainers etc)
        if ! docker info > /dev/null 2>&1; then
            echo "Docker is not running. Please check the Docker daemon."
        fi

    '';
}
