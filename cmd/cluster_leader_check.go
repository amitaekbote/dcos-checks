// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"fmt"
	"net"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var validArgs = map[string]bool{
	"mesos-leader":     true,
	"marathon-leader":  true,
	"metronome-leader": true,
}

// NewClusterLeaderCheck returns an intialized instance of *ClusterLeaderCheck
func NewClusterLeaderCheck(name string, args []string) DCOSChecker {
	return &ClusterLeaderCheck{
		Name: name,
		Args: args,
	}
}

// clusterCmd represents the cluster command
var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Cluster health checks",
	Long: `List of checks to verify cluster is healthy/up
Usage:
cluster mesos-leader
cluster marathon-leader
cluster metronome-leader
`,
	Run: func(cmd *cobra.Command, args []string) {
		RunCheck(context.TODO(), NewClusterLeaderCheck("DC/OS cluster system checks", args))
	},
}

func init() {
	RootCmd.AddCommand(clusterCmd)
}

// ClusterLeaderCheck verifies dc/os is up
type ClusterLeaderCheck struct {
	Name string
	Args []string
}

// ID returns a unique check identifier.
func (cs *ClusterLeaderCheck) ID() string {
	return cs.Name
}

// Run the cluster check
func (cs *ClusterLeaderCheck) Run(ctx context.Context, cfg *CLIConfigFlags) (string, int, error) {
	var args = cs.Args

	var keys []string
	for key := range validArgs {
		keys = append(keys, key)
	}

	if len(args) == 0 {
		return "", statusFailure, fmt.Errorf("No args provided, valid args %v", keys)
	}

	for _, arg := range args {
		switch arg {
		case "mesos-leader":
			return cs.MesosLeader("leader.mesos")
		case "marathon-leader":
			var portM int
			if cfg.ForceTLS {
				portM = adminrouterMasterHTTPSPort
			}
			return cs.MarathonLeader(cfg, URLFields{host: "leader.mesos", path: "/service/marathon/v2/leader", port: portM})
		case "metronome-leader":
			return cs.MetronomeLeader(cfg, URLFields{host: "leader.mesos", path: "/service/metronome/v1/jobs"})
		default:
		}
	}
	return "", statusFailure, fmt.Errorf("Option not supported, valid args %v", keys)
}

// MesosLeader resolves leader.mesos
func (cs *ClusterLeaderCheck) MesosLeader(hostname string) (string, int, error) {
	mesosIP, err := net.LookupHost(hostname)
	if err != nil {
		return "", statusFailure, errors.Wrapf(err, "LookupHost failed")
	}
	if len(mesosIP) == 0 {
		return "", statusFailure, errors.Wrapf(err, "LookupHost did not return any address")
	}
	return "", statusOK, nil
}

// MarathonLeader resolves /service/marathon/v2/leader
func (cs *ClusterLeaderCheck) MarathonLeader(cfg *CLIConfigFlags, urlOpt URLFields) (string, int, error) {
	status, _, err := HTTPRequest(cfg, urlOpt)
	if err != nil {
		return "", statusFailure, errors.Wrapf(err, "HTTP request error")
	}
	if status == 200 {
		return "", statusOK, nil
	}
	if status == 500 {
		return "", statusFailure, errors.New("status 500")
	}
	return "", statusUnknown, errors.New("Unable to run the check")
}

// MetronomeLeader checks results of /service/metronome/v1/jobs
func (cs *ClusterLeaderCheck) MetronomeLeader(cfg *CLIConfigFlags, urlOpt URLFields) (string, int, error) {
	status, _, err := HTTPRequest(cfg, urlOpt)
	if err != nil {
		return "", statusFailure, errors.Wrapf(err, "HTTP request error")
	}

	if status == 200 {
		return "", statusOK, nil
	}
	if status == 500 {
		return "", statusFailure, errors.New("status 500")
	}
	return "", statusUnknown, errors.New("Unable to run the check")
}
