package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

var (
	// ErrStatusNotOK returned when mesos reponse code is not 200
	ErrStatusNotOK = errors.New("mesos response status code is not 200")
)

type RespBody struct {
	Slaves []Slave `json:"slaves"`
}

type Slave struct {
	ID       string `json:"id"`
	Hostname string `json:"hostname"`
	// ...
}

// Param endpoint is address(or domain name) and port of mesos node.
// Example 1:
// Endpoint: master.mesos/mesos
// GET http://master.mesos/mesos/slaves
// Example 2:
// Endpoint: 1.2.3.4:5050
// GET http://1.2.3.4:5050/slaves
func MesosHosts(endpoint string, timeoutMs int) (hostnames []string, err error) {
	apiURL := fmt.Sprintf("http://%s/slaves", endpoint)

	data, err := httpGetSlaves(apiURL, timeoutMs)
	if err != nil {
		return
	}

	respBody := RespBody{}
	err = json.Unmarshal(data, &respBody)

	for _, s := range respBody.Slaves {
		hostnames = append(hostnames, s.Hostname)
	}
	return
}

func httpGetSlaves(apiURL string, timeoutMs int) (body []byte, err error) {
	client := http.Client{
		Timeout: time.Duration(timeoutMs) * time.Millisecond,
	}
	resp, err := client.Get(apiURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = ErrStatusNotOK
		log.Printf("mesos error response: %+v\n", resp)
		return
	}

	body, err = ioutil.ReadAll(resp.Body)
	return
}
