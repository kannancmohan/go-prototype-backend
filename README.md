# go-prototype-backend
A prototype to build golang backend app's. It has the following integration ootb
* metrics - a metrics server that exposes an endpoint to be used by prometheus
* tracing - integration with grafana tempo to export trace data



## Tech Stack 
| Item                                       | version      | desc                                                |
| :----------------------------------------- | :----------: | --------------------------------------------------: |
| golang                                     |   1.23       |                                                     |
| testcontainers                             |   0.35.0     |  github.com/testcontainers/testcontainers-go        |
| prometheus                                 |   1.21.0     |  github.com/prometheus/client_golang                |
| opentelemetry                              |   1.34.0     |  go.opentelemetry.io/otel                           |

## Project Structure
```

```
## Project setup 

### Project Prerequisite 
* golang
* docker instance - required for testcontainers. check shell.nix for remote docker configuration
* delve - [optional] for debugging go projects
* air - [optional] for hot/live reloading go projects

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
golangci-lint run -v
```

To run only a specific linter(eg staticcheck)
```
golangci-lint run --no-config --disable-all --enable=staticcheck -v
```


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
