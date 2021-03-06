package lint

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/flant/logboek"
	"github.com/flant/shluz"

	"github.com/flant/werf/cmd/werf/common"
	"github.com/flant/werf/pkg/deploy"
	"github.com/flant/werf/pkg/deploy/helm"
	"github.com/flant/werf/pkg/docker"
	"github.com/flant/werf/pkg/images_manager"
	"github.com/flant/werf/pkg/tag_strategy"
	"github.com/flant/werf/pkg/true_git"
	"github.com/flant/werf/pkg/werf"
)

var commonCmdData common.CmdData

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "lint",
		Short:                 "Run lint procedure for the werf chart",
		DisableFlagsInUseLine: true,
		Annotations: map[string]string{
			common.CmdEnvAnno: common.EnvsDescription(common.WerfSecretKey),
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := common.ProcessLogOptions(&commonCmdData); err != nil {
				common.PrintHelp(cmd)
				return err
			}

			return runLint()
		},
	}

	common.SetupDir(&commonCmdData, cmd)
	common.SetupTmpDir(&commonCmdData, cmd)
	common.SetupHomeDir(&commonCmdData, cmd)

	common.SetupEnvironment(&commonCmdData, cmd)
	common.SetupDockerConfig(&commonCmdData, cmd, "")

	common.SetupSet(&commonCmdData, cmd)
	common.SetupSetString(&commonCmdData, cmd)
	common.SetupValues(&commonCmdData, cmd)
	common.SetupSecretValues(&commonCmdData, cmd)
	common.SetupIgnoreSecretKey(&commonCmdData, cmd)

	common.SetupLogOptions(&commonCmdData, cmd)

	return cmd
}

func runLint() error {
	if err := werf.Init(*commonCmdData.TmpDir, *commonCmdData.HomeDir); err != nil {
		return fmt.Errorf("initialization error: %s", err)
	}

	if err := shluz.Init(filepath.Join(werf.GetServiceDir(), "locks")); err != nil {
		return err
	}

	if err := true_git.Init(true_git.Options{Out: logboek.GetOutStream(), Err: logboek.GetErrStream(), LiveGitOutput: *commonCmdData.LogVerbose || *commonCmdData.LogDebug}); err != nil {
		return err
	}

	if err := deploy.Init(deploy.InitOptions{HelmInitOptions: helm.InitOptions{WithoutKube: true}}); err != nil {
		return err
	}

	if err := docker.Init(*commonCmdData.DockerConfig, *commonCmdData.LogVerbose, *commonCmdData.LogDebug); err != nil {
		return err
	}

	projectDir, err := common.GetProjectDir(&commonCmdData)
	if err != nil {
		return fmt.Errorf("getting project dir failed: %s", err)
	}

	werfConfig, err := common.GetRequiredWerfConfig(projectDir, true)
	if err != nil {
		return fmt.Errorf("unable to load werf config: %s", err)
	}

	imagesRepoManager, err := common.GetImagesRepoManager("REPO", common.MultirepoImagesRepoMode)
	if err != nil {
		return err
	}

	// TODO: optionally use tags by signatures using conveyor
	tag := "TAG"
	tagStrategy := tag_strategy.Custom
	var imagesInfoGetters []images_manager.ImageInfoGetter
	var imagesNames []string
	for _, imageConfig := range werfConfig.StapelImages {
		imagesNames = append(imagesNames, imageConfig.Name)
	}
	for _, imageConfig := range werfConfig.ImagesFromDockerfile {
		imagesNames = append(imagesNames, imageConfig.Name)
	}
	for _, imageName := range imagesNames {
		d := &images_manager.ImageInfo{Name: imageName, WithoutRegistry: true, ImagesRepoManager: imagesRepoManager, Tag: tag}
		imagesInfoGetters = append(imagesInfoGetters, d)
	}

	return deploy.RunLint(projectDir, werfConfig, imagesRepoManager, imagesInfoGetters, tag, tagStrategy, deploy.LintOptions{
		Values:          *commonCmdData.Values,
		SecretValues:    *commonCmdData.SecretValues,
		Set:             *commonCmdData.Set,
		SetString:       *commonCmdData.SetString,
		Env:             *commonCmdData.Environment,
		IgnoreSecretKey: *commonCmdData.IgnoreSecretKey,
	})
}
