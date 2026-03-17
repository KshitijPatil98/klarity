package main

import (
	"flag"
	"os"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	log := ctrl.Log.WithName("main")

	cfg := ctrl.GetConfigOrDie()
	log.Info("connected to cluster", "apiServer", cfg.Host)

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{})
	if err != nil {
		log.Error(err, "unable to create manager")
		os.Exit(1)
	}

	log.Info("starting manager")

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "problem running manager")
		os.Exit(1)
	}
}
