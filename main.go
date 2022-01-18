package main

import (
	"flag"
	"fmt"
	"github.com/MGSousa/go-generator"
	"github.com/MGSousa/go-jenkins-editor/api"
	"github.com/MGSousa/go-jenkins-editor/response"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
)

var (
	httpPort   = flag.Int("port", 5000, "UI Port")
	jenkinsUrl = flag.String("jenkinsUrl", "http://localhost:8080", "Jenkins Host Url")
	cacheProv  = flag.String("cacheProvider", "buntdb", "Cache provider (buntdb, redis)")

	username   string
	token      string
	jobsPrefix string

	production bool
)

// Server inbound listener with defined routes
// also initialize cache provider, fetches all jobs
func Server() {
	jenkins := &api.Jenkins{
		HostUrl:  *jenkinsUrl,
		Username: username,
		Token:    token,
	}
	jenkins.InitAPI(*cacheProv)

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
			code := jenkins.GetPipeline(ctx.Params().Get("pipeline"))
			ctx.ViewData("pipelines", jenkins.Jobs.Stringify(jobsPrefix))
			ctx.ViewData("name", jenkins.Pipeline.Name)
			ctx.ViewData("code", code)
			ctx.ViewData("type", jenkins.Pipeline.Type)
			ctx.ViewData("dashboard",
				fmt.Sprintf("%s/job/%s", *jenkinsUrl, jenkins.Pipeline.Name))

			_ = ctx.View("editor.html")
		},
		Method: "GET",
		Path:   "/pipeline/{pipeline:string}",
	})

	// ajax route
	server.Register(generator.Routes{
		Fn: func(ctx generator.Context) {
			if content := ctx.FormValue("content"); content != "" {
				if err := jenkins.UpdatePipeline(ctx.Params().Get("pipeline"), content); err != nil {
					if _, e := ctx.JSON(generator.Map{
						"status":  false,
						"message": err.Error(),
					}); e != nil {
						log.Fatal("update error JSON", e)
					}
					return
				}
				if _, e := ctx.JSON(generator.Map{
					"status":  true,
					"message": response.PipelineUpdate,
				}); e != nil {
					log.Fatal(e)
				}
			}
		},
		Method: "POST",
		Path:   "/pipeline/{pipeline:string}",
	})

	server.Register(generator.Routes{
		Fn: func(ctx generator.Context) {
			if content := ctx.FormValue("content"); content != "" {
				if res, _ := jenkins.ValidatePipeline(content); res != "" {
					if strings.TrimSpace(res) != response.IsValidPipeline {
						if _, e := ctx.JSON(generator.Map{
							"status":  false,
							"message": res,
						}); e != nil {
							log.Fatal("validation error JSON", e)
						}
					} else {
						if _, e := ctx.JSON(generator.Map{
							"status":  true,
							"message": "No errors",
						}); e != nil {
							log.Fatal(e)
						}
					}
					return
				}
			}
		},
		Method: "POST",
		Path:   "/pipeline/checker",
	})
	server.HttpPort = *httpPort
	server.Serve(production)
}

func main() {
	flag.StringVar(&jobsPrefix, "jobsPrefix", "", "Custom Jobs prefix to be displayed only")
	flag.Parse()

	// loads environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatal("error loading .env", err)
	}
	if os.Getenv("APP_ENV") == "production" {
		production = true
	}
	username = os.Getenv("JENKINS_ADMIN_USERNAME")
	token = os.Getenv("JENKINS_ADMIN_API_TOKEN")
	if username == "" || token == "" {
		log.Fatal("error: jenkins username/token not provided!")
	}

	// init server
	Server()
}
