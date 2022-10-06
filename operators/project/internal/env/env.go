package env

import (
	"time"

	"github.com/codingconcepts/env"
)

type Env struct {
	ReconcilePeriod         time.Duration `env:"RECONCILE_PERIOD"`
	MaxConcurrentReconciles int           `env:"MAX_CONCURRENT_RECONCILES"`

	ProjectCfgName    string `env:"PROJECT_CONFIGMAP_NAME" required:"true"`
	DockerSecretName  string `env:"DOCKER_SECRET_NAME" required:"true"`
	AdminRoleName     string `env:"ADMIN_ROLE_NAME" required:"true"`
	SvcAccountName    string `env:"SVC_ACCOUNT_NAME" required:"true"`
	AccountRouterName string `env:"ACCOUNT_ROUTER_NAME" required:"true"`
}

func GetEnvOrDie() *Env {
	var ev Env
	if err := env.Set(&ev); err != nil {
		panic(err)
	}
	return &ev
}
