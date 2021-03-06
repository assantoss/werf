package deploy

import (
	"github.com/flant/logboek"
	"github.com/flant/werf/pkg/deploy/helm"
)

type DismissOptions struct {
	WithNamespace bool
	WithHooks     bool
}

func RunDismiss(releaseName, namespace, _ string, opts DismissOptions) error {
	logboek.Debug.LogF("Dismiss options: %#v\n", opts)
	logboek.Debug.LogF("Namespace: %s\n", namespace)
	return helm.PurgeHelmRelease(releaseName, namespace, opts.WithNamespace, opts.WithHooks)
}
