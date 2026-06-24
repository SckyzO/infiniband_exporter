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
	"os/exec"
	"syscall"
)

// runWithProcessGroup runs cmd as the leader of a new process group, so
// that on context deadline we can kill the whole subtree with one syscall
// instead of just the direct PID. Without this, exec.CommandContext only
// SIGKILLs the immediate child; any grandchildren (e.g. flint, mst,
// mlxlink spawned by ibswinfo.sh) get re-parented to init and keep
// running against the fabric. Same pattern landed upstream as
// treydock/gpfs_exporter#79 for mmrepquota / mmlsfileset.
//
// We override cmd.Cancel rather than calling kill after cmd.Run() returns:
// once Wait has reaped the leader, kill(-pgid) is racy because the leader's
// PID is eligible for reuse and the cleanup contract of the pgid is fuzzy.
// Killing the group from inside Cancel runs while every process is still
// alive and the leader is still parked in our reaper.
func runWithProcessGroup(ctx context.Context, cmd *exec.Cmd) error {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	cmd.Cancel = func() error {
		// cmd.Process is set by Start before Cancel can fire. Negative
		// PID = kill the whole process group. Best-effort on error: the
		// next Wait4 surfaces the underlying state regardless.
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	return cmd.Run()
}
