package deploy

import (
	"github.com/flant/logboek"

	"github.com/flant/werf/pkg/deploy/secret"
	"github.com/flant/werf/pkg/deploy/werf_chart"
)

func PrepareWerfChart(projectName, chartDir, env string, m secret.Manager, secretValues []string, serviceValues map[string]interface{}) (*werf_chart.WerfChart, error) {
	werfChart, err := werf_chart.InitWerfChart(projectName, chartDir, env, m)
	if err != nil {
		return nil, err
	}

	for _, path := range secretValues {
		if err = werfChart.SetSecretValuesFile(path, m); err != nil {
			return nil, err
		}
	}

	if serviceValues != nil {
		if err = werfChart.SetServiceValues(serviceValues); err != nil {
			return nil, err
		}
	}

	logboek.Debug.LogF("werf chart: %#v\n", werfChart)

	return werfChart, nil
}
