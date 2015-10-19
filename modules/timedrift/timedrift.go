// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

/* The timedrift module evaluate the local time of a target against
network time retrieved using NTP.

Usage documentation is online at http://mig.mozilla.org/doc/module_timedrift.html
*/
package timedrift /* import "mig.ninja/mig/modules/timedrift" */

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mig.ninja/mig/modules"
	"net"
	"os"
	"strings"
	"time"
)

type module struct {
}

func (m *module) NewRun() modules.Runner {
	return new(run)
}

func init() {
	modules.Register("timedrift", new(module))
}

type run struct {
	Parameters params
	Results    modules.Result
}

// a simple parameters structure, the format is arbitrary
type params struct {
	Drift string `json:"drift"`
}

type elements struct {
	HasCheckedDrift bool     `json:"hascheckeddrift"`
	IsWithinDrift   bool     `json:"iswithindrift,omitempty"`
	Drifts          []string `json:"drifts,omitempty"`
	LocalTime       string   `json:"localtime"`
}

type statistics struct {
	ExecTime string     `json:"exectime"`
	NtpStats []ntpstats `json:"ntpstats,omitempty"`
}

type ntpstats struct {
	Host      string    `json:"host"`
	Time      time.Time `json:"time"`
	Latency   string    `json:"latency"`
	Drift     string    `json:"drift"`
	Reachable bool      `json:"reachable"`
}

var NtpBackupPool = []string{
	`time.nist.gov`,
	`0.pool.ntp.org`,
	`1.pool.ntp.org`,
	`2.pool.ntp.org`,
	`3.pool.ntp.org`}

func (r *run) ValidateParameters() (err error) {
	if r.Parameters.Drift != "" {
		_, err = time.ParseDuration(r.Parameters.Drift)
	}
	return err
}

func (r *run) Run(in io.Reader) (out string) {
	var (
		stats   statistics
		el      elements
		drift   time.Duration
		ntpFile *os.File
		ntpScan *bufio.Scanner
		ntpPool []string
	)
	defer func() {
		if e := recover(); e != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", e))
			r.Results.Success = false
			buf, _ := json.Marshal(r.Results)
			out = string(buf[:])
		}
	}()
	el.LocalTime = time.Now().Format(time.RFC3339Nano)
	t1 := time.Now()
	err := modules.ReadInputParameters(in, &r.Parameters)
	if err != nil {
		panic(err)
	}
	err = r.ValidateParameters()
	if err != nil {
		panic(err)
	}
	// if drift is not set, skip the ntp test
	if r.Parameters.Drift == "" {
		r.Results.FoundAnything = true
		goto done
	}
	drift, err = time.ParseDuration(r.Parameters.Drift)
	if err != nil {
		panic(err)
	}
	// assume host has synched time and set to false if not true
	el.IsWithinDrift = true

	//Load ntp servers from /etc/ntp.conf
	ntpFile, err = os.Open("/etc/ntp.conf")
	if err != nil {
		r.Results.Errors = append(r.Results.Errors,
			fmt.Sprintf("Using backup NTP hosts. Failed to read /etc/ntp.conf with error '%v'", err))
	} else {
		defer ntpFile.Close()
		ntpScan = bufio.NewScanner(ntpFile)
		for ntpScan.Scan() {
			ntpFields := strings.Fields(ntpScan.Text())
			if len(ntpFields) < 2 {
				continue
			}
			if ntpFields[0] == "server" {
				ntpPool = append(ntpPool, ntpFields[1])
			}
		}
	}

	//Add our hardcoded online servers to the end of our ntpPool as fallbacks
	ntpPool = append(ntpPool, NtpBackupPool...)

	// attempt to get network time from each of the NTP servers, and exit
	// as soon as we get a valid result from one of them
	for _, ntpsrv := range ntpPool {
		t, lat, err := GetNetworkTime(ntpsrv)
		if err != nil {
			// failed to get network time, log a failure and try another one
			stats.NtpStats = append(stats.NtpStats, ntpstats{
				Host:      ntpsrv,
				Reachable: false,
			})
			continue
		}

		// compare network time to local time
		localtime := time.Now()
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
			continue
		}
		if localtime.Before(t.Add(-drift)) {
			el.IsWithinDrift = false
			el.Drifts = append(el.Drifts, fmt.Sprintf("Local time is behind ntp host %s by %s", ntpsrv, t.Sub(localtime).String()))
		} else if localtime.After(t.Add(drift)) {
			el.IsWithinDrift = false
			el.Drifts = append(el.Drifts, fmt.Sprintf("Local time is ahead of ntp host %s by %s", ntpsrv, localtime.Sub(t).String()))
		}
		stats.NtpStats = append(stats.NtpStats, ntpstats{
			Host:      ntpsrv,
			Time:      t,
			Latency:   lat,
			Drift:     localtime.Sub(t).String(),
			Reachable: true,
		})
		el.HasCheckedDrift = true

		// comparison succeeded, exit the loop
		break
	}
	if !el.IsWithinDrift {
		r.Results.FoundAnything = true
	}
done:
	stats.ExecTime = time.Now().Sub(t1).String()
	out = r.buildResults(el, stats)
	return
}

// GetNetworkTime queries a given NTP server to obtain the network time
func GetNetworkTime(host string) (t time.Time, latency string, err error) {
	raddr, err := net.ResolveUDPAddr("udp", host+":123")
	if err != nil {
		return
	}
	// NTP request is 48 bytes long, we only set the first byte
	data := make([]byte, 48)
	// Flags: 0x1b (27)
	// 00...... leap indicator (0)
	// ..011... version number (3)
	// .....011 mode: client (3)
	data[0] = 3<<3 | 3

	t1 := time.Now()
	con, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return
	}
	defer con.Close()
	// send the request
	_, err = con.Write(data)
	if err != nil {
		return
	}
	// wait up to 5 seconds for the response
	con.SetDeadline(time.Now().Add(5 * time.Second))
	// read up to 48 bytes from the response
	_, err = con.Read(data)
	if err != nil {
		return
	}
	latency = time.Now().Sub(t1).String()
	// Response format (from the RFC)
	//  0                   1                   2                   3
	//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |LI | VN  |Mode |    Stratum     |     Poll      |  Precision   |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |                         Root Delay                            |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |                         Root Dispersion                       |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |                          Reference ID                         |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |                                                               |
	// +                     Reference Timestamp (64)                  +
	// |                                                               |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |                                                               |
	// +                      Origin Timestamp (64)                    +
	// |                                                               |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |                                                               |
	// +                      Receive Timestamp (64)                   +
	// |                                                               |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |                                                               |
	// +                      Transmit Timestamp (64)                  +
	// |                                                               |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	var sec, frac uint64
	sec = uint64(data[43]) | uint64(data[42])<<8 | uint64(data[41])<<16 | uint64(data[40])<<24
	frac = uint64(data[47]) | uint64(data[46])<<8 | uint64(data[45])<<16 | uint64(data[44])<<24
	if sec == 0 || frac == 0 {
		err = fmt.Errorf("null response received from NTP host")
		return
	}
	nsec := sec * 1e9
	nsec += (frac * 1e9) >> 32

	t = time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(nsec)).Local()

	return
}

// buildResults marshals the results
func (r *run) buildResults(el elements, stats statistics) string {
	r.Results.Elements = el
	r.Results.Statistics = stats
	if len(r.Results.Errors) == 0 {
		r.Results.Success = true
	}
	// if was supposed to check drift but hasn't, set success to false
	if r.Parameters.Drift != "" && !el.HasCheckedDrift {
		r.Results.Success = false
	}
	jsonOutput, err := json.Marshal(r.Results)
	if err != nil {
		panic(err)
	}
	return string(jsonOutput[:])
}

func (r *run) PrintResults(result modules.Result, foundOnly bool) (prints []string, err error) {
	var (
		el    elements
		stats statistics
	)
	err = result.GetElements(&el)
	if err != nil {
		return
	}
	prints = append(prints, "local time is "+el.LocalTime)
	if el.HasCheckedDrift {
		if el.IsWithinDrift {
			prints = append(prints, "local time is within acceptable drift from NTP servers")
		} else {
			prints = append(prints, "local time is out of sync from NTP servers")
			for _, drift := range el.Drifts {
				prints = append(prints, drift)
			}
		}
	}
	// stop here if foundOnly is set, we don't want to see errors and stats
	if foundOnly {
		return
	}
	for _, e := range result.Errors {
		prints = append(prints, "error:", e)
	}
	err = result.GetStatistics(&stats)
	if err != nil {
		panic(err)
	}
	prints = append(prints, "stat: execution time was "+stats.ExecTime)
	for _, ntpstat := range stats.NtpStats {
		if ntpstat.Reachable {
			prints = append(prints, "stat: "+ntpstat.Host+" responded in "+ntpstat.Latency+" with time "+ntpstat.Time.UTC().String()+". local time drifts by "+ntpstat.Drift)
		} else {
			prints = append(prints, "stat: "+ntpstat.Host+" was unreachable")
		}
	}
	if result.Success {
		prints = append(prints, fmt.Sprintf("timedrift module has succeeded"))
	} else {
		prints = append(prints, fmt.Sprintf("timedrift module has failed"))
	}
	return
}

func printHelp(isCmd bool) {
	dash := ""
	if isCmd {
		dash = "-"
	}
	fmt.Printf(`timedrift returns the local time of a system and, when %sdrift is set,
verifies that local time is within acceptable range of network time by querying NTP servers

%sdrift <duration>	allowed time drift window. a value of "5s" compares local
			time with ntp hosts and returns a drift failure if local
			time is too far out of sync.

If no drift is set, the module only returns local time.
`, dash, dash)
}

func (r *run) ParamsCreator() (interface{}, error) {
	fmt.Println("initializing timedrift parameters creation")
	var err error
	var p params
	printHelp(false)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("drift> ")
		scanner.Scan()
		if err := scanner.Err(); err != nil {
			fmt.Println("Invalid input. Try again")
			continue
		}
		input := scanner.Text()
		if input == "help" {
			printHelp(false)
			continue
		}
		if input != "" {
			_, err = time.ParseDuration(input)
			if err != nil {
				fmt.Println("invalid drift duration. try again. ex: drift> 5s")
				continue
			}
		}
		p.Drift = input
		break
	}
	r.Parameters = p
	return r.Parameters, r.ValidateParameters()
}

func (r *run) ParamsParser(args []string) (interface{}, error) {
	var (
		err   error
		drift string
		fs    flag.FlagSet
	)
	if len(args) >= 1 && args[0] == "help" {
		printHelp(true)
		return nil, fmt.Errorf("help printed")
	}
	if len(args) == 0 {
		return r.Parameters, nil
	}
	fs.Init("time", flag.ContinueOnError)
	fs.StringVar(&drift, "drift", "", "see help")
	err = fs.Parse(args)
	if err != nil {
		return nil, err
	}
	_, err = time.ParseDuration(drift)
	if err != nil {
		return nil, fmt.Errorf("invalid drift duration. try help.")
	}
	r.Parameters.Drift = drift
	return r.Parameters, r.ValidateParameters()
}
