# Black Duck scans

Jenkins pipelines that run the nightly Black Duck scans for cordial. They were moved here from the `devops-tools` repository.

* `source/Jenkinsfile` scans the cordial source code and reports to the `Cordial-Source` project group.

* `docker/Jenkinsfile` triggers `BlackDuckScans/docker-scans` for the `gdna` image. It reports to the `Cordial-Containers` project group.

## Source scan

The source scan runs inside the image built from `source/Dockerfile`, which is a Temurin 21 JRE (needed by Detect) with Go copied in from `golang:latest`. Jenkins builds the image with `--pull`, so every run uses the latest stable Go. There is no Jenkins global Go tool to keep up to date.

On each run the pipeline:

1. Builds the agent image and checks out cordial.

2. Runs `go mod tidy` on the checkout so the scan sees a clean dependency graph. Nothing is pushed back to the repository.

3. Runs Black Duck Detect against the checkout, with `--detect.go.path` pointing at the image's Go.

`--detect.go.mod.dependency.types.excluded=UNUSED` drops modules that are only in the module graph and are never built. Without it, Black Duck reports entries such as `github.com/yuin/goldmark`, which reaches `go.sum` only through the `go.mod` of an old `golang.org/x/tools`.
