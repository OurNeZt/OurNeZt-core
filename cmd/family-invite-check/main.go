// family-invite-check presents the real PostgreSQL regressions as readable
// reviewer results. It uses temporary schemas, not a running Core deployment.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

var checks = []struct{ test, label string }{
	{"TestFamilyInvitePostgresRegenerationRevokesAllPreviousCodes", "Replaced codes cannot add members; the latest code works and other families are unaffected."},
	{"TestFamilyInvitePostgresRegenerationRequiresManager", "Only owners and family admins can replace invite codes."},
	{"TestFamilyInvitePostgresFailedReplacementRollsBack", "Failed replacements leave the previous invite usable."},
	{"TestFamilyInvitePostgresExpiredCodesCannotJoin", "Expired codes cannot add members."},
	{"TestFamilyInvitePostgresConcurrentRegenerations", "Simultaneous regenerations leave only the latest code active."},
	{"TestFamilyInvitePostgresJoinWaitsForRegeneration", "A join waiting behind regeneration rejects the old code."},
	{"TestFamilyInvitePostgresRegenerationWaitsForJoin", "A join completed before regeneration keeps its membership."},
	{"TestFamilyInvitePostgresJoinRechecksExpiryAfterWaiting", "A code that expires while a join waits is rejected."},
}

const testPackage = "github.com/OurNeZt/ournezt-core/internal/repository/postgres"

func main() {
	timeout := flag.Duration("timeout", 2*time.Minute, "Maximum test duration (up to 10m)")
	flag.Parse()
	if err := run(context.Background(), *timeout, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, timeout time.Duration, out io.Writer) error {
	if timeout <= 0 || timeout > 10*time.Minute {
		return fmt.Errorf("-timeout must be positive and at most 10m")
	}
	if strings.TrimSpace(os.Getenv("TEST_DATABASE_URL")) == "" {
		return fmt.Errorf("set TEST_DATABASE_URL to a disposable PostgreSQL database; no checks were run")
	}
	module, err := os.ReadFile("go.mod")
	if err != nil || !strings.HasPrefix(string(module), "module github.com/OurNeZt/ournezt-core") {
		return fmt.Errorf("run this command from the OurNeZt-core repository root")
	}
	if _, err := fmt.Fprintln(out, "Checking family invite revocation in isolated PostgreSQL schemas..."); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout+15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "-json", "./internal/repository/postgres", "-run", "^TestFamilyInvitePostgres", "-count=1", "-timeout", timeout.String())
	cmd.Stderr = io.Discard // Do not echo database credentials or raw server errors.
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not start Go tests; ensure Go is installed: %w", err)
	}
	result := summarize(stdout, out)
	if result != nil {
		cancel()
	}
	waitErr := cmd.Wait()
	if result != nil {
		return result
	}
	if waitErr != nil {
		return fmt.Errorf("the Go test process failed; run go test ./internal/repository/postgres -run TestFamilyInvitePostgres -count=1 -v for details")
	}
	_, err = fmt.Fprintln(out, "PASS: all family invite checks completed. Temporary schemas were cleaned up; no existing families were changed.")
	return err
}

func summarize(input io.Reader, out io.Writer) error {
	labels := make(map[string]string, len(checks))
	for _, check := range checks {
		labels[check.test] = check.label
	}
	passed := make(map[string]bool, len(checks))
	packagePassed, incomplete := false, false
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), 2<<20)
	for scanner.Scan() {
		var event struct{ Action, Package, Test string }
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return fmt.Errorf("could not read Go test results")
		}
		if event.Package != testPackage {
			continue
		}
		if event.Action == "fail" || event.Action == "skip" {
			incomplete = true
		}
		if event.Test == "" {
			if event.Action == "pass" {
				packagePassed = true
			}
			continue
		}
		label, expected := labels[event.Test]
		if !expected {
			continue
		}
		var result string
		switch event.Action {
		case "pass":
			result, passed[event.Test] = "PASS", true
		case "fail":
			result = "FAIL"
		case "skip":
			result = "SKIP"
		default:
			continue
		}
		if _, err := fmt.Fprintf(out, "%s: %s\n", result, label); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("could not read Go test results")
	}
	if incomplete || !packagePassed || len(passed) != len(checks) {
		return fmt.Errorf("some checks failed, were skipped, or did not finish; run go test ./internal/repository/postgres -run TestFamilyInvitePostgres -count=1 -v for details")
	}
	return nil
}
