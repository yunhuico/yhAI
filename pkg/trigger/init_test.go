package trigger

import (
	"os"
	"testing"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
)

var dkronSuite = &DkronSuite{}

func TestMain(m *testing.M) {
	defer dkronSuite.Close()

	log.Init("go-testing.trigger", log.DebugLevel)
	os.Exit(m.Run())
}
