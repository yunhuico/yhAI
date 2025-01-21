package webhook

import (
	"os"
	"testing"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
)

func TestMain(m *testing.M) {
	log.Init("go-testing.webhook", log.DebugLevel)
	os.Exit(m.Run())
}
