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
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"log/slog"

	kingpin "github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	mockedExitStatus = 0
	mockedStdout     string
	_, cancel        = context.WithTimeout(context.Background(), 5*time.Second)
	switchDevices    = []InfinibandDevice{
		{Type: "SW", LID: "2052", GUID: "0x506b4b03005c2740", Name: "iswr0l1",
			Uplinks: map[string]InfinibandUplink{
				"35": {Type: "CA", LID: "1432", PortNumber: "1", GUID: "0x506b4b0300cc02a6", Name: "p0001 HCA-1", Rate: (25 * 4 * 125000000), RawRate: 1.2890625e+10},
			},
		},
		{Type: "SW", LID: "1719", GUID: "0x7cfe9003009ce5b0", Name: "iswr1l1",
			Uplinks: map[string]InfinibandUplink{
				"1":  {Type: "SW", LID: "1516", PortNumber: "1", GUID: "0x7cfe900300b07320", Name: "ib-i1l2s01", Rate: (25 * 4 * 125000000), RawRate: 1.2890625e+10},
				"10": {Type: "CA", LID: "134", PortNumber: "1", GUID: "0x7cfe9003003b4bde", Name: "o0001 HCA-1", Rate: (25 * 4 * 125000000), RawRate: 1.2890625e+10},
				"11": {Type: "CA", LID: "133", PortNumber: "1", GUID: "0x7cfe9003003b4b96", Name: "o0002 HCA-1", Rate: (25 * 4 * 125000000), RawRate: 1.2890625e+10},
			},
		},
	}
)

func SetIbnetdiscoverExec(t *testing.T, setErr bool, timeout bool) {
	IbnetdiscoverExec = func(ctx context.Context) (string, error) {
		if setErr {
			return "", fmt.Errorf("Error")
		}
		if timeout {
			return "", context.DeadlineExceeded
		}
		out, err := ReadFixture("ibnetdiscover", "test")
		if err != nil {
			t.Fatal(err.Error())
			return "", err
		}
		return out, nil
	}
}

func SetPerfqueryExecs(t *testing.T, setErr bool, timeout bool) {
	PerfqueryExec = func(guid string, port string, extraArgs []string, ctx context.Context) (string, error) {
		if setErr {
			return "", fmt.Errorf("Error")
		}
		if timeout {
			return "", context.DeadlineExceeded
		}
		var out string
		var err error
		if len(extraArgs) == 2 {
			out, err = ReadFixture("perfquery", guid)
			if err != nil {
				t.Fatal(err.Error())
				return "", err
			}
		} else {
			out, err = ReadFixture("perfquery-rcv-error", fmt.Sprintf("%s-%s", guid, port))
			if err != nil {
				t.Fatal(err.Error())
				return "", err
			}
		}
		return out, nil
	}
}

func fakeExecCommand(ctx context.Context, command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestExecCommandHelper", "--", command}
	cs = append(cs, args...)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], cs...)
	es := strconv.Itoa(mockedExitStatus)
	tmp, _ := os.MkdirTemp("", "fake")
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1",
		"GOCOVERDIR=" + tmp,
		"STDOUT=" + mockedStdout,
		"EXIT_STATUS=" + es}
	return cmd
}

func TestExecCommandHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	//nolint:staticcheck
	fmt.Fprint(os.Stdout, os.Getenv("STDOUT"))
	i, _ := strconv.Atoi(os.Getenv("EXIT_STATUS"))
	os.Exit(i)
}

func setupGatherer(collector prometheus.Collector) prometheus.Gatherer {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)
	gatherers := prometheus.Gatherers{registry}
	return gatherers
}

// TestCollectorsCoexist regression test: every collector wired into a
// single registry must coexist. Specifically, the ibswinfo and switch
// collectors must not register descriptors under the same metric name
// with different label sets — client_golang rejects that at
// MustRegister time.
func TestCollectorsCoexist(t *testing.T) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(NewSwitchCollector(&switchDevices, false, slog.New(slog.DiscardHandler)))
	registry.MustRegister(NewIbswinfoCollector(&switchDevices, false, slog.New(slog.DiscardHandler)))
	registry.MustRegister(NewHCACollector(&switchDevices, false, slog.New(slog.DiscardHandler)))
}

// TestEndToEndPipeline drives the whole collection pipeline through the
// fixture-backed mocks: parse `ibnetdiscover` output, build the device
// list, and let SwitchCollector and HCACollector emit metrics by reading
// back-mocked `perfquery` output. It does not assert the exact metric
// values (those are covered by the per-collector tests); it asserts the
// pipeline produces a non-trivial number of samples without errors and
// with the expected metric families. This is the contract a reader of
// /metrics actually relies on.
func TestEndToEndPipeline(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--ibswinfo.static-cache-ttl=0", "--ibnetdiscover.cache-ttl=0"}); err != nil {
		t.Fatal(err)
	}
	resetIbnetdiscoverCache()
	SetIbnetdiscoverExec(t, false, false)
	SetPerfqueryExecs(t, false, false)

	disco := NewIBNetDiscover(false, slog.New(slog.DiscardHandler))
	switches, hcas, err := disco.GetPorts()
	if err != nil {
		t.Fatalf("GetPorts: %s", err)
	}
	if len(*switches) == 0 || len(*hcas) == 0 {
		t.Fatalf("topology empty: %d switches, %d hcas", len(*switches), len(*hcas))
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(disco)
	registry.MustRegister(NewSwitchCollector(switches, false, slog.New(slog.DiscardHandler)))
	registry.MustRegister(NewHCACollector(hcas, false, slog.New(slog.DiscardHandler)))

	mfs, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather: %s", err)
	}
	if len(mfs) == 0 {
		t.Fatal("registry produced no metric families")
	}

	// Spot-check that the headline metric families show up. If any of
	// these are missing the integration is broken even if individual
	// per-collector tests pass.
	want := []string{
		"infiniband_switch_info",
		"infiniband_switch_up",
		"infiniband_switch_port_transmit_data_bytes_total",
		"infiniband_hca_info",
		"infiniband_hca_up",
		"infiniband_hca_port_transmit_data_bytes_total",
	}
	got := make(map[string]bool, len(mfs))
	for _, mf := range mfs {
		got[mf.GetName()] = true
	}
	for _, name := range want {
		if !got[name] {
			t.Errorf("integration: missing metric family %q", name)
		}
	}
}
