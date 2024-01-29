# filestore-server
Includes code artefacts to launch a simple filestore HTTP server.

![DeploymentArchitetcure.drawio.png]docs/DeploymentArchitetcure.drawio.png

# Notes:
- This being a simple demo version, the http server is hardcoded to be exposed on `localhost` and port `8080`

# Steps to initialize the fileserver directly on a instance
1. clone this repo and make sure that the working directory has main.go.
2. run `$./server` on windows and `$server` (This is the ready compiled version )
NOTE: To run the server locally in dev mode `$go run main.go` or to generate another version of executable run `$go build -o <desired_executable_name> main.go`


# Create & run a docker container
Pre-requisite: Install docker to the instance and start docker
1. clone this repo and make sure that the working directory has main.go and the Dockerfile.
2. run `$docker build -t go-server .` to build the docker image
3. run `$docker run -p 8080:8080 go-server`

# Deploy the image a Kubernetes deployment
run `$kubectl apply -f k8_deployment.yaml`