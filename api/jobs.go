package api

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"strings"
)

type (
	// Jobs list retrieved from API
	Jobs struct {
		All names `json:"jobs"`
	}
	names []struct {
		Name string `json:"name"`
	}
)

// Stringify stringify jobs names
func (jobs *Jobs) Stringify(jobsPrefix string) (allJobs string) {
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

// getAllJobs get all jobs from API
func (j *Jenkins) getAllJobs() {
	jobs, err := j.request("GET", "api/json?tree=jobs[name]", nil)
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(jobs, &j.Jobs); err != nil {
		log.Fatal(err)
	}
}
