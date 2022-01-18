package api

import (
	"errors"
	"fmt"
	"github.com/MGSousa/go-jenkins-editor/cache"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
)

type Jenkins struct {
	HostUrl  string
	Username string
	Token    string

	// Pipeline opts
	Pipeline Pipeline

	// Jobs list retrieved from API
	Jobs Jobs

	// internal cache
	cache cache.Cache
}

// InitAPI initiate cache
func (j *Jenkins) InitAPI(cacheProvider string) {
	j.cache.Init(cacheProvider)
	j.getAllJobs()
}

// request define handler for all requests done with Jenkins
// also process some 4XX responses
func (j *Jenkins) request(method, url string, body io.Reader, w ...*multipart.Writer) ([]byte, error) {
	req, err := http.NewRequest(method, fmt.Sprintf("%s/%s", j.HostUrl, url), body)
	if err != nil {
		log.Errorf("error on initiate request: %s", err)
		return nil, err
	}
	if len(w) > 0 {
		req.Header.Set("Content-Type", w[0].FormDataContentType())
	}

	req.SetBasicAuth(j.Username, j.Token)
	client := &http.Client{}
	job, err := client.Do(req)
	if err != nil {
		log.Errorf("Error on make request: %s", err)
		return nil, err
	}
	defer job.Body.Close()

	switch job.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, errors.New(job.Status)
	}

	response, err := ioutil.ReadAll(job.Body)
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	return response, nil
}
