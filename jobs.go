package main

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strings"
)

type (
	Jobs	struct {
		All 	Names `json:"jobs"`
	}
	Names []struct {
		Name  	string `json:"name"`
	}
)

// Stringify
func (jobs *Jobs) Stringify() (allJobs string) {
	for i := range jobs.All {
		if strings.HasPrefix(jobs.All[i].Name, "kk-") {
			allJobs += fmt.Sprintf("%s,", jobs.All[i].Name)
		}
	}
	return
}

// getAllJobs
func (j *Jenkins) getAllJobs() {
	req, err := http.NewRequest("GET",
		fmt.Sprintf("%s/api/json?tree=jobs[name]", *jenkinsUrl), nil)
	if err != nil {
		log.Errorln(err)
	}
	req.SetBasicAuth(j.username, j.token)
	res, _ := j.request(req)
	defer res.Body.Close()

	jobs, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Errorln(err)
		return
	}
	if err := json.Unmarshal(jobs, &j.Jobs); err != nil {
		log.Errorln(err)
	}
}
