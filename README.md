# go-prototype-backend
A prototype to build golang backend app's.
OOTB features
* metrics - inbuilt metrics server that exposes metrics for prometheus
* tracing - supports integration with tracing backends like grafana tempo
* logging - supports integration with different logging framework(supports slog ootb)


## Tech Stack 
| Item                                       | version      | desc                                                |
| :----------------------------------------- | :----------: | --------------------------------------------------: |
| golang                                     |   1.23       |                                                     |
| testcontainers                             |   0.35.0     |  github.com/testcontainers/testcontainers-go        |
| prometheus                                 |   1.21.0     |  github.com/prometheus/client_golang                |
| opentelemetry                              |   1.34.0     |  go.opentelemetry.io/otel                           |

## Linting & Configuration Tools
| Item                      | config file          | version    | desc                                                |
| :-------------------------|:---------------------| :--------: | -------------------------------------------------------------------------: |
| direnv                    | .envrc               |   2.35.0   |  automatically loads/unloads environment variables based on the directory  |
| golangci-lint             | .golangci.yml        |   1.62.2   |  go linter that helps detect and fix coding issues  			 			      |

## Git hook tools
| Item                      | config file          | version    | desc                                                |
| :-------------------------|:---------------------| :--------: | -------------------------------------------------------------------------: |
| lefthook                  | lefthook.yml         |   1.8.1    |  git hooks manager to automate code quality checks, tests etc  			   |
| commitlint                | commitlint.config.js |   19.5.0   |  tool to enforce consistent commit message format based on custom rules  	|
| gitleaks                  |                      |   8.21.2   |  tool to detects hardcoded secrets in codebases to prevent accidental leaks|

## Release/Publish Tools (github)
| Item                      | config file          | version    | desc                                                |
| :-------------------------|:---------------------| :--------: | -------------------------------------------------------------------------: |
| semantic-release          | .releaserc.json      |   v4       |  tool to determines the next release version, generates release notes & changelog and publishes a release tag|
| goreleaser                | .goreleaser.yml      |   v6       |  generates tar file and publishes Docker images|


## Project Structure
```

```
## Project setup 

### Project Prerequisite 
* golang
* docker instance - required for testcontainers. check shell.nix for remote docker configuration
* delve - [optional] for debugging go projects
* air - [optional] for hot/live reloading go projects

### Development Environment Setup (using Nix Flakes)

This project uses Nix flakes to provide a consistent development environment with all necessary tools and dependencies.

**Prerequisites:**

*   **Nix Installation**: Ensure you have Nix installed on your system. Follow the instructions on the [official Nix website](https://nixos.org/download.html).
*   **Flakes Enabled (Required)**: Nix flakes are an experimental feature and must be enabled. Add or uncomment the following line in your Nix configuration file (usually `/etc/nix/nix.conf` for multi-user installs or `~/.config/nix/nix.conf` for single-user installs):
    ```
    experimental-features = nix-command flakes
    ```
    You may need to restart the Nix daemon after editing the configuration
    
    for mac `sudo launchctl kickstart -k system/org.nixos.nix-daemon`

    for linux `sudo systemctl restart nix-daemon` or similar

**Starting the Environment:**

1.  **Navigate to Project Directory**: Open your terminal and change to the root directory of this project.
2.  **Activate Environment**: Run the following command:
    ```bash
    nix develop
    # or
    # nix shell
    ```
    This command will build or download the environment defined in `flake.nix` and drop you into a new shell session with all the specified tools (Go, delve, air, linters, etc.) available in your `PATH`.

    *(Optional) Using Direnv:* If you have `direnv` installed and hooked into your shell, you can automate the environment activation. The included `.envrc` file contains `use flake`. Simply run `direnv allow .` once in the project directory. Afterwards, `direnv` will automatically load the Nix environment whenever you `cd` into the project directory.

**Updating Dependencies (flake.lock):**

The `flake.lock` file pins the exact versions of all dependencies (like `nixpkgs`). To update these dependencies to their latest compatible versions based on the inputs defined in `flake.nix`:

1.  Run the following command in the project root:
    ```bash
    nix flake update
    ```
2.  This command updates the `flake.lock` file.
3.  Commit the updated `flake.lock` file to your version control system (e.g., Git) to ensure all collaborators use the same updated dependencies.

Update the lock file periodically or when you need newer versions of the tools or libraries provided by the flake inputs.

### Project Initial setup

#### Init the module 
```
go mod init github.com/kannancmohan/go-prototype-backend
```

#### [optional] Init air for hot reloading
```
air init
```
adjust the generated '.air.toml' file to accommodate project specif changes

### Project Build & Execution

#### Project environment variables 

* For development environment:

     The env variables can be defined in .envrc file. The direnv tool will automatically load the env variables from .envrc file
     
     if you update the .envrc file on the fly, use command "direnv reload" to reload the env variables

#### App Build & Execution

##### Build App
```
make build
or
go build ./...
```

##### Run App
```
make run
or
go run cmd/api/*.go
```

##### Run App Test
```
make test
or
go test -v ./...
```

##### Run App Test (skip integration test)
```
make test-skip-integration-tests
or
go test -v -tags skip_integration_tests ./...
```
##### Run linters
```
make lint
or
golangci-lint run -v
```

To run only a specific linter(eg staticcheck)
```
golangci-lint run --no-config --disable-all --enable=staticcheck -v
```
##### Git hooks

###### Install lefthook (onetime setup)
```
go install github.com/evilmartians/lefthook@latest
```
lefthook can also be installed manually or via nixshell

###### Initialize your project (onetime setup)
```
lefthook install
```

###### Configure the lefthook.yml (onetime setup)
the "lefthook install" command will generate a lefthook.yml file. Update it accordingly

###### Skipping lefthook check
Add --no-verify flag to skip hook based check

Eg: to skip pre-commit lefthook check 
```
git commit -m "chore: some test message"  --no-verify
```

###### Dry-run lefthook commands
Eg: dry-run pre-commit hook commands
```
echo "feat: add new API" | lefthook run pre-commit
```

##### Auto build and deployment

###### To verify if .goreleaser.yaml is valid
```
goreleaser check
```
this requires installing goreleaser cli

#### Configuring Tracing 
##### Adding tracing to incoming http request 
Ensure that the TracerProvider is set globally or set in the handler 

Setting TracerProvider globally 
```
otel.SetTracerProvider(tp)
```

or

Setting TracerProvider in handler
```
otelhttp.NewHandler(yourHandler, "handle-request")
```

##### Adding tracing to outgoing http call
Configure the http client to use otelhttp.NewTransport 
```
		httpClient: &http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		}
```

Ensure that the outgoing call request is created using context 
```
http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost:%d/", t.externalPort), nil)
```

## GitHub Release Workflow

This project uses GitHub Actions and GoReleaser to automate the release process, including building and publishing Docker images to GitHub Container Registry (GHCR).

### Prerequisites

1. **GitHub Personal Access Token (PAT)**:
   - Go to GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)
   - Click "Generate new token" and select the following scopes:
     - `repo` (Full control of repositories)
     - `read:packages` (to download packages)
     - `write:packages` (to publish packages)
     - `delete:packages` (optional,to delete packages)
     - `workflow` (optional, for workflow interactions)
   - Copy the generated token

2. **Add PAT to Repository Secrets**:
   - Go to your repository → Settings → Secrets and variables → Actions
   - Click "New Environment' to create a new environment called 'Release'. (if environment not already present)
   - Click "Add environment secret"
   - Name: `GO_GITHUB_TOKEN`
   - Value: Paste your PAT token
   - Click "Add secret"

3. **Configure GitHub Repository Settings**:
   - Go to repository → Settings → Actions → General
   - Under "Workflow permissions," select "Read and write permissions"
   - (Optional) Enable "Allow GitHub Actions to create and approve pull requests"
   - Save changes

### Using the Release Workflow

1. **Creating a Release**:
   - **Option 1: Using Tags**
     - Create and push a tag to trigger the workflow:
     ```bash
     git tag v1.0.0  # Use semantic versioning
     git push origin v1.0.0
     ```
   
   - **Option 2: Using Semantic Commit Messages**
     - Make commits with appropriate semantic-release prefixes:
       - `feat:` for new features (bumps minor version)
       - `fix:` for bug fixes (bumps patch version)
       - `BREAKING CHANGE:` in commit body for breaking changes (bumps major version)
     - Push to the main branch to trigger the workflow:
     ```bash
     git push origin main
     ```
     - Semantic-release will analyze commit messages and automatically create a new release when appropriate

2. **Testing Locally**:
   - To test the release process locally without publishing:
   ```bash
   # Set environment variable for local testing
   export GITHUB_REPOSITORY="your-username/go-prototype-backend"
   
   # If you need to test Docker publishing (though not recommended for local testing)
   # export GO_GITHUB_TOKEN=your_github_token
   
   # Run GoReleaser in snapshot mode
   goreleaser release --snapshot --clean --config .github/config/.goreleaser.yml
   ```

3. **Verifying Configuration**:
   ```bash
   # Check if GoReleaser config is valid
   goreleaser check --config .github/config/.goreleaser.yml
   ```

### Docker Images

After a successful release, Docker images will be available at:
- `ghcr.io/your-username/go-prototype-backend:VERSION`
- `ghcr.io/your-username/go-prototype-backend:latest`


With both AMD64 and ARM64 architectures supported.

Also, check https://github.com/your-username?tab=packages