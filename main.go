package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"github.com/MGSousa/go-generator"
	"github.com/MGSousa/go-jenkins-editor/cache"
	"github.com/antchfx/xmlquery"
	"github.com/kataras/iris/v12"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strings"
)

var (
	httpPort 	= flag.Int("port", 5000, "UI Port")
	jenkinsUrl 	= flag.String("jenkinsUrl", "http://192.168.233.165:8080", "Jenkins Base Url")
	username 	= flag.String("username", "", "Jenkins Username")
	password 	= flag.String("password", "", "Jenkins Password")
	cacheProv 	= flag.String("cacheProvider", "redis", "Internal Cache Provider (redis)")
	jobsPrefix	string
)

type Jenkins struct {
	// authentication
	username 		string
	token 			string

	// internal cache
	cache 			cache.Cache

	// pipeline name
	pipeline 		string

	// jobs API
	Jobs 			Jobs
}

// request
func (j *Jenkins) request(req *http.Request) (job *http.Response, err error) {
	client := &http.Client{}
	if job, err = client.Do(req); err != nil {
		log.Errorln(err)
	}
	return
}

// server
func (j *Jenkins) Server() {
	j.getAllJobs()

	j.cache.Init(*cacheProv)

	server := &generator.Server {
		Bindata:   generator.Binary {
			Asset:      Asset,
			AssetInfo:  AssetInfo,
			AssetNames: AssetNames,
			Gzip: 		true,
		},
		Extension: ".html",
		PublicDir: "./public",
		Reload:    true,
	}
	server.App()

	// main route
	server.Register(generator.Routes {
		Fn: func(ctx iris.Context) {
			pipelineName := ctx.Params().Get("pipeline")
			code := j.GetPipeline(pipelineName)
			ctx.ViewData("pipelines", j.Jobs.Stringify())
			ctx.ViewData("name", j.pipeline)
			ctx.ViewData("code", code)
			ctx.ViewData("dashboard",
				fmt.Sprintf("%s/job/%s", *jenkinsUrl, j.pipeline))

			_ = ctx.View("editor.html")
		},
		Method: "GET",
		Path:   "/pipeline/{pipeline:string}",
	})

	// ajax route
	server.Register(generator.Routes {
		Fn: func(ctx iris.Context) {
			if content := ctx.FormValue("content"); content != "" {
				if err := j.UpdatePipeline(ctx.Params().Get("pipeline"), content); err != nil {
					ctx.JSON(iris.Map{
						"status": false,
						"message": err.Error(),
					})
					return
				}
				ctx.JSON(iris.Map{
					"status": true,
					"message": "Pipeline updated",
				})
			}
		},
		Method: "POST",
		Path:   "/pipeline/{pipeline:string}",
	})

	server.Register(generator.Routes {
		Fn: func(ctx iris.Context) {
			if content := ctx.FormValue("content"); content != "" {
				if res, _ := j.ValidatePipeline(Normalize(content, false)); res != "" {
					if strings.TrimSpace(res) != "Jenkinsfile successfully validated." {
						ctx.JSON(iris.Map{
							"status":  true,
							"message": res,
						})
					} else {
						ctx.JSON(iris.Map{
							"status": false,
							"message": "No errors",
						})
					}
					return
				}
			}
		},
		Method: "POST",
		Path:   "/pipeline/checker",
	})
	server.HttpPort = *httpPort
	server.Serve()
}

// GetPipeline
func (j *Jenkins) GetPipeline(name string) (code string) {
	req, err := http.NewRequest("GET",
		fmt.Sprintf("%s/job/%s/config.xml/api/json", *jenkinsUrl, name), nil)
	if err != nil {
		log.Errorln(err)
	}
	req.SetBasicAuth(j.username, j.token)
	job, _ := j.request(req)
	defer job.Body.Close()

	rawDoc, err := ioutil.ReadAll(job.Body)
	if err != nil {
		log.Errorln(err)
		return
	}

	rd := bytes.ReplaceAll(rawDoc, []byte("version='1.1'"), []byte("version='1.0'"))
	rd = bytes.ReplaceAll(rd, []byte("version=\"1.1\""), []byte("version=\"1.0\""))
	b := bytes.NewReader(rd)
	doc, err := xmlquery.Parse(b)
	if err != nil {
		log.Errorln(err)
	}

	if baseElm := doc.SelectElement("flow-definition"); baseElm != nil {
		j.pipeline = name
		code = doc.SelectElement("flow-definition").
			SelectElement("definition").
			SelectElement("script").
			InnerText()

		if _, err := j.cache.Set(fmt.Sprintf("%s-xml", name), ConcatBytes(rawDoc, ""));
			err != nil {
			log.Fatalf("Cannot save XML: %s", err)
		}
	} else {
		log.Warnf("Job [%s] is not of type Pipeline", name)
	}

	return
}

// UpdatePipeline
func (j *Jenkins) UpdatePipeline(name, content string) (err error) {
	var xml string
	if xml, err = j.cache.Get(fmt.Sprintf("%s-xml", name)); xml != "" {
		xmlStr := ConcatBytes([]byte(xml), Normalize(content, true))

		// validate and check for errors on save
		if res, _ := j.ValidatePipeline(Normalize(content, false)); res != "" {
			if strings.TrimSpace(res) != "Jenkinsfile successfully validated." {
				return errors.New(res)
			}
		}

		req, err := http.NewRequest("POST",
			fmt.Sprintf("%s/job/%s/config.xml/api/json", *jenkinsUrl, name),
			strings.NewReader(xmlStr))
		if err != nil {
			log.Errorln(err)
			return err
		}
		req.SetBasicAuth(j.username, j.token)
		job, err := j.request(req)
		if err != nil {
			log.Errorln(err)
			return err
		}
		defer job.Body.Close()

		_, err = ioutil.ReadAll(job.Body)
		if err != nil {
			log.Warningln(job.Status, err)
			return err
		}
	}
	return err
}

// ValidatePipeline
func (j *Jenkins) ValidatePipeline(content string) (string, error) {
	b := new(bytes.Buffer)
	w := multipart.NewWriter(b)
	err := w.WriteField("jenkinsfile", content)
	if err != nil {
		log.Errorln(err)
		return "", err
	}
	_ = w.Close()

	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/pipeline-model-converter/validate", *jenkinsUrl), b)
	if err != nil {
		log.Errorln(err)
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	req.SetBasicAuth(j.username, j.token)
	job, err := j.request(req)
	if err != nil {
		log.Errorln(err)
		return "", err
	}
	defer job.Body.Close()

	result, err := ioutil.ReadAll(job.Body)
	if err != nil {
		log.Warningln(job.Status, err)
		return "", err
	}
	return string(result), nil
}

func main() {
	flag.StringVar(&jobsPrefix, "jobsPrefix", "", "Custom Jobs prefix to be displayed only")
	flag.Parse()

	if *username == "" || *password == "" {
		log.Fatal("Auth: Username/Password not provided!")
	}
	jenkins := &Jenkins {
		username: 	*username,
		token: 		*password,
	}
	jenkins.Server()
}