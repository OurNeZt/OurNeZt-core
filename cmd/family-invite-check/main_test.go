package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func results(t *testing.T, mode string) string {
	t.Helper()
	var data bytes.Buffer
	write := func(action, test string) {
		t.Helper()
		if err := json.NewEncoder(&data).Encode(map[string]string{"Action": action, "Package": testPackage, "Test": test}); err != nil {
			t.Fatal(err)
		}
	}
	for i, check := range checks {
		if mode == "missing test" && i == 0 {
			continue
		}
		action := "pass"
		if i == 0 {
			switch mode {
			case "failure":
				action = "fail"
			case "skipped":
				action = "skip"
			case "nested skip":
				write("skip", check.test+"/fixture")
			}
		}
		write(action, check.test)
	}
	if mode != "missing package" {
		write("pass", "")
	}
	return data.String()
}

func TestReviewerSummarizesRealRequiredCases(t *testing.T) {
	var out bytes.Buffer
	if err := summarize(strings.NewReader(results(t, "success")), &out); err != nil {
		t.Fatal(err)
	}
	if strings.Count(out.String(), "PASS:") != len(checks) || strings.Contains(out.String(), "TestFamilyInvite") {
		t.Fatalf("output is not a complete readable report: %s", &out)
	}
}

func TestReviewerRejectsIncompleteOrFailedChecks(t *testing.T) {
	for _, mode := range []string{"failure", "skipped", "nested skip", "missing test", "missing package", "malformed"} {
		t.Run(mode, func(t *testing.T) {
			input := results(t, mode)
			if mode == "malformed" {
				input = "not test JSON"
			}
			if err := summarize(strings.NewReader(input), &bytes.Buffer{}); err == nil {
				t.Fatalf("%s falsely passed", mode)
			}
		})
	}
}

type brokenProgressWriter struct{}

func (brokenProgressWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestReviewerStopsOnOutputFailure(t *testing.T) {
	if err := summarize(strings.NewReader(results(t, "success")), brokenProgressWriter{}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("output failure lost: %v", err)
	}
}

func TestReviewerRequiresDatabaseAndValidTimeout(t *testing.T) {
	t.Setenv("TEST_DATABASE_URL", "")
	if err := run(context.Background(), time.Minute, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "no checks were run") {
		t.Fatalf("missing database was not reported: %v", err)
	}
	for _, timeout := range []time.Duration{0, -time.Second, 11 * time.Minute} {
		if err := run(context.Background(), timeout, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "timeout") {
			t.Fatalf("invalid timeout accepted: %v", err)
		}
	}
}
