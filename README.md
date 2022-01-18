# Jenkins Pipeline Editor

[![Go Report Card](https://goreportcard.com/badge/github.com/MGSousa/go-jenkins-editor)](https://goreportcard.com/report/github.com/MGSousa/go-jenkins-editor)
![Actions Status](https://github.com/MGSousa/go-jenkins-editor/workflows/Release/badge.svg)

Editor for Jenkins declarative pipelines with Groovy and Shell highlighting

## Set your environment variables
```sh
cp .env{.example,}
```

## Docker
 - Build
```sh
docker build -t go-jenkins-editor . 
```

 - Run
```sh
export PORT=XXXX
docker run --rm -p $PORT:$PORT -d go-jenkins-editor -port $PORT -jenkinsUrl http://localhost:8080
```

 - After declaring arguments you can open in any Web Browser: *localhost:$PORT/pipeline/$PIPELINE_NAME*