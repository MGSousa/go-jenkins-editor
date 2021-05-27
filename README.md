# Jenkins Pipeline Editor

[![Go Report Card](https://goreportcard.com/badge/github.com/MGSousa/go-jenkins-editor)](https://goreportcard.com/report/github.com/MGSousa/go-jenkins-editor)
![Actions Status](https://github.com/MGSousa/go-jenkins-editor/workflows/Release/badge.svg)

Editor for Jenkins declarative and scripted pipelines with Groovy and Shell highlighting

## Docker
 - Build
```sh
docker build --build-arg JK_PORT=$PORT --build-arg JK_URL='XXX' --build-arg JK_USER="XXX" --build-arg JK_PASS="XXXXXXX" --build-arg JK_JOBSP=XXX -t go-jenkins-editor . 
```

 - Run
```sh
docker run --rm -p $PORT:$PORT -d go-jenkins-editor
```

 - Then you can open in any Web Browser: *localhost:$PORT/pipeline/$PIPELINE_NAME*