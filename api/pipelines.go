package api

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/MGSousa/go-jenkins-editor/response"
	"github.com/antchfx/xmlquery"
	log "github.com/sirupsen/logrus"
	"mime/multipart"
	"strings"
)

type Pipeline struct {
	Name string
	Type string
}

// GetPipeline retrieve pipeline data
func (j *Jenkins) GetPipeline(name string) (code string) {
	rawDoc, _ := j.request(
		"GET", fmt.Sprintf("job/%s/config.xml/api/json", name), nil)

	rd := bytes.ReplaceAll(rawDoc, []byte("version='1.1'"), []byte("version='1.0'"))
	rd = bytes.ReplaceAll(rd, []byte("version=\"1.1\""), []byte("version=\"1.0\""))
	b := bytes.NewReader(rd)
	doc, err := xmlquery.Parse(b)
	if err != nil {
		log.Errorln(err)
	}

	// parse pipeline type
	j.Pipeline.Name = name
	if baseElm := doc.SelectElement("flow-definition"); baseElm != nil {
		j.Pipeline.Type = "groovy"
		code = baseElm.
			SelectElement("definition").
			SelectElement("script").
			InnerText()

		if _, err := j.cache.Set(fmt.Sprintf("%s-xml", name), concatBytes(rawDoc, "")); err != nil {
			log.Fatalf("Cannot save XML: %s", err)
		}
	} else if elm := doc.SelectElement("project"); elm != nil {
		if shell := elm.SelectElement("builders").SelectElement("hudson.tasks.Shell"); shell != nil {
			j.Pipeline.Type = "sh"
			code = shell.InnerText()
		} else {
			j.Pipeline.Type = ""
			log.Warnf("Job [%s] is not of type Shell/Groovy! Ignoring display...", name)
		}
	}

	return
}

// UpdatePipeline update pipeline data
func (j *Jenkins) UpdatePipeline(name, content string) (err error) {
	var xml string
	if xml, err = j.cache.Get(fmt.Sprintf("%s-xml", name)); xml != "" {
		xmlStr := concatBytes([]byte(xml), normalize(content, true))

		// validate and check for errors on save
		if res, _ := j.ValidatePipeline(content); res != "" {
			if strings.TrimSpace(res) != response.IsValidPipeline {
				return errors.New(res)
			}
		}

		// update pipeline
		if _, err := j.request(
			"POST",
			fmt.Sprintf("job/%s/config.xml/api/json", name),
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

	if err = w.WriteField("jenkinsfile", normalize(content, false)); err != nil {
		log.Errorf("Error on sending content: %s", err.Error())
		return
	}
	// finishes the multipart message when it closes
	_ = w.Close()

	req, err = j.request(
		"POST", "pipeline-model-converter/validate", b, w)
	if err != nil {
		log.Errorf("Error on validate: %s", err.Error())
		return
	}
	return string(req), nil
}
