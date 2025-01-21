package services

import (
	"encoding/json"
	"testing"

	"linkernetworks.com/dcos-backend/common/persistence/entity"

	marathon "github.com/LinkerNetworks/go-marathon"
	"github.com/bmizerany/assert"
)

func TestStringInSlice(t *testing.T) {
	var cases = []struct {
		s      string
		slice  []string
		expect bool
	}{
		{"", []string{}, false},
		{"a", []string{}, false},
		{"a", []string{"a"}, true},
		{"a", []string{"a", "b"}, true},
		{"a", []string{"b", "c"}, false},
		{"tom", []string{"tom", "cat"}, true},
	}

	for _, c := range cases {
		//call
		got := stringInSlice(c.s, c.slice)
		//assert
		assert.Equal(t, c.expect, got)
	}
}

func TestIsLocalRegistryImage(t *testing.T) {
	c1_image := ""
	c1_list := []string{}
	c1_expect := false

	c2_image := "java"
	c2_list := []string{"https://index.docker.io/v1/"}
	c2_expect := false

	c3_image := "java"
	c3_list := []string{"https://index.docker.io/v1/", "ec2-52-78-64-61.ap-northeast-2.compute.amazonaws.com:5000"}
	c3_expect := false

	c4_image := "docker.io/java"
	c4_list := []string{"https://index.docker.io/v1/", "ec2-52-78-64-61.ap-northeast-2.compute.amazonaws.com:5000"}
	c4_expect := false

	c5_image := "docker.io/java:1.7.0"
	c5_list := []string{"https://index.docker.io/v1/", "ec2-52-78-64-61.ap-northeast-2.compute.amazonaws.com:5000"}
	c5_expect := false

	c6_image := "ec2-52-78-64-61.ap-northeast-2.compute.amazonaws.com:5000/java"
	c6_list := []string{"https://index.docker.io/v1/", "ec2-52-78-64-61.ap-northeast-2.compute.amazonaws.com:5000"}
	c6_expect := true

	c7_image := "ec2-52-78-64-61.ap-northeast-2.compute.amazonaws.com:5000/java:latest"
	c7_list := []string{"https://index.docker.io/v1/", "ec2-52-78-64-61.ap-northeast-2.compute.amazonaws.com:5000"}
	c7_expect := true

	c8_image := "ec2-52-78-64-61.ap-northeast-2.compute.amazonaws.com:5000/java:1.7.0"
	c8_list := []string{"https://index.docker.io/v1/", "ec2-52-78-64-61.ap-northeast-2.compute.amazonaws.com:5000"}
	c8_expect := true

	c9_image := "ec2-52-78-64-61.ap-northeast-2.compute.amazonaws.com:5000/java:1.7.0"
	c9_list := []string{"https://index.docker.io/v1/", "ec2-52-78-64-61.ap-northeast-2.compute.amazonaws.com:5000"}
	c9_expect := true

	c10_image := "ec2-52-78-64-61.ap-northeast-2.compute.amazonaws.com:5000/java:1.7.0"
	c10_list := []string{"https://index.docker.io/v1/", "ec2-52-78-64-61.ap-northeast-2.compute.amazonaws.com:5000"}
	c10_expect := true

	c11_image := "git.linkeriot.io:5000/java:1.7.0"
	c11_list := []string{"https://index.docker.io/v1/", "ec2-52-78-64-61.ap-northeast-2.compute.amazonaws.com:5000", "git.linkeriot.io:5000"}
	c11_expect := true

	c12_image := "daocloud.io/library/nginx"
	c12_list := []string{"https://index.docker.io/v1/", "ec2-52-78-64-61.ap-northeast-2.compute.amazonaws.com:5000", "git.linkeriot.io:5000"}
	c12_expect := false

	c13_image := "git.linkeriot.io:5000/someone/java:1.7.0"
	c13_list := []string{"https://index.docker.io/v1/", "ec2-52-78-64-61.ap-northeast-2.compute.amazonaws.com:5000", "git.linkeriot.io:5000"}
	c13_expect := true

	// not allow protocol in regitry url in our registry except https://index.docker.io/v1/
	c14_image := "git.linkeriot.io:5000/someone/java:1.7.0"
	c14_list := []string{"https://index.docker.io/v1/", "ec2-52-78-64-61.ap-northeast-2.compute.amazonaws.com:5000", "http://git.linkeriot.io:5000"}
	c14_expect := false

	var cases = []struct {
		image  string
		list   []string
		expect bool
	}{
		{c1_image, c1_list, c1_expect},
		{c2_image, c2_list, c2_expect},
		{c3_image, c3_list, c3_expect},
		{c4_image, c4_list, c4_expect},
		{c5_image, c5_list, c5_expect},
		{c6_image, c6_list, c6_expect},
		{c7_image, c7_list, c7_expect},
		{c8_image, c8_list, c8_expect},
		{c9_image, c9_list, c9_expect},
		{c10_image, c10_list, c10_expect},
		{c11_image, c11_list, c11_expect},
		{c12_image, c12_list, c12_expect},
		{c13_image, c13_list, c13_expect},
		{c14_image, c14_list, c14_expect},
	}

	for _, c := range cases {
		//call
		got := isLocalRegistryImage(c.image, c.list)
		//assert
		assert.Equal(t, c.expect, got)
	}
}

func TestCheckAndInjectUriApp(t *testing.T) {
	var config = []byte(`
	{
		"auths": {
			"https://index.docker.io/v1/": {
				"auth": "enlmZGVkaDo0MjczODEy"
			},
			"ec2-54-199-254-212.ap-northeast-1.compute.amazonaws.com:5000": {
				"auth": "enhxOnBhc3N3b3Jk"
			},
			"ec2-52-78-64-61.ap-northeast-2.compute.amazonaws.com:5000":{
				"auth": "enlmZGVkaDo0Mjcrer2"
			}
		}
	}`)

	var dockerAuthConfig = entity.DockerAuthConfig{}
	err := json.Unmarshal(config, &dockerAuthConfig)
	if err != nil {
		t.Errorf("unmarshal json error: %v", err)
	}

	var registryList entity.RegistryList
	for key := range dockerAuthConfig.Auths {
		registryList.RegistryList = append(registryList.RegistryList, key)
	}

	app := marathon.NewDockerApplication()
	app.ID = "a"
	app.Container.Docker.Image = "ec2-54-199-254-212.ap-northeast-1.compute.amazonaws.com:5000/java:1.8"

	checkAndInjectUri(app, registryList)
	if app.Uris == nil {
		t.Fail()
		return
	}
	if !stringInSlice(URI_DOCKER_TAR_GZ, *app.Uris) {
		t.Fail()
	}
}
