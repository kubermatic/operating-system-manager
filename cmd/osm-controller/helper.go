/*
Copyright 2022 The Operating System Manager contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"net"
	"os"
	"path"
	"strings"

	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func createManager(opt *options) (manager.Manager, error) {
	// Manager options
	options := manager.Options{
		LeaderElection:          opt.enableLeaderElection,
		LeaderElectionID:        "operating-system-manager",
		LeaderElectionNamespace: opt.namespace,
		HealthProbeBindAddress:  opt.healthProbeAddress,
		MetricsBindAddress:      opt.metricsAddress,
		Port:                    9443,
		Namespace:               opt.namespace,
	}

	mgr, err := manager.New(config.GetConfigOrDie(), options)
	if err != nil {
		return nil, fmt.Errorf("error building ctrlruntime manager: %v", err)
	}

	// Add health endpoints
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return nil, fmt.Errorf("failed to add health check: %v", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return nil, fmt.Errorf("failed to add readiness check: %v", err)
	}
	return mgr, nil
}

func parseClusterDNSIPs(s string) ([]net.IP, error) {
	var ips []net.IP
	sips := strings.Split(s, ",")
	for _, sip := range sips {
		ip := net.ParseIP(strings.TrimSpace(sip))
		if ip == nil {
			return nil, fmt.Errorf("unable to parse ip %s", sip)
		}
		ips = append(ips, ip)
	}
	return ips, nil
}

// getKubeConfigPath returns the path to the kubeconfig file.
func getKubeConfigPath() string {
	if os.Getenv("KUBECONFIG") != "" {
		return os.Getenv("KUBECONFIG")
	}
	return path.Join(homedir.HomeDir(), ".kube/config")
}
