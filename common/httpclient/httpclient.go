package httpclient

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

type Header struct {
	Key   string
	Value string
}

func Http_get(url string, body string, headers ...Header) (resp *http.Response, err error) {
	client, _ := getClient(false, "")
	var Body io.Reader

	if body == "" {
		Body = nil
	} else {
		Body = ioutil.NopCloser(strings.NewReader(body))
	}
	req, err := http.NewRequest("GET", "http://"+url, Body)
	if err != nil {
		return
	}
	for _, header := range headers {
		req.Header.Set(header.Key, header.Value)
	}
	resp, err = client.Do(req)
	return
}

func Http_post(url string, body string, headers ...Header) (resp *http.Response, err error) {
	client, _ := getClient(false, "")
	var Body io.Reader
	if body == "" {
		Body = nil
	} else {
		Body = ioutil.NopCloser(strings.NewReader(body))
	}
	req, _ := http.NewRequest("POST", "http://"+url, Body)
	for _, header := range headers {
		req.Header.Set(header.Key, header.Value)
	}

	// req.Header.Set("Content-Type", contenttype)
	resp, err = client.Do(req)
	return
}

func Http_put(url string, body string, headers ...Header) (resp *http.Response, err error) {
	client, _ := getClient(false, "")
	var Body io.Reader
	if body == "" {
		Body = nil
	} else {
		Body = ioutil.NopCloser(strings.NewReader(body))
	}
	req, _ := http.NewRequest("PUT", "http://"+url, Body)
	// req.Header.Set("Content-Type", contenttype)
	for _, header := range headers {
		req.Header.Set(header.Key, header.Value)
	}
	resp, err = client.Do(req)
	return
}

func Http_delete(url string, body string, headers ...Header) (resp *http.Response, err error) {
	client, _ := getClient(false, "")
	var Body io.Reader
	if body == "" {
		Body = nil
	} else {
		Body = ioutil.NopCloser(strings.NewReader(body))
	}
	req, _ := http.NewRequest("DELETE", "http://"+url, Body)
	// req.Header.Set("Content-Type", contenttype)
	for _, header := range headers {
		req.Header.Set(header.Key, header.Value)
	}
	resp, err = client.Do(req)
	return
}

func Https_get(url string, body string, caCertPath string, headers ...Header) (resp *http.Response, err error) {
	client, err := getClient(true, caCertPath)
	if err != nil {
		return
	}
	var Body io.Reader

	if body == "" {
		Body = nil
	} else {
		Body = ioutil.NopCloser(strings.NewReader(body))
	}
	req, err := http.NewRequest("GET", "https://"+url, Body)
	if err != nil {
		return
	}
	for _, header := range headers {
		req.Header.Set(header.Key, header.Value)
	}
	resp, err = client.Do(req)
	return
}

func Https_post(url string, body string, caCertPath string, headers ...Header) (resp *http.Response, err error) {
	client, err := getClient(true, caCertPath)
	if err != nil {
		return
	}
	var Body io.Reader
	if body == "" {
		Body = nil
	} else {
		Body = ioutil.NopCloser(strings.NewReader(body))
	}
	req, _ := http.NewRequest("POST", "https://"+url, Body)
	for _, header := range headers {
		req.Header.Set(header.Key, header.Value)
	}

	// req.Header.Set("Content-Type", contenttype)
	resp, err = client.Do(req)
	return
}

func Https_put(url string, body string, caCertPath string, headers ...Header) (resp *http.Response, err error) {
	client, err := getClient(true, caCertPath)
	if err != nil {
		return
	}
	var Body io.Reader
	if body == "" {
		Body = nil
	} else {
		Body = ioutil.NopCloser(strings.NewReader(body))
	}
	req, _ := http.NewRequest("PUT", "https://"+url, Body)
	for _, header := range headers {
		req.Header.Set(header.Key, header.Value)
	}
	resp, err = client.Do(req)
	return
}

func Https_delete(url string, body string, caCertPath string, headers ...Header) (resp *http.Response, err error) {
	client, err := getClient(true, caCertPath)
	if err != nil {
		return
	}
	var Body io.Reader
	if body == "" {
		Body = nil
	} else {
		Body = ioutil.NopCloser(strings.NewReader(body))
	}
	req, _ := http.NewRequest("DELETE", "https://"+url, Body)
	for _, header := range headers {
		req.Header.Set(header.Key, header.Value)
	}
	resp, err = client.Do(req)
	return
}

func getClient(isHttps bool, caCertPath string) (client *http.Client, err error) {
	if !isHttps {
		client = &http.Client{}
		return
	}

	pool := x509.NewCertPool()
	// for _, caCertPath := range caCertPaths {
	// 	caCrt, err := ioutil.ReadFile(caCertPath)
	// 	if err != nil {
	// 		return
	// 	}
	// 	pool.AppendCertsFromPEM(caCrt)
	// }
	caCrt, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		return
	}
	pool.AppendCertsFromPEM(caCrt)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: pool},
	}

	client = &http.Client{Transport: tr}
	return
}
