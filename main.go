/*
Copyright 2026 Oscar Romeu

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
	"os"

	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	imagev1beta2 "github.com/fluxcd/image-reflector-controller/api/v1beta2"
	"github.com/fluxcd/pkg/runtime/logger"
	"github.com/oscar-romeu/imagerepo-mirror/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(imagev1beta2.AddToScheme(scheme))

	// +kubebuilder:scaffold:scheme
}

func main() {
	var (
		metricsAddr          string
		enableLeaderElection bool
		destinationRegistry  string
		workers              int
		tagWorkers           int
		logOptions           logger.Options
	)

	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&destinationRegistry, "destination-registry", "",
		"Destination registry prefix to mirror images to (e.g. europe-west4-docker.pkg.dev/my-project/my-repo).")
	flag.IntVar(&workers, "workers", 4, "Number of ImageRepository reconciles to run concurrently.")
	flag.IntVar(&tagWorkers, "tag-workers", 4, "Number of tag copies to run concurrently per reconcile.")
	logOptions.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(logger.NewLogger(logOptions))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:           scheme,
		Metrics:          metricsserver.Options{BindAddress: metricsAddr},
		LeaderElection:   enableLeaderElection,
		LeaderElectionID: "imagerepo-mirror.oscar-romeu.io",
		Logger:           ctrl.Log,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.ImageRepositoryWatcher{
		Client:              mgr.GetClient(),
		DestinationRegistry: destinationRegistry,
		Workers:             workers,
		TagWorkers:          tagWorkers,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ImageRepositoryWatcher")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
