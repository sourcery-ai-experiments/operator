package env

import (
	"time"

	"github.com/codingconcepts/env"
)

type Env struct {
	ReconcilePeriod          time.Duration `env:"RECONCILE_PERIOD" required:"true"`
	MaxConcurrentReconciles  int           `env:"MAX_CONCURRENT_RECONCILES"`
	DefaultClusterIssuerName string        `env:"DEFAULT_CLUSTER_ISSUER_NAME" required:"true"`

	KloudliteEnvRouteSwitcher string `env:"KLOUDLITE_ENV_ROUTE_SWITCHER" required:"true"`

	AcmeEmail             string `env:"ACME_EMAIL" required:"true"`
	WildcardCertName      string `env:"WILDCARD_CERT_NAME" required:"true"`
	WildcardCertNamespace string `env:"WILDCARD_CERT_NAMESPACE" required:"true"`
}

func GetEnvOrDie() *Env {
	var ev Env
	if err := env.Set(&ev); err != nil {
		panic(err)
	}
	return &ev
}
