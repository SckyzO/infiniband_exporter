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
	"bytes"
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"log/slog"

	kingpin "github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	CollectIbswinfo = kingpin.Flag("collector.ibswinfo", "Enable ibswinfo data collection (BETA)").Default("false").Bool()
	ibswinfoPath    = kingpin.Flag("ibswinfo.path", "Path to ibswinfo").Default("ibswinfo").String()
	ibswinfoTimeout = kingpin.Flag("ibswinfo.timeout", "Timeout for ibswinfo execution").Default("10s").Duration()
	// ibswinfo is ~1.4 s per switch on HDR fabrics; 4 in flight is the
	// observed sweet spot (~4× faster) before the SMA starts contending.
	ibswinfoMaxConcurrent = kingpin.Flag("ibswinfo.max-concurrent", "Max number of concurrent ibswinfo executions").Default("4").Int()
	// While the static-field cache is fresh, scrapes use the lighter
	// `ibswinfo -o vitals` mode (dynamic registers only) and merge the
	// cached PartNumber / SerialNumber / PSID / FirmwareVersion back in.
	// 0 disables the optimization.
	ibswinfoStaticCacheTTL                  = kingpin.Flag("ibswinfo.static-cache-ttl", "TTL for caching static ibswinfo fields. 0 disables the cache.").Default("15m").Duration()
	IbswinfoExec           IbswinfoExecFunc = ibswinfo
)

// IbswinfoExecFunc lets tests substitute the underlying ibswinfo invocation.
// vitals=true requests the lightweight `-o vitals` output mode.
type IbswinfoExecFunc func(lid string, vitals bool, ctx context.Context) (string, error)

// ibswinfoCacheEntry stores the fields that practically never change between
// scrapes (hardware identifiers and firmware version).
type ibswinfoCacheEntry struct {
	PartNumber      string
	SerialNumber    string
	PSID            string
	FirmwareVersion string
	lastRefresh     time.Time
}

// ibswinfoStaticCache is a package-global keyed by GUID. It survives
// between setupGathers() invocations — Prometheus's HTTP handler
// re-builds an IbswinfoCollector on every scrape, so per-instance
// state is reset each time. ibnetdiscoverCache uses the same pattern.
var ibswinfoStaticCache sync.Map // map[guid]ibswinfoCacheEntry

type IbswinfoCollector struct {
	devices              *[]InfinibandDevice
	logger               *slog.Logger
	collector            string
	Duration             *prometheus.Desc
	Error                *prometheus.Desc
	Timeout              *prometheus.Desc
	HardwareInfo         *prometheus.Desc
	Uptime               *prometheus.Desc
	PowerSupplyStatus    *prometheus.Desc
	PowerSupplyDCPower   *prometheus.Desc
	PowerSupplyFanStatus *prometheus.Desc
	PowerSupplyWatts     *prometheus.Desc
	Temp                 *prometheus.Desc
	FanStatus            *prometheus.Desc
	FanRPM               *prometheus.Desc
	Up                   *prometheus.Desc
}

type Ibswinfo struct {
	device          InfinibandDevice
	PartNumber      string
	SerialNumber    string
	PSID            string
	FirmwareVersion string
	Uptime          float64
	PowerSupplies   []SwitchPowerSupply
	Temp            float64
	FanStatus       string
	Fans            []SwitchFan
	duration        float64
	error           float64
	timeout         float64
}

type SwitchPowerSupply struct {
	ID        string
	Status    string
	DCPower   string
	FanStatus string
	PowerW    float64
}

type SwitchFan struct {
	ID  string
	RPM float64
}

func NewIbswinfoCollector(devices *[]InfinibandDevice, runonce bool, logger *slog.Logger) *IbswinfoCollector {
	collector := "ibswinfo"
	if runonce {
		collector = "ibswinfo-runonce"
	}
	return &IbswinfoCollector{
		devices:   devices,
		logger:    logger.With("collector", collector),
		collector: collector,
		// "ibswinfo" subsystem (not "switch") so these descriptors do not
		// collide with SwitchCollector when both are registered.
		Duration: prometheus.NewDesc(prometheus.BuildFQName(namespace, "ibswinfo", "collect_duration_seconds"),
			"Time spent collecting metrics for this device, in seconds.", []string{"guid", "collector", "switch"}, nil),
		Error: prometheus.NewDesc(prometheus.BuildFQName(namespace, "ibswinfo", "collect_error"),
			"1 if the most recent collection for this device errored, 0 otherwise.", []string{"guid", "collector", "switch"}, nil),
		Timeout: prometheus.NewDesc(prometheus.BuildFQName(namespace, "ibswinfo", "collect_timeout"),
			"1 if the most recent collection for this device timed out, 0 otherwise.", []string{"guid", "collector", "switch"}, nil),
		HardwareInfo: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "hardware_info"),
			"Constant 1 carrying switch hardware identification labels (firmware version, PSID, part/serial numbers).", []string{"guid", "firmware_version", "psid", "part_number", "serial_number", "switch"}, nil),
		Uptime: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "uptime_seconds"),
			"Switch firmware uptime in seconds since last reboot.", []string{"guid", "switch"}, nil),
		PowerSupplyStatus: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "power_supply_status_info"),
			"Constant 1 with the current PSU status string label (1 series per PSU per state).", []string{"guid", "psu", "status", "switch"}, nil),
		PowerSupplyDCPower: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "power_supply_dc_power_status_info"),
			"Constant 1 with the current DC power status string label (1 series per PSU per state).", []string{"guid", "psu", "status", "switch"}, nil),
		PowerSupplyFanStatus: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "power_supply_fan_status_info"),
			"Constant 1 with the current PSU fan status string label (1 series per PSU per state).", []string{"guid", "psu", "status", "switch"}, nil),
		PowerSupplyWatts: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "power_supply_watts"),
			"Power drawn by the PSU in watts.", []string{"guid", "psu", "switch"}, nil),
		Temp: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "temperature_celsius"),
			"Switch ASIC temperature in degrees Celsius.", []string{"guid", "switch"}, nil),
		FanStatus: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "fan_status_info"),
			"Constant 1 with the current overall fan status string label.", []string{"guid", "status", "switch"}, nil),
		FanRPM: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "fan_rpm"),
			"Switch fan rotation speed in RPM (one series per fan).", []string{"guid", "fan", "switch"}, nil),
		Up: prometheus.NewDesc(prometheus.BuildFQName(namespace, "ibswinfo", "up"),
			"1 if the latest ibswinfo scrape of this switch succeeded, 0 otherwise (timeout or error).", []string{"guid", "switch"}, nil),
	}
}

func (s *IbswinfoCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- s.Duration
	ch <- s.Error
	ch <- s.Timeout
	ch <- s.HardwareInfo
	ch <- s.Uptime
	ch <- s.PowerSupplyStatus
	ch <- s.PowerSupplyDCPower
	ch <- s.PowerSupplyFanStatus
	ch <- s.PowerSupplyWatts
	ch <- s.Temp
	ch <- s.FanStatus
	ch <- s.FanRPM
	ch <- s.Up
}

func (s *IbswinfoCollector) Collect(ch chan<- prometheus.Metric) {
	collectTime := time.Now()
	swinfos, errors, timeouts := s.collect()
	for _, swinfo := range swinfos {
		ch <- prometheus.MustNewConstMetric(s.HardwareInfo, prometheus.GaugeValue, 1, swinfo.device.GUID,
			swinfo.FirmwareVersion, swinfo.PSID, swinfo.PartNumber, swinfo.SerialNumber, swinfo.device.Name)
		ch <- prometheus.MustNewConstMetric(s.Uptime, prometheus.GaugeValue, swinfo.Uptime, swinfo.device.GUID, swinfo.device.Name)
		ch <- prometheus.MustNewConstMetric(s.Duration, prometheus.GaugeValue, swinfo.duration, swinfo.device.GUID, s.collector, swinfo.device.Name)
		ch <- prometheus.MustNewConstMetric(s.Error, prometheus.GaugeValue, swinfo.error, swinfo.device.GUID, s.collector, swinfo.device.Name)
		ch <- prometheus.MustNewConstMetric(s.Timeout, prometheus.GaugeValue, swinfo.timeout, swinfo.device.GUID, s.collector, swinfo.device.Name)
		up := 1 - swinfo.error - swinfo.timeout
		ch <- prometheus.MustNewConstMetric(s.Up, prometheus.GaugeValue, up, swinfo.device.GUID, swinfo.device.Name)
		for _, psu := range swinfo.PowerSupplies {
			if psu.Status != "" {
				ch <- prometheus.MustNewConstMetric(s.PowerSupplyStatus, prometheus.GaugeValue, 1, swinfo.device.GUID, psu.ID, psu.Status, swinfo.device.Name)
			}
			if psu.DCPower != "" {
				ch <- prometheus.MustNewConstMetric(s.PowerSupplyDCPower, prometheus.GaugeValue, 1, swinfo.device.GUID, psu.ID, psu.DCPower, swinfo.device.Name)
			}
			if psu.FanStatus != "" {
				ch <- prometheus.MustNewConstMetric(s.PowerSupplyFanStatus, prometheus.GaugeValue, 1, swinfo.device.GUID, psu.ID, psu.FanStatus, swinfo.device.Name)
			}
			if !math.IsNaN(psu.PowerW) {
				ch <- prometheus.MustNewConstMetric(s.PowerSupplyWatts, prometheus.GaugeValue, psu.PowerW, swinfo.device.GUID, psu.ID, swinfo.device.Name)
			}
		}
		if !math.IsNaN(swinfo.Temp) {
			ch <- prometheus.MustNewConstMetric(s.Temp, prometheus.GaugeValue, swinfo.Temp, swinfo.device.GUID, swinfo.device.Name)
		}
		if swinfo.FanStatus != "" {
			ch <- prometheus.MustNewConstMetric(s.FanStatus, prometheus.GaugeValue, 1, swinfo.device.GUID, swinfo.FanStatus, swinfo.device.Name)
		}
		for _, fan := range swinfo.Fans {
			if !math.IsNaN(fan.RPM) {
				ch <- prometheus.MustNewConstMetric(s.FanRPM, prometheus.GaugeValue, fan.RPM, swinfo.device.GUID, fan.ID, swinfo.device.Name)
			}
		}
	}
	ch <- prometheus.MustNewConstMetric(collectErrors, prometheus.GaugeValue, errors, s.collector)
	ch <- prometheus.MustNewConstMetric(collecTimeouts, prometheus.GaugeValue, timeouts, s.collector)
	ch <- prometheus.MustNewConstMetric(collectDuration, prometheus.GaugeValue, time.Since(collectTime).Seconds(), s.collector)
	if strings.HasSuffix(s.collector, "-runonce") {
		ch <- prometheus.MustNewConstMetric(lastExecution, prometheus.GaugeValue, float64(time.Now().Unix()), s.collector)
	}
}

// useVitalsForGUID returns true when a fresh static-cache entry exists for
// the device. It is the only place that reads --ibswinfo.static-cache-ttl;
// a TTL of 0 means "always full".
func (s *IbswinfoCollector) useVitalsForGUID(guid string) (ibswinfoCacheEntry, bool) {
	ttl := *ibswinfoStaticCacheTTL
	if ttl <= 0 {
		return ibswinfoCacheEntry{}, false
	}
	v, ok := ibswinfoStaticCache.Load(guid)
	if !ok {
		return ibswinfoCacheEntry{}, false
	}
	entry := v.(ibswinfoCacheEntry)
	if time.Since(entry.lastRefresh) >= ttl {
		return entry, false
	}
	return entry, true
}

func (s *IbswinfoCollector) collect() ([]Ibswinfo, float64, float64) {
	var ibswinfos []Ibswinfo
	var ibswinfosLock sync.Mutex
	var errors, timeouts uint64
	limit := make(chan int, *ibswinfoMaxConcurrent)
	wg := &sync.WaitGroup{}
	s.logger.Debug("Collecting ibswinfo on devices", "count", len(*s.devices))
	for _, device := range *s.devices {
		limit <- 1
		wg.Add(1)
		go func(device InfinibandDevice) {
			defer func() {
				<-limit
				wg.Done()
			}()
			ctxibswinfo, cancelibswinfo := context.WithTimeout(context.Background(), *ibswinfoTimeout)
			defer cancelibswinfo()
			cached, useVitals := s.useVitalsForGUID(device.GUID)
			s.logger.Debug("Run ibswinfo", "lid", device.LID, "vitals", useVitals)
			start := time.Now()
			ibswinfoOut, ibswinfoErr := IbswinfoExec(device.LID, useVitals, ctxibswinfo)
			ibswinfoData := Ibswinfo{duration: time.Since(start).Seconds()}
			if ibswinfoErr == context.DeadlineExceeded {
				ibswinfoData.timeout = 1
				s.logger.Error("Timeout collecting ibswinfo data", "guid", device.GUID, "lid", device.LID)
				atomic.AddUint64(&timeouts, 1)
			} else if ibswinfoErr != nil {
				ibswinfoData.error = 1
				s.logger.Error("Error collecting ibswinfo data", "err", fmt.Sprintf("%s:%s", ibswinfoErr, ibswinfoOut), "guid", device.GUID, "lid", device.LID)
				atomic.AddUint64(&errors, 1)
			}
			if ibswinfoErr == nil {
				var parseErr error
				if useVitals {
					parseErr = parseIbswinfoVitals(ibswinfoOut, &ibswinfoData, s.logger)
					if parseErr == nil {
						// Merge the static fields we kept off the wire.
						ibswinfoData.PartNumber = cached.PartNumber
						ibswinfoData.SerialNumber = cached.SerialNumber
						ibswinfoData.PSID = cached.PSID
						ibswinfoData.FirmwareVersion = cached.FirmwareVersion
					}
				} else {
					parseErr = parse_ibswinfo(ibswinfoOut, &ibswinfoData, s.logger)
					if parseErr == nil && *ibswinfoStaticCacheTTL > 0 {
						ibswinfoStaticCache.Store(device.GUID, ibswinfoCacheEntry{
							PartNumber:      ibswinfoData.PartNumber,
							SerialNumber:    ibswinfoData.SerialNumber,
							PSID:            ibswinfoData.PSID,
							FirmwareVersion: ibswinfoData.FirmwareVersion,
							lastRefresh:     time.Now(),
						})
					}
				}
				if parseErr != nil {
					s.logger.Error("Error parsing ibswinfo output", "guid", device.GUID, "lid", device.LID, "vitals", useVitals)
					atomic.AddUint64(&errors, 1)
				} else {
					ibswinfoData.device = device
					ibswinfosLock.Lock()
					ibswinfos = append(ibswinfos, ibswinfoData)
					ibswinfosLock.Unlock()
				}
			}
		}(device)
	}
	wg.Wait()
	close(limit)
	return ibswinfos, float64(errors), float64(timeouts)
}

func ibswinfoArgs(lid string, vitals bool) (string, []string) {
	var command string
	var args []string
	if *useSudo {
		command = "sudo"
		args = []string{*ibswinfoPath}
	} else {
		command = *ibswinfoPath
	}
	args = append(args, "-d", fmt.Sprintf("lid-%s", lid))
	if vitals {
		// `-o vitals` skips the static MFT registers (MSGI/MSCI/SPZR)
		// and only reads MGIR/MGPIR/MSPS/MTMP/MTCAP/MFCR — significantly
		// faster on every scrape, but the output format and key set are
		// different (parse_ibswinfo_vitals handles it).
		args = append(args, "-o", "vitals")
	}
	return command, args
}

func ibswinfo(lid string, vitals bool, ctx context.Context) (string, error) {
	command, args := ibswinfoArgs(lid, vitals)
	cmd := execCommand(ctx, command, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return "", ctx.Err()
	} else if err != nil {
		return stderr.String(), err
	}
	return stdout.String(), nil
}

func parse_ibswinfo(out string, data *Ibswinfo, logger *slog.Logger) error {
	data.Temp = math.NaN()
	lines := strings.Split(out, "\n")
	psus := make(map[string]SwitchPowerSupply)
	var err error
	var powerSupplies []SwitchPowerSupply
	var fans []SwitchFan
	var psuID string
	var dividerCount int
	rePSU := regexp.MustCompile(`PSU([0-9]) status`)
	reFan := regexp.MustCompile(`fan#([0-9]+)`)
	for _, line := range lines {
		if strings.HasPrefix(line, "-----") {
			dividerCount++
		}
		l := strings.Split(line, "|")
		if len(l) != 2 {
			continue
		}
		key := strings.TrimSpace(l[0])
		value := strings.TrimSpace(l[1])
		switch key {
		case "part number":
			data.PartNumber = value
		case "serial number":
			data.SerialNumber = value
		case "PSID":
			data.PSID = value
		case "firmware version":
			data.FirmwareVersion = value
		}
		if strings.HasPrefix(key, "uptime") {
			// Convert Nd-H:M:S to time that ParseDuration understands
			var days float64
			uptimeHMS := ""
			uptime_s1 := strings.Split(value, "-")
			if len(uptime_s1) == 2 {
				daysStr := strings.Replace(uptime_s1[0], "d", "", 1)
				days, err = strconv.ParseFloat(daysStr, 64)
				if err != nil {
					logger.Error("Unable to parse uptime duration", "err", err, "value", value)
					continue
				}
				uptimeHMS = uptime_s1[1]
			} else {
				uptimeHMS = value
			}
			t1, err := time.Parse("15:04:05", uptimeHMS)
			if err != nil {
				logger.Error("Unable to parse uptime duration", "err", err, "value", value)
				continue
			}
			t2, _ := time.Parse("15:04:05", "00:00:00")
			data.Uptime = (days * 86400) + t1.Sub(t2).Seconds()
		}
		var psu SwitchPowerSupply
		psu.PowerW = math.NaN()
		matchesPSU := rePSU.FindStringSubmatch(key)
		if len(matchesPSU) == 2 {
			psuID = matchesPSU[1]
			psu.Status = value
		}
		if psu.Status == "" && psuID != "" && dividerCount < 4 {
			if p, ok := psus[psuID]; ok {
				psu = p
			}
		}
		if key == "DC power" {
			psu.DCPower = value
		}
		if key == "fan status" && dividerCount < 4 {
			psu.FanStatus = value
		}
		if key == "power (W)" {
			powerW, err := strconv.ParseFloat(value, 64)
			if err == nil {
				psu.PowerW = powerW
			} else {
				logger.Error("Unable to parse power (W)", "err", err, "value", value)
				return err
			}
		}
		if psuID != "" && dividerCount < 4 {
			psus[psuID] = psu
		}
		if key == "temperature (C)" {
			temp, err := strconv.ParseFloat(value, 64)
			if err == nil {
				data.Temp = temp
			} else {
				logger.Error("Unable to parse temperature (C)", "err", err, "value", value)
				return err
			}
		}
		if key == "fan status" && dividerCount >= 4 {
			data.FanStatus = value
		}
		matchesFan := reFan.FindStringSubmatch(key)
		if len(matchesFan) == 2 {
			fan := SwitchFan{
				ID: matchesFan[1],
			}
			if value == "" {
				fan.RPM = math.NaN()
				fans = append(fans, fan)
				continue
			}
			rpm, err := strconv.ParseFloat(value, 64)
			if err == nil {
				fan := SwitchFan{
					ID:  matchesFan[1],
					RPM: rpm,
				}
				fans = append(fans, fan)
			} else {
				logger.Error("Unable to parse fan RPM", "err", err, "value", value)
				return err
			}
		}
	}
	for id, psu := range psus {
		psu.ID = id
		powerSupplies = append(powerSupplies, psu)
	}
	data.PowerSupplies = powerSupplies
	data.Fans = fans
	return nil
}

// parseIbswinfoVitals parses the output of `ibswinfo -d lid-X -o vitals`.
// The format differs from the default one: lines use ":" as separator, the
// keys are different (e.g. "uptime (sec)" instead of "uptime (d-h:m:s)"),
// and only dynamic fields are present — no part number, serial, PSID,
// firmware, or status flags. Static fields are merged from the cache by
// the caller.
//
// Sample input:
//
//	uptime (sec)       : 16982312
//	psu0.power (W)     : 92
//	psu1.power (W)     : 102
//	cur.temp (C)       : 73
//	max.temp (C)       : 80
//	fan#1.speed (rpm)  : 6355
func parseIbswinfoVitals(out string, data *Ibswinfo, logger *slog.Logger) error {
	data.Temp = math.NaN()
	rePSU := regexp.MustCompile(`^psu([0-9]+)\.power \(W\)$`)
	reFan := regexp.MustCompile(`^fan#([0-9]+)\.speed \(rpm\)$`)
	psus := make(map[string]SwitchPowerSupply)
	var fans []SwitchFan
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "uptime (sec)":
			sec, err := strconv.ParseFloat(value, 64)
			if err != nil {
				logger.Error("Unable to parse vitals uptime (sec)", "err", err, "value", value)
				continue
			}
			data.Uptime = sec
			continue
		case "cur.temp (C)":
			temp, err := strconv.ParseFloat(value, 64)
			if err != nil {
				logger.Error("Unable to parse vitals cur.temp (C)", "err", err, "value", value)
				continue
			}
			data.Temp = temp
			continue
		}
		if m := rePSU.FindStringSubmatch(key); len(m) == 2 {
			psu := SwitchPowerSupply{ID: m[1], PowerW: math.NaN()}
			if value != "" {
				powerW, err := strconv.ParseFloat(value, 64)
				if err != nil {
					logger.Error("Unable to parse vitals psu power (W)", "err", err, "psu", m[1], "value", value)
				} else {
					psu.PowerW = powerW
				}
			}
			psus[m[1]] = psu
			continue
		}
		if m := reFan.FindStringSubmatch(key); len(m) == 2 {
			fan := SwitchFan{ID: m[1], RPM: math.NaN()}
			if value != "" {
				rpm, err := strconv.ParseFloat(value, 64)
				if err != nil {
					logger.Error("Unable to parse vitals fan speed (rpm)", "err", err, "fan", m[1], "value", value)
				} else {
					fan.RPM = rpm
				}
			}
			fans = append(fans, fan)
		}
	}
	for id, psu := range psus {
		psu.ID = id
		data.PowerSupplies = append(data.PowerSupplies, psu)
	}
	data.Fans = fans
	return nil
}
