package main

import (
	"flag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"operators.kloudlite.io/agent"
	artifactsv1 "operators.kloudlite.io/apis/artifacts/v1"
	crdsv1 "operators.kloudlite.io/apis/crds/v1"
	elasticsearchmsvcv1 "operators.kloudlite.io/apis/elasticsearch.msvc/v1"
	influxdbmsvcv1 "operators.kloudlite.io/apis/influxdb.msvc/v1"
	mongodbCluster "operators.kloudlite.io/apis/mongodb-cluster.msvc/v1"
	mongodbStandalone "operators.kloudlite.io/apis/mongodb-standalone.msvc/v1"
	mysqlclustermsvcv1 "operators.kloudlite.io/apis/mysql-cluster.msvc/v1"
	mysqlstandalonemsvcv1 "operators.kloudlite.io/apis/mysql-standalone.msvc/v1"
	opensearchmsvcv1 "operators.kloudlite.io/apis/opensearch.msvc/v1"
	redisclustermsvcv1 "operators.kloudlite.io/apis/redis-cluster.msvc/v1"
	redisstandalonemsvcv1 "operators.kloudlite.io/apis/redis-standalone.msvc/v1"
	s3awsv1 "operators.kloudlite.io/apis/s3.aws/v1"
	serverlessv1 "operators.kloudlite.io/apis/serverless/v1"
	artifactscontrollers "operators.kloudlite.io/controllers/artifacts"
	"operators.kloudlite.io/env"
	"operators.kloudlite.io/lib/logging"
	"operators.kloudlite.io/lib/redpanda"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(crdsv1.AddToScheme(scheme))
	utilruntime.Must(mongodbStandalone.AddToScheme(scheme))
	utilruntime.Must(mongodbCluster.AddToScheme(scheme))
	utilruntime.Must(mysqlstandalonemsvcv1.AddToScheme(scheme))
	utilruntime.Must(mysqlclustermsvcv1.AddToScheme(scheme))
	utilruntime.Must(redisstandalonemsvcv1.AddToScheme(scheme))
	utilruntime.Must(redisclustermsvcv1.AddToScheme(scheme))
	utilruntime.Must(influxdbmsvcv1.AddToScheme(scheme))
	utilruntime.Must(serverlessv1.AddToScheme(scheme))
	utilruntime.Must(elasticsearchmsvcv1.AddToScheme(scheme))
	utilruntime.Must(opensearchmsvcv1.AddToScheme(scheme))
	utilruntime.Must(s3awsv1.AddToScheme(scheme))
	utilruntime.Must(artifactsv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	// flag.StringVar(&metricsAddr, "metrics-bind-address", ":9091", "The address the metric endpoint binds to.")
	// flag.StringVar(&probeAddr, "health-probe-bind-address", ":9092", "The address the probe endpoint binds to.")
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":12345", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":12346", "The address the probe endpoint binds to.")
	flag.BoolVar(
		&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.",
	)

	var isDev bool
	flag.BoolVar(&isDev, "dev", false, "Enable development mode")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	myLogger := logging.NewOrDie(
		&logging.Options{Name: "operator-logger", Dev: isDev},
	)

	mgr, err := func() (manager.Manager, error) {
		cOpts := ctrl.Options{
			Scheme:                     scheme,
			MetricsBindAddress:         metricsAddr,
			Port:                       9443,
			HealthProbeBindAddress:     probeAddr,
			LeaderElection:             enableLeaderElection,
			LeaderElectionID:           "bf38d2f9.kloudlite.io",
			LeaderElectionResourceLock: "configmaps",
		}
		if isDev {
			return ctrl.NewManager(&rest.Config{Host: "localhost:8080"}, cOpts)
		}
		return ctrl.NewManager(ctrl.GetConfigOrDie(), cOpts)
	}()
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	envVars := env.Must(env.GetEnv())

	if err = (&artifactscontrollers.HarborProjectReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Env:    envVars,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "HarborProject")
		os.Exit(1)
	}
	if err = (&artifactscontrollers.HarborUserAccountReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "HarborUserAccount")
		os.Exit(1)
	}

	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}

	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	consumer, err := redpanda.NewConsumer(envVars.KafkaBrokers, envVars.KafkaConsumerGroupId, envVars.KafkaIncomingTopic)
	if err != nil {
		setupLog.Error(err, "creating redpanda consumer")
		panic(err)
	}
	consumer.SetupLogger(logging.NewZapLogger(types.NamespacedName{}))
	defer consumer.Close()

	go agent.Run(consumer, myLogger)

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		panic(err)
	}
}
