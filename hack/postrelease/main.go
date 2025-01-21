package main

import (
	"fmt"
	"log"
	"os"

	"github.com/xanzy/go-gitlab"
)

func main() {
	client, err := NewGitLabClient()
	if err != nil {
		log.Fatal(err)
	}
	projectID := os.Getenv("CI_PROJECT_ID")
	tag := os.Getenv("CI_COMMIT_TAG")
	//projectURL := os.Getenv("CI_PROJECT_URL")
	//log.Printf("Get Release Info: %s\n", tag)
	//fileMetaData, _, err := client.RepositoryFiles.GetFileMetaData(projectID, fmt.Sprintf("docs/release/%s.md", tag),
	//	&gitlab.GetFileMetaDataOptions{Ref: gitlab.String("main")})
	//if err != nil {
	//	log.Fatal(err)
	//}
	release, _, err := client.Releases.GetRelease(projectID, tag)
	if err != nil {
		log.Fatal(err)
	}
	//	releaseDocsMd := fmt.Sprintf(`
	//See [the release docs](%s/-/blob/%s/docs/release/%s.md) :rocket:
	//`, projectURL, fileMetaData.CommitID, tag)
	//	releaseDockerImageMd := fmt.Sprintf(`
	//## Docker images
	//* `+"`"+`docker pull registry.jihulab.com/jihulab/ultrafox/ultrafox:%s`+"`"+`
	//* `+"`"+`docker pull registry.jihulab.com/jihulab/ultrafox/ultrafox:%s-linux-amd64`+"`"+`
	//* `+"`"+`docker pull registry.jihulab.com/jihulab/ultrafox/ultrafox:%s-linux-arm64`+"`"+`
	//`, tag, tag, tag)
	releaseDockerImageMd := fmt.Sprintf(`
## Docker images
* `+"`"+`docker pull registry.jihulab.com/ultrafox/ultrafox:%s`+"`"+`
`, tag)
	description := fmt.Sprintf("%s\n%s", release.Description, releaseDockerImageMd)
	ops := gitlab.UpdateReleaseOptions{
		Description: &description,
		Milestones:  &[]string{tag[1:]},
	}
	log.Printf("Update Release: %s\n", tag)
	_, _, err = client.Releases.UpdateRelease(projectID, tag, &ops)
	if err != nil {
		log.Fatal(err)
	}
}

func NewGitLabClient() (*gitlab.Client, error) {
	host := os.Getenv("CI_API_V4_URL")
	token := os.Getenv("GITLAB_TOKEN")
	return gitlab.NewClient(token, gitlab.WithBaseURL(host))
}
