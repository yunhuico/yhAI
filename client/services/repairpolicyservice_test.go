package services

import (
	"testing"

	"github.com/bmizerany/assert"
)

func TestGenerateEmailContent(t *testing.T) {
	const expect = `
		Dear user1,

		  We have detected unusual resource usage on your Linker DC/OS cluster.
		  Here is a short description of the alert.

		  Cluster name: cluster1
		  Host machine IP: 1.2.3.4
		  Alert name: High CPU usage
		  Have lasted for: 10m
		  Theresholds: 80.00% (current value: 83.45%)
		  Start at: Mon, 02 Jan 2006 15:04:05 MST

		  Please login to the host and check if anything goes wrong.

		  Best regards
		  Linker Networks
		   `
	gotContent, gotErr := generateEmailContent("user1", "cluster1", "1.2.3.4",
		"High CPU usage", "10m", "80.00", "83.45", "Mon, 02 Jan 2006 15:04:05 MST")
	assert.Equal(t, nil, gotErr)
	assert.Equal(t, gotContent, expect)
}

func TestCheckScaleNumber(t *testing.T) {
	var cases = []struct {
		inputScaleNumber       string
		inputAppNumber         string
		inputInstanceMaxNumber string
		inputInstanceMinNumber string
		expectedNumberStr      string
		expectedIspartial      bool
		expectedResult         bool
		expectingErr           bool
	}{
		// ut 1
		{"", "", "", "", "", false, false, true},
		// ut 2
		{"1", "3", "5", "1", "1", false, true, false},
		// ut 3
		{"6", "3", "5", "1", "5", true, true, false},
		// ut 4
		{"5", "3", "5", "1", "5", false, true, false},
		// ut 5
		{"3", "5", "5", "1", "3", false, true, false},
		// ut 6
		{"1", "3", "", "1", "1", false, true, false},
		// ut 7
		{"1", "3", "", "2", "2", true, true, false},
		// ut 8
		{"1", "3", "", "3", "3", false, false, false},
	}

	for _, c := range cases {
		gotNumberStr, gotIsPartial, gotResult, gotErr := checkScaleNumber(c.inputScaleNumber,
			c.inputAppNumber, c.inputInstanceMaxNumber, c.inputInstanceMinNumber)

		assert.Equal(t, gotNumberStr, c.expectedNumberStr)
		assert.Equal(t, gotIsPartial, c.expectedIspartial)
		assert.Equal(t, gotResult, c.expectedResult)
		if c.expectingErr {
			assert.NotEqual(t, gotErr, nil)
		}
	}
}
