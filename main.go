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
	"io"
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

	// pipeline type
	ctype			string

	// jobs API
	Jobs 			Jobs
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
			ctx.ViewData("type", j.ctype)
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
	rawDoc, _ := j.request(
		"GET", fmt.Sprintf("%s/job/%s/config.xml/api/json", *jenkinsUrl, name), nil)

	rd := bytes.ReplaceAll(rawDoc, []byte("version='1.1'"), []byte("version='1.0'"))
	rd = bytes.ReplaceAll(rd, []byte("version=\"1.1\""), []byte("version=\"1.0\""))
	b := bytes.NewReader(rd)
	doc, err := xmlquery.Parse(b)
	if err != nil {
		log.Errorln(err)
	}

	if baseElm := doc.SelectElement("flow-definition"); baseElm != nil {
		j.pipeline = name
		j.ctype = "groovy"
		code = baseElm.
			SelectElement("definition").
			SelectElement("script").
			InnerText()

		if _, err := j.cache.Set(fmt.Sprintf("%s-xml", name), ConcatBytes(rawDoc, ""));
			err != nil {
			log.Fatalf("Cannot save XML: %s", err)
		}
	} else {
		if baseShellElm := doc.SelectElement("project"); baseShellElm != nil {
			j.pipeline = name
			j.ctype = "sh"
			code = baseShellElm.
				SelectElement("builders").
				SelectElement("hudson.tasks.Shell").
				SelectElement("command").
				InnerText()
		} else {
			log.Warnf("Job [%s] is not of type Pipeline! Ignoring diplay...", name)
		}
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

		// update pipeline
		if _, err := j.request(
			"POST",
			fmt.Sprintf("%s/job/%s/config.xml/api/json", *jenkinsUrl, name),
			strings.NewReader(xmlStr)); err != nil {
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

	result, err := j.request(
		"POST", fmt.Sprintf("%s/pipeline-model-converter/validate", *jenkinsUrl), b, w)
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

// request
func (j *Jenkins) request(method, url string, body io.Reader, w ...*multipart.Writer) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	if len(w) > 0 {
		req.Header.Set("Content-Type", w[0].FormDataContentType())
	}

	req.SetBasicAuth(j.username, j.token)
	client := &http.Client{}
	job, err := client.Do(req)
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	defer job.Body.Close()

	response, err := ioutil.ReadAll(job.Body)
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	return response, nil
}