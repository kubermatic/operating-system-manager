/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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
	"context"
	"flag"
	"os"

	"go.uber.org/zap"

	"k8c.io/operating-system-manager/pkg/controllers/osp"
	"k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

type options struct {
	workerCount int
	namespace   string
}

func main() {
	klog.InitFlags(nil)

	opt := &options{}
	flag.IntVar(&opt.workerCount, "worker-count", 10, "Number of workers which process reconciliation in parallel.")
	flag.StringVar(&opt.namespace, "namespace", "", "The namespace where the OSP controller will run.")

	if len(opt.namespace) == 0 {
		klog.Fatal("-namespace is required")
	}

	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{})
	if err != nil {
		klog.Error(err, "could not create manager")
		os.Exit(1)
	}
	if err = v1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		klog.Fatal(err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		klog.Fatal(err)
	}

	log := logger.Sugar()
	ctx := context.Background()

	if err := osp.Add(ctx, log, mgr, opt.namespace, opt.workerCount); err != nil {
		klog.Fatal(err)
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		klog.Fatalf("Failed to start OSP controller: %v", zap.Error(err))
	}
}
