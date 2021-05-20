package main

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"strings"
)

type (
	Jobs struct {
		All Names `json:"jobs"`
	}
	Names []struct {
		Name string `json:"name"`
	}
)

// Stringify stringify jobs names
func (jobs *Jobs) Stringify() (allJobs string) {
	for i := range jobs.All {
		if jobsPrefix != "" {
			if strings.HasPrefix(jobs.All[i].Name, jobsPrefix) {
				allJobs += fmt.Sprintf("%s,", jobs.All[i].Name)
			}
		} else {
			allJobs += fmt.Sprintf("%s,", jobs.All[i].Name)
		}
	}
	return
}

// getAllJobs get all jobs
func (j *Jenkins) getAllJobs() {
	jobs, err := j.request(
		"GET", fmt.Sprintf("%s/api/json?tree=jobs[name]", *jenkinsUrl), nil)
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(jobs, &j.Jobs); err != nil {
		log.Fatal(err)
	}
}
