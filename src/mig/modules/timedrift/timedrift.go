// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package timedrift

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"mig"
	"net"
	"os"
	"time"
)

// init is called by the Go runtime at startup. We use this function to
// register the module in a global array of available modules, so the
// agent knows we exist
func init() {
	mig.RegisterModule("timedrift", func() interface{} {
		return new(Runner)
	})
}

// Runner gives access to the exported functions and structs of the module
type Runner struct {
	Parameters params
	Results    results
}

type results struct {
	FoundAnything bool        `json:"foundanything"`
	Success       bool        `json:"success"`
	Elements      checkedtime `json:"elements"`
	Statistics    statistics  `json:"statistics"`
	Errors        []string    `json:"errors"`
}

// a simple parameters structure, the format is arbitrary
type params struct {
	Drift string `json:"drift"`
}

type checkedtime struct {
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

func (r Runner) ValidateParameters() (err error) {
	if r.Parameters.Drift != "" {
		_, err = time.ParseDuration(r.Parameters.Drift)
	}
	return err
}

func (r Runner) Run(Args []byte) string {
	var (
		stats statistics
		ct    checkedtime
		drift time.Duration
	)
	ct.LocalTime = time.Now().Format(time.RFC3339Nano)
	t1 := time.Now()
	err := json.Unmarshal(Args, &r.Parameters)
	if err != nil {
		r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		return r.buildResults()
	}
	err = r.ValidateParameters()
	if err != nil {
		r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		return r.buildResults()
	}
	// if drift is not set, skip the ntp test
	if r.Parameters.Drift == "" {
		r.Results.FoundAnything = true
		goto done
	}
	drift, err = time.ParseDuration(r.Parameters.Drift)
	if err != nil {
		r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		return r.buildResults()
	}
	// assume host has synched time and set to false if not true
	ct.IsWithinDrift = true
	for _, host := range NtpPool {
		t, lat, err := GetTimeFromHost(host)
		localtime := time.Now()
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
			continue
		}
		if localtime.Before(t.Add(-drift)) {
			ct.IsWithinDrift = false
			ct.Drifts = append(ct.Drifts, fmt.Sprintf("Local time is behind ntp host %s by %s", host, t.Sub(localtime).String()))
		} else if localtime.After(t.Add(drift)) {
			ct.IsWithinDrift = false
			ct.Drifts = append(ct.Drifts, fmt.Sprintf("Local time is ahead of ntp host %s by %s", host, localtime.Sub(t).String()))
		}
		stats.NtpStats = append(stats.NtpStats, ntpstats{
			Host:      host,
			Time:      t,
			Latency:   lat,
			Drift:     localtime.Sub(t).String(),
			Reachable: true,
		})
		r.Results.FoundAnything = true
		ct.HasCheckedDrift = true
	}
done:
	r.Results.Elements = ct
	stats.ExecTime = time.Now().Sub(t1).String()
	r.Results.Statistics = stats
	return r.buildResults()
}

var NtpPool = [...]string{`0.pool.ntp.org`, `1.pool.ntp.org`, `2.pool.ntp.org`, `3.pool.ntp.org`}

func GetTimeFromHost(host string) (t time.Time, lat string, err error) {
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
	lat = time.Now().Sub(t1).String()
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

	nsec := sec * 1e9
	nsec += (frac * 1e9) >> 32

	t = time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(nsec)).Local()

	return
}

// buildResults marshals the results
func (r Runner) buildResults() string {
	if len(r.Results.Errors) == 0 {
		r.Results.Success = true
	}
	jsonOutput, err := json.Marshal(r.Results)
	if err != nil {
		panic(err)
	}
	return string(jsonOutput[:])
}

func (r Runner) PrintResults(rawResults []byte, foundOnly bool) (prints []string, err error) {
	var results results
	err = json.Unmarshal(rawResults, &results)
	if err != nil {
		return
	}
	prints = append(prints, "local time is "+results.Elements.LocalTime)
	if results.Elements.HasCheckedDrift {
		if results.Elements.IsWithinDrift {
			prints = append(prints, "local time is within acceptable drift from NTP servers")
		} else {
			prints = append(prints, "local time is out of sync from NTP servers")
			for _, drift := range results.Elements.Drifts {
				prints = append(prints, drift)
			}
		}
	}
	// stop here if foundOnly is set, we don't want to see errors and stats
	if foundOnly {
		return
	}
	for _, e := range results.Errors {
		prints = append(prints, "error:", e)
	}
	fmt.Println("stat: execution time", results.Statistics.ExecTime)
	for _, ntpstat := range results.Statistics.NtpStats {
		if ntpstat.Reachable {
			prints = append(prints, "stat: "+ntpstat.Host+" responded in "+ntpstat.Latency+" with time "+ntpstat.Time.UTC().String()+". local time drifts by "+ntpstat.Drift)
		} else {
			prints = append(prints, "stat: "+ntpstat.Host+" was unreachable")
		}
	}
	return
}

func printHelp(isCmd bool) {
	dash := ""
	if isCmd {
		dash = "-"
	}
	fmt.Printf(`timedrift returns the local time of a system, and verifies if it is within
acceptable range of NTP time if %sdrift is set.

%sdrift <duration>	allowed time drift window. a value of "5s" compares local
			time with ntp hosts and returns a drift failure if local
			time is too far out of sync.

If no drift is set, the module only returns local time.
`, dash, dash)
}

func (r Runner) ParamsCreator() (interface{}, error) {
	fmt.Println("initializing netstat parameters creation")
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
	return p, nil
}

func (r Runner) ParamsParser(args []string) (interface{}, error) {
	var (
		err   error
		drift string
		fs    flag.FlagSet
		p     params
	)
	if len(args) == 0 {
		return p, nil
	}
	if len(args) > 1 && args[0] == "help" {
		printHelp(true)
		return nil, nil
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
	p.Drift = drift
	return p, nil
}
