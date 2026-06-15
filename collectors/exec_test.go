// Copyright 2020 Trey Dockendorf
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collectors

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestRunWithProcessGroupKillsDescendants reproduces the orphan-child leak
// that motivated this helper: when ibswinfo.sh times out, exec.CommandContext
// SIGKILLs the shell but leaves grandchildren (sleep here, but flint / mst /
// mlxlink in production) running re-parented to init.
//
// The test spawns a shell that forks a long sleep, writes the sleep's PID to
// a temp file, and waits. We let the context expire, then read the PID and
// assert the descendant is gone. Without the helper setting up a PGID and
// the Cancel hook sending SIGKILL to the negative leader, this check finds
// the sleep still running.
//
// We identify the descendant by reading /proc/<pid>/comm rather than the raw
// kill(pid, 0) probe, because the kernel can reuse a PID within tens of
// milliseconds — a recycled PID would defeat the kill-zero check even though
// the actual sleep was killed correctly.
func TestRunWithProcessGroupKillsDescendants(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not found")
	}

	pidFile := filepath.Join(t.TempDir(), "child.pid")
	script := `sleep 30 & echo $! > "$0"; wait`

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", script, pidFile)
	if err := runWithProcessGroup(ctx, cmd); err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if ctx.Err() != context.DeadlineExceeded {
		t.Fatalf("expected DeadlineExceeded, got %v", ctx.Err())
	}

	// Give the kernel a beat to reap the SIGKILL'd group. 200ms is plenty
	// for SIGKILL to land and the descendant to transition to zombie state.
	time.Sleep(200 * time.Millisecond)

	raw, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("read pid file: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatalf("parse pid %q: %v", raw, err)
	}

	if descendantStillSleeping(pid) {
		// Clean up before failing so the leaked sleep doesn't survive the
		// whole test binary.
		_ = syscall.Kill(pid, syscall.SIGKILL)
		t.Fatalf("descendant pid %d still alive after timeout — process group not killed", pid)
	}
}

// descendantStillSleeping reports whether the given PID is a live `sleep`
// process. False if /proc/<pid> is gone (already reaped), if the slot was
// recycled by an unrelated command, or if the process is a zombie (killed
// but not yet reaped — common when the descendant's parent has died and the
// test binary inherited but doesn't wait4 it).
func descendantStillSleeping(pid int) bool {
	procDir := filepath.Join("/proc", strconv.Itoa(pid))

	comm, err := os.ReadFile(filepath.Join(procDir, "comm"))
	if err != nil {
		return false
	}
	if strings.TrimSpace(string(comm)) != "sleep" {
		return false
	}

	status, err := os.ReadFile(filepath.Join(procDir, "status"))
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(status), "\n") {
		if strings.HasPrefix(line, "State:") {
			// Z (zombie) and X (dead) mean SIGKILL already landed; only
			// R/S/D/T/t represent a process still consuming resources.
			return !strings.Contains(line, "Z (zombie)") && !strings.Contains(line, "X (dead)")
		}
	}
	return false
}

// TestNakedCommandContextLeaksDescendants is the negative control: it
// reproduces the bug runWithProcessGroup exists to fix, so the contract is
// not just asserted but observable. Using exec.CommandContext directly
// (the historical pattern) leaves grandchildren running after the context
// timeout. If this test ever starts passing, the Go runtime grew
// process-group cleanup of its own — at which point the helper is
// redundant and can be removed.
func TestNakedCommandContextLeaksDescendants(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not found")
	}

	pidFile := filepath.Join(t.TempDir(), "child.pid")
	script := `sleep 30 & echo $! > "$0"; wait`

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", script, pidFile)
	_ = cmd.Run()
	time.Sleep(50 * time.Millisecond)

	raw, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("read pid file: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatalf("parse pid %q: %v", raw, err)
	}

	if !descendantStillSleeping(pid) {
		// Bug gone or process recycled. Either way the helper is moot.
		t.Skipf("descendant pid %d already dead or recycled — Go runtime may have learned to clean process groups; consider dropping runWithProcessGroup", pid)
	}
	// Bug reproduced — descendant is alive. Clean up so it doesn't outlive
	// the test binary.
	_ = syscall.Kill(pid, syscall.SIGKILL)
}

// TestRunWithProcessGroupSetsPgid is a narrower assertion: the helper must
// always set Setpgid on cmd.SysProcAttr, even when one was already provided
// by the caller. Guards against accidental regressions that would drop the
// flag.
func TestRunWithProcessGroupSetsPgid(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "true")
	if err := runWithProcessGroup(ctx, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.Setpgid {
		t.Fatalf("Setpgid not applied: %+v", cmd.SysProcAttr)
	}

	// Preserve a caller-provided SysProcAttr.
	cmd2 := exec.CommandContext(ctx, "true")
	cmd2.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}
	if err := runWithProcessGroup(ctx, cmd2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cmd2.SysProcAttr.Setpgid {
		t.Fatal("Setpgid not applied when SysProcAttr was pre-populated")
	}
	if cmd2.SysProcAttr.Pdeathsig != syscall.SIGTERM {
		t.Fatal("pre-existing SysProcAttr field overwritten")
	}
}
