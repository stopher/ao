package auroraconfig

import (
	"fmt"
	"github.com/skatteetaten/ao/pkg/cmdoptions"
	"github.com/skatteetaten/ao/pkg/configuration"
	"github.com/skatteetaten/ao/pkg/openshift"
	"os"
	"os/exec"
	"testing"
)

const REPO_PATH = "/tmp/ao/testRepo"

var testConfig *configuration.ConfigurationClass = &configuration.ConfigurationClass{
	OpenshiftConfig: &openshift.OpenshiftConfig{
		Localhost:   true,
		Affiliation: "aurora",
	},
	PersistentOptions: &cmdoptions.CommonCommandOptions{
		Localhost: true,
	},
}

func repoSetup(gitRemoteUrl string) {
	// Clear old test files
	os.RemoveAll(REPO_PATH)
	os.MkdirAll(REPO_PATH, 0755)
	os.Chdir(REPO_PATH)

	exec.Command("git", "init").Run()
	exec.Command("git", "remote", "add", "origin", gitRemoteUrl).Run()
}

func TestValidateRepo(t *testing.T) {
	gitRemoteUrl := fmt.Sprintf(GIT_URL_FORMAT, "user", testConfig.GetAffiliation())
	repoSetup(gitRemoteUrl)

	if err := ValidateRepo(gitRemoteUrl); err != nil {
		t.Error(err)
	}
}

func TestFindGitPath(t *testing.T) {
	gitRemoteUrl := fmt.Sprintf(GIT_URL_FORMAT, "user", testConfig.GetAffiliation())
	repoSetup(gitRemoteUrl)

	test := REPO_PATH + "/random/test"

	os.MkdirAll(test, 0755)
	os.Chdir(test)

	wd, _ := os.Getwd()
	path, found := FindGitPath(wd)
	if !found || path != REPO_PATH {
		t.Error("Expected git repo to be found")
	}
}
