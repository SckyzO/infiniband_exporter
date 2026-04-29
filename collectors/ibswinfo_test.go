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
	"math"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"log/slog"

	kingpin "github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestParseIBSWInfo(t *testing.T) {
	out, err := ReadFixture("ibswinfo", "test1")
	if err != nil {
		t.Fatal("Unable to read fixture")
	}
	data := Ibswinfo{}
	err = parse_ibswinfo(out, &data, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if data.PartNumber != "MSB7790-ES2F" {
		t.Errorf("Unexpected part number, got %s", data.PartNumber)
	}
	if data.SerialNumber != "MT1943X00498" {
		t.Errorf("Unexpected serial number, got %s", data.SerialNumber)
	}
	if data.PSID != "MT_1880110032" {
		t.Errorf("Unexpected PSID, got %s", data.PSID)
	}
	if data.FirmwareVersion != "11.2008.2102" {
		t.Errorf("Unexpected firmware version, got %s", data.FirmwareVersion)
	}
	if data.Uptime != 13862333 {
		t.Errorf("Unexpected uptime, got %f", data.Uptime)
	}
	if len(data.PowerSupplies) != 2 {
		t.Errorf("Unexpected number of power supplies, got %d", len(data.PowerSupplies))
	}
	var psu0 SwitchPowerSupply
	for _, psu := range data.PowerSupplies {
		if psu.ID == "0" {
			psu0 = psu
			break
		}
	}
	if psu0.Status != "OK" {
		t.Errorf("Unexpected power supply status, got %s", psu0.Status)
	}
	if psu0.DCPower != "OK" {
		t.Errorf("Unexpected power supply dc power status, got %s", psu0.DCPower)
	}
	if psu0.FanStatus != "OK" {
		t.Errorf("Unexpected power supply fan status, got %s", psu0.FanStatus)
	}
	if psu0.PowerW != 72 {
		t.Errorf("Unexpected power supply watts, got %f", psu0.PowerW)
	}
	if data.Temp != 45 {
		t.Errorf("Unexpected temp, got %f", data.Temp)
	}
	if data.FanStatus != "ERROR" {
		t.Errorf("Unexpected fan status, got %s", data.FanStatus)
	}
	if len(data.Fans) != 8 {
		t.Errorf("Unexpected number of fans, got %d", len(data.Fans))
	}
	var fan1 SwitchFan
	for _, fan := range data.Fans {
		if fan.ID == "1" {
			fan1 = fan
			break
		}
	}
	if fan1.RPM != 8493 {
		t.Errorf("Unexpected fan RPM, got %f", fan1.RPM)
	}
}

func TestParseIBSWInfoFailedPSU(t *testing.T) {
	out, err := ReadFixture("ibswinfo", "test3")
	if err != nil {
		t.Fatal("Unable to read fixture")
	}
	data := Ibswinfo{}
	err = parse_ibswinfo(out, &data, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if data.PartNumber != "MQM8790-HS2F" {
		t.Errorf("Unexpected part number, got %s", data.PartNumber)
	}
	if data.SerialNumber != "MT2148T25782" {
		t.Errorf("Unexpected serial number, got %s", data.SerialNumber)
	}
	if data.PSID != "MT_0000000063" {
		t.Errorf("Unexpected PSID, got %s", data.PSID)
	}
	if data.FirmwareVersion != "27.2010.4102" {
		t.Errorf("Unexpected firmware version, got %s", data.FirmwareVersion)
	}
	if len(data.PowerSupplies) != 2 {
		t.Errorf("Unexpected number of power supplies, got %d", len(data.PowerSupplies))
	}
	var psu0, psu1 SwitchPowerSupply
	for _, psu := range data.PowerSupplies {
		if psu.ID == "0" {
			psu0 = psu
			break
		}
	}
	if psu0.Status != "OK" {
		t.Errorf("Unexpected power supply status, got %s", psu0.Status)
	}
	if psu0.DCPower != "OK" {
		t.Errorf("Unexpected power supply dc power status, got %s", psu0.DCPower)
	}
	if psu0.FanStatus != "OK" {
		t.Errorf("Unexpected power supply fan status, got %s", psu0.FanStatus)
	}
	if psu0.PowerW != 287 {
		t.Errorf("Unexpected power supply watts, got %f", psu0.PowerW)
	}
	for _, psu := range data.PowerSupplies {
		if psu.ID == "1" {
			psu1 = psu
			break
		}
	}
	if psu1.Status != "OK" {
		t.Errorf("Unexpected power supply status, got %s", psu1.Status)
	}
	if psu1.DCPower != "ERROR" {
		t.Errorf("Unexpected power supply dc power status, got %s", psu1.DCPower)
	}
	if psu1.FanStatus != "ERROR" {
		t.Errorf("Unexpected power supply fan status, got %s", psu1.FanStatus)
	}
	if !math.IsNaN(psu1.PowerW) {
		t.Errorf("Unexpected power supply watts, got %f", psu1.PowerW)
	}
	if data.Temp != 47 {
		t.Errorf("Unexpected temp, got %f", data.Temp)
	}
	if data.FanStatus != "OK" {
		t.Errorf("Unexpected fan status, got %s", data.FanStatus)
	}
	if len(data.Fans) != 9 {
		t.Errorf("Unexpected number of fans, got %d", len(data.Fans))
	}
	var fan1 SwitchFan
	for _, fan := range data.Fans {
		if fan.ID == "1" {
			fan1 = fan
			break
		}
	}
	if fan1.RPM != 5959 {
		t.Errorf("Unexpected fan RPM, got %f", fan1.RPM)
	}
}

func TestParseIBSWInfoErrors(t *testing.T) {
	tests := []string{
		"test-err1",
		"test-err2",
		"test-err3",
	}
	for i, test := range tests {
		out, err := ReadFixture("ibswinfo", test)
		if err != nil {
			t.Fatalf("Unable to read fixture %s", test)
		}
		data := Ibswinfo{}
		err = parse_ibswinfo(out, &data, slog.New(slog.DiscardHandler))
		if err == nil {
			t.Errorf("Expected an error for test %s(%d)", test, i)
		}
	}
}

func TestIbswinfoCollector(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--ibswinfo.static-cache-ttl=0"}); err != nil {
		t.Fatal(err)
	}
	IbswinfoExec = func(lid string, vitals bool, ctx context.Context) (string, error) {
		if lid == "1719" {
			out, err := ReadFixture("ibswinfo", "test1")
			return out, err
		} else if lid == "2052" {
			out, err := ReadFixture("ibswinfo", "test2")
			return out, err
		} else {
			return "", nil
		}
	}
	expected := `
		# HELP infiniband_exporter_collect_errors Number of errors that occurred during collection
		# TYPE infiniband_exporter_collect_errors gauge
		infiniband_exporter_collect_errors{collector="ibswinfo"} 0
		# HELP infiniband_exporter_collect_timeouts Number of timeouts that occurred during collection
		# TYPE infiniband_exporter_collect_timeouts gauge
		infiniband_exporter_collect_timeouts{collector="ibswinfo"} 0
		# HELP infiniband_switch_fan_rpm Switch fan rotation speed in RPM (one series per fan).
		# TYPE infiniband_switch_fan_rpm gauge
		infiniband_switch_fan_rpm{fan="1",guid="0x506b4b03005c2740",switch="iswr0l1"} 6125
		infiniband_switch_fan_rpm{fan="1",guid="0x7cfe9003009ce5b0",switch="iswr1l1"} 8493
		infiniband_switch_fan_rpm{fan="2",guid="0x506b4b03005c2740",switch="iswr0l1"} 5251
		infiniband_switch_fan_rpm{fan="2",guid="0x7cfe9003009ce5b0",switch="iswr1l1"} 7349
		infiniband_switch_fan_rpm{fan="3",guid="0x506b4b03005c2740",switch="iswr0l1"} 6013
		infiniband_switch_fan_rpm{fan="3",guid="0x7cfe9003009ce5b0",switch="iswr1l1"} 8441
		infiniband_switch_fan_rpm{fan="4",guid="0x506b4b03005c2740",switch="iswr0l1"} 5335
		infiniband_switch_fan_rpm{fan="4",guid="0x7cfe9003009ce5b0",switch="iswr1l1"} 7270
		infiniband_switch_fan_rpm{fan="5",guid="0x506b4b03005c2740",switch="iswr0l1"} 6068
		infiniband_switch_fan_rpm{fan="5",guid="0x7cfe9003009ce5b0",switch="iswr1l1"} 8337
		infiniband_switch_fan_rpm{fan="6",guid="0x506b4b03005c2740",switch="iswr0l1"} 5423
		infiniband_switch_fan_rpm{fan="6",guid="0x7cfe9003009ce5b0",switch="iswr1l1"} 7156
		infiniband_switch_fan_rpm{fan="7",guid="0x506b4b03005c2740",switch="iswr0l1"} 5854
		infiniband_switch_fan_rpm{fan="7",guid="0x7cfe9003009ce5b0",switch="iswr1l1"} 8441
		infiniband_switch_fan_rpm{fan="8",guid="0x506b4b03005c2740",switch="iswr0l1"} 5467
		infiniband_switch_fan_rpm{fan="8",guid="0x7cfe9003009ce5b0",switch="iswr1l1"} 7232
		infiniband_switch_fan_rpm{fan="9",guid="0x506b4b03005c2740",switch="iswr0l1"} 5906
		# HELP infiniband_switch_fan_status_info Constant 1 with the current overall fan status string label.
		# TYPE infiniband_switch_fan_status_info gauge
		infiniband_switch_fan_status_info{guid="0x506b4b03005c2740",status="OK",switch="iswr0l1"} 1
		infiniband_switch_fan_status_info{guid="0x7cfe9003009ce5b0",status="ERROR",switch="iswr1l1"} 1
		# HELP infiniband_switch_hardware_info Constant 1 carrying switch hardware identification labels (firmware version, PSID, part/serial numbers).
		# TYPE infiniband_switch_hardware_info gauge
		infiniband_switch_hardware_info{firmware_version="11.2008.2102",guid="0x7cfe9003009ce5b0",part_number="MSB7790-ES2F",psid="MT_1880110032",serial_number="MT1943X00498",switch="iswr1l1"} 1
		infiniband_switch_hardware_info{firmware_version="27.2010.3118",guid="0x506b4b03005c2740",part_number="MQM8790-HS2F",psid="MT_0000000063",serial_number="MT2152T10239",switch="iswr0l1"} 1
		# HELP infiniband_switch_power_supply_dc_power_status_info Constant 1 with the current DC power status string label (1 series per PSU per state).
		# TYPE infiniband_switch_power_supply_dc_power_status_info gauge
		infiniband_switch_power_supply_dc_power_status_info{guid="0x506b4b03005c2740",psu="0",status="OK",switch="iswr0l1"} 1
		infiniband_switch_power_supply_dc_power_status_info{guid="0x506b4b03005c2740",psu="1",status="OK",switch="iswr0l1"} 1
		infiniband_switch_power_supply_dc_power_status_info{guid="0x7cfe9003009ce5b0",psu="0",status="OK",switch="iswr1l1"} 1
		infiniband_switch_power_supply_dc_power_status_info{guid="0x7cfe9003009ce5b0",psu="1",status="OK",switch="iswr1l1"} 1
		# HELP infiniband_switch_power_supply_fan_status_info Constant 1 with the current PSU fan status string label (1 series per PSU per state).
		# TYPE infiniband_switch_power_supply_fan_status_info gauge
		infiniband_switch_power_supply_fan_status_info{guid="0x506b4b03005c2740",psu="0",status="OK",switch="iswr0l1"} 1
		infiniband_switch_power_supply_fan_status_info{guid="0x506b4b03005c2740",psu="1",status="OK",switch="iswr0l1"} 1
		infiniband_switch_power_supply_fan_status_info{guid="0x7cfe9003009ce5b0",psu="0",status="OK",switch="iswr1l1"} 1
		infiniband_switch_power_supply_fan_status_info{guid="0x7cfe9003009ce5b0",psu="1",status="OK",switch="iswr1l1"} 1
		# HELP infiniband_switch_power_supply_status_info Constant 1 with the current PSU status string label (1 series per PSU per state).
		# TYPE infiniband_switch_power_supply_status_info gauge
		infiniband_switch_power_supply_status_info{guid="0x506b4b03005c2740",psu="0",status="OK",switch="iswr0l1"} 1
		infiniband_switch_power_supply_status_info{guid="0x506b4b03005c2740",psu="1",status="OK",switch="iswr0l1"} 1
		infiniband_switch_power_supply_status_info{guid="0x7cfe9003009ce5b0",psu="0",status="OK",switch="iswr1l1"} 1
		infiniband_switch_power_supply_status_info{guid="0x7cfe9003009ce5b0",psu="1",status="OK",switch="iswr1l1"} 1
		# HELP infiniband_switch_power_supply_watts Power drawn by the PSU in watts.
		# TYPE infiniband_switch_power_supply_watts gauge
		infiniband_switch_power_supply_watts{guid="0x506b4b03005c2740",psu="0",switch="iswr0l1"} 154
		infiniband_switch_power_supply_watts{guid="0x506b4b03005c2740",psu="1",switch="iswr0l1"} 134
		infiniband_switch_power_supply_watts{guid="0x7cfe9003009ce5b0",psu="0",switch="iswr1l1"} 72
		infiniband_switch_power_supply_watts{guid="0x7cfe9003009ce5b0",psu="1",switch="iswr1l1"} 71
		# HELP infiniband_switch_temperature_celsius Switch ASIC temperature in degrees Celsius.
		# TYPE infiniband_switch_temperature_celsius gauge
		infiniband_switch_temperature_celsius{guid="0x506b4b03005c2740",switch="iswr0l1"} 53
		infiniband_switch_temperature_celsius{guid="0x7cfe9003009ce5b0",switch="iswr1l1"} 45
		# HELP infiniband_switch_uptime_seconds Switch firmware uptime in seconds since last reboot.
		# TYPE infiniband_switch_uptime_seconds gauge
		infiniband_switch_uptime_seconds{guid="0x506b4b03005c2740",switch="iswr0l1"} 8301347
        infiniband_switch_uptime_seconds{guid="0x7cfe9003009ce5b0",switch="iswr1l1"} 13862333
	`
	collector := NewIbswinfoCollector(&switchDevices, false, slog.New(slog.DiscardHandler))
	gatherers := setupGatherer(collector)
	if val, err := testutil.GatherAndCount(gatherers); err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if val != 52 {
		t.Errorf("Unexpected collection count %d, expected 52", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected),
		"infiniband_switch_power_supply_status_info", "infiniband_switch_power_supply_dc_power_status_info",
		"infiniband_switch_power_supply_fan_status_info", "infiniband_switch_power_supply_watts",
		"infiniband_switch_temperature_celsius", "infiniband_switch_fan_status_info", "infiniband_switch_fan_rpm",
		"infiniband_switch_hardware_info", "infiniband_switch_uptime_seconds",
		"infiniband_exporter_collect_errors", "infiniband_exporter_collect_timeouts"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func TestIbswinfoCollectorMissingStatus(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--ibswinfo.static-cache-ttl=0"}); err != nil {
		t.Fatal(err)
	}
	IbswinfoExec = func(lid string, vitals bool, ctx context.Context) (string, error) {
		if lid == "1719" {
			out, err := ReadFixture("ibswinfo", "test1-missing")
			return out, err
		} else if lid == "2052" {
			out, err := ReadFixture("ibswinfo", "test2")
			return out, err
		} else {
			return "", nil
		}
	}
	collector := NewIbswinfoCollector(&switchDevices, false, slog.New(slog.DiscardHandler))
	gatherers := setupGatherer(collector)
	if val, err := testutil.GatherAndCount(gatherers); err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if val != 45 {
		t.Errorf("Unexpected collection count %d, expected 45", val)
	}
}

func TestIbswinfoCollectorError(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--ibswinfo.static-cache-ttl=0"}); err != nil {
		t.Fatal(err)
	}
	IbswinfoExec = func(lid string, vitals bool, ctx context.Context) (string, error) {
		var out string
		var err error
		if lid == "1719" {
			out, _ = ReadFixture("ibswinfo", "test-err1")
			err = nil
		} else if lid == "2052" {
			out = ""
			err = fmt.Errorf("Error")
		}
		return out, err
	}
	expected := `
		# HELP infiniband_exporter_collect_errors Number of errors that occurred during collection
		# TYPE infiniband_exporter_collect_errors gauge
		infiniband_exporter_collect_errors{collector="ibswinfo"} 2
		# HELP infiniband_exporter_collect_timeouts Number of timeouts that occurred during collection
		# TYPE infiniband_exporter_collect_timeouts gauge
		infiniband_exporter_collect_timeouts{collector="ibswinfo"} 0
	`
	w := os.Stderr
	logger := slog.New(slog.NewTextHandler(w, nil))
	collector := NewIbswinfoCollector(&switchDevices, false, logger)
	gatherers := setupGatherer(collector)
	if val, err := testutil.GatherAndCount(gatherers); err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if val != 3 {
		t.Errorf("Unexpected collection count %d, expected 3", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected),
		"infiniband_switch_power_supply_status_info",
		"infiniband_exporter_collect_errors", "infiniband_exporter_collect_timeouts"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func TestIbswinfoCollectorErrorRunonce(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--ibswinfo.static-cache-ttl=0"}); err != nil {
		t.Fatal(err)
	}
	IbswinfoExec = func(lid string, vitals bool, ctx context.Context) (string, error) {
		var out string
		var err error
		if lid == "1719" {
			out, _ = ReadFixture("ibswinfo", "test-err1")
			err = nil
		} else if lid == "2052" {
			out = ""
			err = fmt.Errorf("Error")
		}
		return out, err
	}
	expected := `
		# HELP infiniband_exporter_collect_errors Number of errors that occurred during collection
		# TYPE infiniband_exporter_collect_errors gauge
		infiniband_exporter_collect_errors{collector="ibswinfo-runonce"} 2
		# HELP infiniband_exporter_collect_timeouts Number of timeouts that occurred during collection
		# TYPE infiniband_exporter_collect_timeouts gauge
		infiniband_exporter_collect_timeouts{collector="ibswinfo-runonce"} 0
	`
	collector := NewIbswinfoCollector(&switchDevices, true, slog.New(slog.DiscardHandler))
	gatherers := setupGatherer(collector)
	if val, err := testutil.GatherAndCount(gatherers); err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if val != 4 {
		t.Errorf("Unexpected collection count %d, expected 4", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected),
		"infiniband_switch_power_supply_status_info",
		"infiniband_exporter_collect_errors", "infiniband_exporter_collect_timeouts"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func TestIbswinfoCollectorTimeout(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--ibswinfo.static-cache-ttl=0"}); err != nil {
		t.Fatal(err)
	}
	IbswinfoExec = func(lid string, vitals bool, ctx context.Context) (string, error) {
		return "", context.DeadlineExceeded
	}
	expected := `
		# HELP infiniband_exporter_collect_errors Number of errors that occurred during collection
		# TYPE infiniband_exporter_collect_errors gauge
		infiniband_exporter_collect_errors{collector="ibswinfo"} 0
		# HELP infiniband_exporter_collect_timeouts Number of timeouts that occurred during collection
		# TYPE infiniband_exporter_collect_timeouts gauge
		infiniband_exporter_collect_timeouts{collector="ibswinfo"} 2
	`
	collector := NewIbswinfoCollector(&switchDevices, false, slog.New(slog.DiscardHandler))
	gatherers := setupGatherer(collector)
	if val, err := testutil.GatherAndCount(gatherers); err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if val != 3 {
		t.Errorf("Unexpected collection count %d, expected 3", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected),
		"infiniband_switch_power_supply_status_info",
		"infiniband_exporter_collect_errors", "infiniband_exporter_collect_timeouts"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func TestIbswinfoArgs(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--ibswinfo.static-cache-ttl=0"}); err != nil {
		t.Fatal(err)
	}
	trueValue := true
	falseValue := false
	command, args := ibswinfoArgs("100", false)
	if command != "ibswinfo" {
		t.Errorf("Unexpected command, got: %s", command)
	}
	expectedArgs := []string{"-d", "lid-100"}
	if !reflect.DeepEqual(args, expectedArgs) {
		t.Errorf("Unexpected args\nExpected\n%v\nGot\n%v", expectedArgs, args)
	}
	useSudo = &trueValue
	command, args = ibswinfoArgs("100", false)
	if command != "sudo" {
		t.Errorf("Unexpected command, got: %s", command)
	}
	expectedArgs = []string{"ibswinfo", "-d", "lid-100"}
	if !reflect.DeepEqual(args, expectedArgs) {
		t.Errorf("Unexpected args\nExpected\n%v\nGot\n%v", expectedArgs, args)
	}
	useSudo = &falseValue
}

func TestIBSWInfo(t *testing.T) {
	execCommand = fakeExecCommand
	mockedExitStatus = 0
	mockedStdout = "foo"
	defer func() { execCommand = exec.CommandContext }()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := ibswinfo("1", false, ctx)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	if out != mockedStdout {
		t.Errorf("Unexpected out: %s", out)
	}
}

func TestIBSWInfoError(t *testing.T) {
	execCommand = fakeExecCommand
	mockedExitStatus = 1
	mockedStdout = "foo"
	defer func() { execCommand = exec.CommandContext }()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := ibswinfo("1", false, ctx)
	if err == nil {
		t.Errorf("Expected error")
	}
	if out != "" {
		t.Errorf("Unexpected out: %s", out)
	}
}

func TestIBSWInfoTimeout(t *testing.T) {
	execCommand = fakeExecCommand
	mockedExitStatus = 1
	mockedStdout = "foo"
	defer func() { execCommand = exec.CommandContext }()
	ctx, cancel := context.WithTimeout(context.Background(), 0*time.Second)
	defer cancel()
	out, err := ibswinfo("1", false, ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded")
	}
	if out != "" {
		t.Errorf("Unexpected out: %s", out)
	}
}

// TestParseIbswinfoVitals exercises the vitals-format parser against a
// representative fixture (separator ":", different keys than the full
// output). It does not exercise the cache merge path — that is covered
// by TestIbswinfoCollectorCacheHit below.
func TestParseIbswinfoVitals(t *testing.T) {
	out, err := ReadFixture("ibswinfo", "vitals1")
	if err != nil {
		t.Fatal(err.Error())
	}
	var data Ibswinfo
	if err := parseIbswinfoVitals(out, &data, slog.New(slog.DiscardHandler)); err != nil {
		t.Fatalf("Unexpected parse error: %s", err)
	}
	if data.Uptime != 16982312 {
		t.Errorf("Unexpected uptime: %f", data.Uptime)
	}
	if data.Temp != 73 {
		t.Errorf("Unexpected temp: %f", data.Temp)
	}
	if len(data.PowerSupplies) != 2 {
		t.Errorf("Unexpected PSU count: %d", len(data.PowerSupplies))
	}
	if len(data.Fans) != 9 {
		t.Errorf("Unexpected fan count: %d", len(data.Fans))
	}
	// Static fields must remain empty: the vitals format does not carry
	// them; the collector merges them from the cache.
	if data.PartNumber != "" || data.SerialNumber != "" || data.PSID != "" || data.FirmwareVersion != "" {
		t.Errorf("Static fields should be empty in vitals parse, got: %+v", data)
	}
}

// TestIbswinfoCollectorCacheMissFull verifies that the first scrape with a
// non-zero TTL goes through the full path (vitals=false) — there is
// nothing in the cache yet.
func TestIbswinfoCollectorCacheMissFull(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--ibswinfo.static-cache-ttl=15m"}); err != nil {
		t.Fatal(err)
	}
	var fullCalls, vitalsCalls atomic.Uint64
	IbswinfoExec = func(lid string, vitals bool, ctx context.Context) (string, error) {
		if vitals {
			vitalsCalls.Add(1)
		} else {
			fullCalls.Add(1)
		}
		out, err := ReadFixture("ibswinfo", "test1")
		return out, err
	}
	collector := NewIbswinfoCollector(&switchDevices, false, slog.New(slog.DiscardHandler))
	gatherers := setupGatherer(collector)
	if _, err := testutil.GatherAndCount(gatherers); err != nil {
		t.Fatal(err)
	}
	if got := fullCalls.Load(); got != uint64(len(switchDevices)) {
		t.Errorf("Expected %d full calls on cold cache, got %d", len(switchDevices), got)
	}
	if got := vitalsCalls.Load(); got != 0 {
		t.Errorf("Expected 0 vitals calls on cold cache, got %d", got)
	}
}

// TestIbswinfoCollectorCacheHitVitals verifies the second scrape uses the
// vitals path and that static fields are merged back from the cache.
func TestIbswinfoCollectorCacheHitVitals(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--ibswinfo.static-cache-ttl=15m"}); err != nil {
		t.Fatal(err)
	}
	var fullCalls, vitalsCalls atomic.Uint64
	IbswinfoExec = func(lid string, vitals bool, ctx context.Context) (string, error) {
		if vitals {
			vitalsCalls.Add(1)
			out, err := ReadFixture("ibswinfo", "vitals1")
			return out, err
		}
		fullCalls.Add(1)
		out, err := ReadFixture("ibswinfo", "test1")
		return out, err
	}
	collector := NewIbswinfoCollector(&switchDevices, false, slog.New(slog.DiscardHandler))
	gatherers := setupGatherer(collector)
	// Warm the cache.
	if _, err := testutil.GatherAndCount(gatherers); err != nil {
		t.Fatal(err)
	}
	// Second scrape should use vitals for every device.
	if _, err := testutil.GatherAndCount(gatherers); err != nil {
		t.Fatal(err)
	}
	if got := fullCalls.Load(); got != uint64(len(switchDevices)) {
		t.Errorf("Expected %d full calls (cold cache), got %d", len(switchDevices), got)
	}
	if got := vitalsCalls.Load(); got != uint64(len(switchDevices)) {
		t.Errorf("Expected %d vitals calls (warm cache), got %d", len(switchDevices), got)
	}
	// hardware_info should still be present after the vitals scrape (merged
	// from the cache). One per device.
	expected := `
		# HELP infiniband_switch_hardware_info Constant 1 carrying switch hardware identification labels (firmware version, PSID, part/serial numbers).
		# TYPE infiniband_switch_hardware_info gauge
		infiniband_switch_hardware_info{firmware_version="11.2008.2102",guid="0x506b4b03005c2740",part_number="MSB7790-ES2F",psid="MT_1880110032",serial_number="MT1943X00498",switch="iswr0l1"} 1
		infiniband_switch_hardware_info{firmware_version="11.2008.2102",guid="0x7cfe9003009ce5b0",part_number="MSB7790-ES2F",psid="MT_1880110032",serial_number="MT1943X00498",switch="iswr1l1"} 1
	`
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected),
		"infiniband_switch_hardware_info"); err != nil {
		t.Errorf("static fields not merged from cache after vitals scrape:\n%s", err)
	}
}

// TestIbswinfoCollectorCacheExpired verifies that an expired entry forces
// a full scrape again.
func TestIbswinfoCollectorCacheExpired(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--ibswinfo.static-cache-ttl=10ms"}); err != nil {
		t.Fatal(err)
	}
	var fullCalls atomic.Uint64
	IbswinfoExec = func(lid string, vitals bool, ctx context.Context) (string, error) {
		if !vitals {
			fullCalls.Add(1)
		}
		out, err := ReadFixture("ibswinfo", "test1")
		return out, err
	}
	collector := NewIbswinfoCollector(&switchDevices, false, slog.New(slog.DiscardHandler))
	gatherers := setupGatherer(collector)
	if _, err := testutil.GatherAndCount(gatherers); err != nil {
		t.Fatal(err)
	}
	// Wait past the TTL to force re-population.
	time.Sleep(20 * time.Millisecond)
	if _, err := testutil.GatherAndCount(gatherers); err != nil {
		t.Fatal(err)
	}
	want := uint64(2 * len(switchDevices)) // two full passes
	if got := fullCalls.Load(); got != want {
		t.Errorf("Expected %d full calls after TTL expiry, got %d", want, got)
	}
}

// TestIbswinfoCollectorCacheDisabled verifies TTL=0 reproduces pre-v0.15
// behaviour: every scrape is a full call.
func TestIbswinfoCollectorCacheDisabled(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--ibswinfo.static-cache-ttl=0"}); err != nil {
		t.Fatal(err)
	}
	var fullCalls, vitalsCalls atomic.Uint64
	IbswinfoExec = func(lid string, vitals bool, ctx context.Context) (string, error) {
		if vitals {
			vitalsCalls.Add(1)
		} else {
			fullCalls.Add(1)
		}
		out, err := ReadFixture("ibswinfo", "test1")
		return out, err
	}
	collector := NewIbswinfoCollector(&switchDevices, false, slog.New(slog.DiscardHandler))
	gatherers := setupGatherer(collector)
	for i := 0; i < 3; i++ {
		if _, err := testutil.GatherAndCount(gatherers); err != nil {
			t.Fatal(err)
		}
	}
	if got := vitalsCalls.Load(); got != 0 {
		t.Errorf("Expected 0 vitals calls when TTL=0, got %d", got)
	}
	want := uint64(3 * len(switchDevices))
	if got := fullCalls.Load(); got != want {
		t.Errorf("Expected %d full calls when TTL=0, got %d", want, got)
	}
}
