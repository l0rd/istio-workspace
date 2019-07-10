package serve

import (
	"context"
	"fmt"

	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/metrics"

	"github.com/maistra/istio-workspace/pkg/apis"
	"github.com/maistra/istio-workspace/pkg/controller"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/spf13/cobra"
	k8sConfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

var log = logf.Log.WithName("cmd").WithValues("type", "serve")

var (
	metricsHost       = "0.0.0.0"
	metricsPort int32 = 8383
)

// NewCmd creates instance of "ike serve" Cobra Command which is intended to be ran in the
// cluster as it starts istio-workspace operator
func NewCmd() *cobra.Command {
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Starts istio-workspace operator in the cluster",
		RunE:  startOperator,
	}
	return serveCmd
}

func startOperator(cmd *cobra.Command, args []string) error { //nolint[:unparam]
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Error(err, "Failed to get watch namespace")
		return err
	}

	// Get a config to talk to the apiserver
	cfg, err := k8sConfig.GetConfig()
	if err != nil {
		log.Error(err, "")
		return err
	}

	ctx := context.TODO()

	// Become the leader before proceeding
	if e := leader.Become(ctx, "istio-workspace-lock"); e != nil {
		log.Error(e, "")
		return e
	}

	// Create a new Cmd to provide shared dependencies and Start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          namespace,
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	})
	if err != nil {
		log.Error(err, "")
		return err
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err = apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		return nil
	}

	// Setup all Controllers
	if err = controller.AddToManager(mgr); err != nil {
		log.Error(err, "")
		return err
	}

	// Create Service object to expose the metrics port.
	if _, err = metrics.ExposeMetricsPort(ctx, metricsPort); err != nil {
		log.Info(err.Error())
	}

	log.Info("Starting the operator.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		return err
	}

	return nil
}