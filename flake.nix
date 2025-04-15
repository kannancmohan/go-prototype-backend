{
  description = "Development environment for go-prototype-backend";

  inputs = {
    nixpkgs.url = "github:Nixos/nixpkgs/nixos-24.11";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          config = {};
          overlays = [];
        };
        isMacOS = pkgs.stdenv.hostPlatform.isDarwin;
        remoteDockerHost = "ssh://ubuntu@192.168.0.30"; ## set this if you want to use remote docker, else set it to ""

        # Define the list of packages
        packages = with pkgs; [
          # Optional packages
          tree

          ## Added for golang project
          go #version 1.23.6
          delve # debugger for Go
          air # hot reload for Go
          clang # added for vscode Go extension to work
          cyrus_sasl # added for vscode Go extension to work
          direnv # direnv for project/shell specific env variables(see .envrc file)
          commitlint # to validate git commit message
          gitleaks # tool for detecting hardcoded secrets like password etc

          ## Golang Tools
          golangci-lint # https://golangci-lint.run/usage/linters/
          lefthook # tool to run tasks on git hooks

          ## Golang optional Tools
          goreleaser #tool to build and publish go binaries.

          ## Added for golang testing
          docker
        ] ++ lib.optionals isMacOS [
          pkgs.darwin.iproute2mac
          pkgs.darwin.apple_sdk.frameworks.CoreFoundation
        ];

        # Define the shell hook
        shellHook = ''

          ### direnv ###
          eval "$(direnv hook bash)"
          if [ -f .envrc ]; then
              export DIRENV_LOG_FORMAT="" #for disabling log output from direnv
              echo ".envrc found. Allowing direnv..."
              direnv allow .
          fi

          ### remote docker configuration ###
          if [ "${remoteDockerHost}" != "" ]; then
              if [[ "${remoteDockerHost}" == ssh://* ]]; then
                  echo "Using remote Docker and setting ssh for the same"

                  ## Extract username and IP from remoteDockerHost
                  sshUser=$(echo "${remoteDockerHost}" | sed -E 's|ssh://([^@]+)@.*|\1|')
                  sshIp=$(echo "${remoteDockerHost}" | sed -E 's|ssh://[^@]+@(.*)|\1|')

                  ## Check if SSH access is possible to remotehost
                  if ssh -o BatchMode=yes -o ConnectTimeout=5 -q $sshUser@$sshIp exit; then
                      echo "SSH access to $sshUser@$sshIp verified successfully."
                  else
                      echo "Error: Unable to SSH into $sshUser@$sshIp. Please check your SSH configuration and access permissions."
                      exit 1
                  fi

                  ## Start SSH agent if not already running
                  if [ -z "$SSH_AUTH_SOCK" ]; then
                      echo "No existing SSH agent detected. Starting a new one..."
                      eval $(ssh-agent -s)
                      echo "SSH agent started with PID $SSH_AGENT_PID"
                      # Attempt to add key only if we started a new agent
                      if [ "$IS_MACOS" = "true" ]; then
                          ssh-add --apple-use-keychain ~/.ssh/id_ed25519
                      else
                          ssh-add ~/.ssh/id_ed25519
                      fi
                  else
                      echo "Using existing SSH agent ($SSH_AUTH_SOCK)"
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
          ### Use IS_MACOS env var set by mkShell ###
          if [ "$IS_MACOS" = "true" ]; then
              export CGO_CFLAGS="-mmacosx-version-min=13.0"
              export CGO_LDFLAGS="-mmacosx-version-min=13.0"
          fi
        '';

      in
      {
        devShells.default = pkgs.mkShellNoCC {
          inherit packages shellHook;
          env = { IS_MACOS = builtins.toString isMacOS; }; # Pass isMacOS as env var
        };
        packages.default = pkgs.hello; # Add a default package
      }
    );
}