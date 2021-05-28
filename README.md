# Jenkins Pipeline Editor

[![Go Report Card](https://goreportcard.com/badge/github.com/MGSousa/go-jenkins-editor)](https://goreportcard.com/report/github.com/MGSousa/go-jenkins-editor)
![Actions Status](https://github.com/MGSousa/go-jenkins-editor/workflows/Release/badge.svg)

Editor for Jenkins declarative and scripted pipelines with Groovy and Shell highlighting

## Docker
 - Build
```sh
docker build -t go-jenkins-editor . 
```

 - Run
```sh
docker run --rm -p $PORT:$PORT -d go-jenkins-editor -h
```

 - After declaring args you can open in any Web Browser: *localhost:$PORT/pipeline/$PIPELINE_NAME*