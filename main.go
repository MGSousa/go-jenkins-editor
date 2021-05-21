package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"github.com/MGSousa/go-generator"
	"github.com/MGSousa/go-jenkins-editor/cache"
	"github.com/antchfx/xmlquery"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strings"
)

var (
	httpPort   = flag.Int("port", 5000, "UI Port")
	jenkinsUrl = flag.String("jenkinsUrl", "", "Jenkins Host Url")
	username   = flag.String("username", "", "Jenkins username")
	password   = flag.String("password", "", "Jenkins password")
	cacheProv  = flag.String("cacheProvider", "buntdb", "Cache provider (buntdb, redis)")
	jobsPrefix string
)

type (
	Pipeline struct {
		Name string
		Type string
	}

	Jenkins struct {
		username string // authentication
		token    string
		cache    cache.Cache // internal cache
		pipeline Pipeline    // Pipeline opts

		// Jobs list retrieved from API
		Jobs Jobs
	}
)

// request define handler for all requests done with Jenkins
func (j *Jenkins) request(method, url string, body io.Reader, w ...*multipart.Writer) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		log.Errorf("Error on init new request: %s", err)
		return nil, err
	}
	if len(w) > 0 {
		req.Header.Set("Content-Type", w[0].FormDataContentType())
	}

	req.SetBasicAuth(j.username, j.token)
	client := &http.Client{}
	job, err := client.Do(req)
	if err != nil {
		log.Errorf("Error on make request: %s", err)
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

// Server inbound listener with defined routes
func (j *Jenkins) Server() {
	j.getAllJobs()

	j.cache.Init(*cacheProv)

	server := &generator.Server{
		Bindata: generator.Binary{
			Asset:      Asset,
			AssetInfo:  AssetInfo,
			AssetNames: AssetNames,
			Gzip:       true,
		},
		Extension: ".html",
		PublicDir: "./public",
		Reload:    true,
	}
	server.App()

	// main route
	server.Register(generator.Routes{
		Fn: func(ctx generator.Context) {
			code := j.GetPipeline(ctx.Params().Get("pipeline"))
			ctx.ViewData("pipelines", j.Jobs.Stringify())
			ctx.ViewData("name", j.pipeline.Name)
			ctx.ViewData("code", code)
			ctx.ViewData("type", j.pipeline.Type)
			ctx.ViewData("dashboard",
				fmt.Sprintf("%s/job/%s", *jenkinsUrl, j.pipeline.Name))

			_ = ctx.View("editor.html")
		},
		Method: "GET",
		Path:   "/pipeline/{pipeline:string}",
	})

	// ajax route
	server.Register(generator.Routes{
		Fn: func(ctx generator.Context) {
			if content := ctx.FormValue("content"); content != "" {
				if err := j.UpdatePipeline(ctx.Params().Get("pipeline"), content); err != nil {
					ctx.JSON(generator.Map{
						"status":  false,
						"message": err.Error(),
					})
					return
				}
				ctx.JSON(generator.Map{
					"status":  true,
					"message": pipelineUpdate,
				})
			}
		},
		Method: "POST",
		Path:   "/pipeline/{pipeline:string}",
	})

	server.Register(generator.Routes{
		Fn: func(ctx generator.Context) {
			if content := ctx.FormValue("content"); content != "" {
				if res, _ := j.ValidatePipeline(Normalize(content, false)); res != "" {
					if strings.TrimSpace(res) != isValidPipeline {
						ctx.JSON(generator.Map{
							"status":  false,
							"message": res,
						})
					} else {
						ctx.JSON(generator.Map{
							"status":  true,
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

// GetPipeline retrieve pipeline data
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

	j.pipeline.Name = name
	if baseElm := doc.SelectElement("flow-definition"); baseElm != nil {
		j.pipeline.Type = "groovy"
		code = baseElm.
			SelectElement("definition").
			SelectElement("script").
			InnerText()

		if _, err := j.cache.Set(fmt.Sprintf("%s-xml", name), ConcatBytes(rawDoc, "")); err != nil {
			log.Fatalf("Cannot save XML: %s", err)
		}
	} else
		if elm := doc.SelectElement("project"); elm != nil {
			if shell := elm.SelectElement("builders").SelectElement("hudson.tasks.Shell");
			shell != nil {
				j.pipeline.Type = "sh"
				code = shell.InnerText()
			} else {
				j.pipeline.Type = ""
				log.Warnf("Job [%s] is not of type Shell/Groovy! Ignoring display...", name)
			}
	}

	return
}

// UpdatePipeline update pipeline data
func (j *Jenkins) UpdatePipeline(name, content string) (err error) {
	var xml string
	if xml, err = j.cache.Get(fmt.Sprintf("%s-xml", name)); xml != "" {
		xmlStr := ConcatBytes([]byte(xml), Normalize(content, true))

		// validate and check for errors on save
		if res, _ := j.ValidatePipeline(Normalize(content, false)); res != "" {
			if strings.TrimSpace(res) != isValidPipeline {
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

// ValidatePipeline validates pipeline data syntax
// passes multipart header with Jenkins file content to validator
func (j *Jenkins) ValidatePipeline(content string) (res string, err error) {
	var req []byte
	b := new(bytes.Buffer)
	w := multipart.NewWriter(b)

	if err = w.WriteField("jenkinsfile", content); err != nil {
		log.Errorf("Error on sending content: %s", err.Error())
		return
	}
	// finishes the multipart message when it closes
	_ = w.Close()

	req, err = j.request(
		"POST", fmt.Sprintf("%s/pipeline-model-converter/validate", *jenkinsUrl), b, w)
	if err != nil {
		log.Errorf("Error on validate: %s", err.Error())
		return
	}
	return string(req), nil
}

func main() {
	flag.StringVar(&jobsPrefix, "jobsPrefix", "", "Custom Jobs prefix to be displayed only")
	flag.Parse()

	if *username == "" || *password == "" {
		log.Fatal("Auth: Jenkins username/password not provided!")
	}
	jenkins := &Jenkins{
		username: *username,
		token:    *password,
	}
	jenkins.Server()
}
