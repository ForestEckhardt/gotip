package gotip_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitGotip(t *testing.T) {
	suite := spec.New("gotip", spec.Report(report.Terminal{}), spec.Parallel())
	suite("Detect", testDetect)
	suite.Run(t)
}
