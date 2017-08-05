package postrun

import (
	"log"
	"os"
	"testing"

	version "github.com/hashicorp/go-version"
	"github.com/hootsuite/atlantis/logging"
	. "github.com/hootsuite/atlantis/testing_util"
)

var logger = logging.NewSimpleLogger("", log.New(os.Stderr, "", log.LstdFlags), false, logging.Debug)
var postRun = &PostRun{}

func TestPostRunCreateScript_valid(t *testing.T) {
	cmds := []string{"echo", "date"}
	scriptName, err := createScript(cmds)
	Assert(t, scriptName != "", "there should be a script name")
	Assert(t, err == nil, "there should not be an error")
}

func TestPostRunExecuteScript_invalid(t *testing.T) {
	cmds := []string{"invalid", "command"}
	scriptName, _ := createScript(cmds)
	_, err := execute(scriptName)
	Assert(t, err != nil, "there should be an error")
}

func TestPostRunExecuteScript_valid(t *testing.T) {
	cmds := []string{"echo", "date"}
	scriptName, _ := createScript(cmds)
	output, err := execute(scriptName)
	Assert(t, err == nil, "there should not be an error")
	Assert(t, output != "", "there should be output")
}

func TestPostRun_valid(t *testing.T) {
	cmds := []string{"echo", "date"}
	version, _ := version.NewVersion("0.8.8")
	_, err := postRun.Execute(logger, cmds, "/tmp/atlantis", "staging", version)
	Ok(t, err)
}
