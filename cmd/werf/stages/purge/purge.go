package purge

import (
	"fmt"
	"path/filepath"

	"github.com/flant/shluz"

	"github.com/spf13/cobra"

	"github.com/flant/logboek"
	"github.com/flant/werf/cmd/werf/common"
	"github.com/flant/werf/pkg/cleaning"
	"github.com/flant/werf/pkg/docker"
	"github.com/flant/werf/pkg/docker_registry"
	"github.com/flant/werf/pkg/werf"
)

var CmdData struct {
	Force bool
}

var CommonCmdData common.CmdData

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "purge",
		DisableFlagsInUseLine: true,
		Short:                 "Purge project stages from stages storage",
		Long:                  common.GetLongCommandDescription("Purge project stages from stages storage"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := common.ProcessLogOptions(&CommonCmdData); err != nil {
				common.PrintHelp(cmd)
				return err
			}

			common.LogVersion()

			return common.LogRunningTime(func() error {
				return runPurge()
			})
		},
	}

	common.SetupDir(&CommonCmdData, cmd)
	common.SetupTmpDir(&CommonCmdData, cmd)
	common.SetupHomeDir(&CommonCmdData, cmd)

	common.SetupStagesStorage(&CommonCmdData, cmd)
	common.SetupStagesStorageLock(&CommonCmdData, cmd)
	common.SetupDockerConfig(&CommonCmdData, cmd, "Command needs granted permissions to read, pull and delete images from the specified stages storage")
	common.SetupInsecureRegistry(&CommonCmdData, cmd)
	common.SetupSkipTlsVerifyRegistry(&CommonCmdData, cmd)

	common.SetupLogOptions(&CommonCmdData, cmd)
	common.SetupLogProjectDir(&CommonCmdData, cmd)

	common.SetupDryRun(&CommonCmdData, cmd)
	cmd.Flags().BoolVarP(&CmdData.Force, "force", "", false, common.CleaningCommandsForceOptionDescription)

	return cmd
}

func runPurge() error {
	if err := werf.Init(*CommonCmdData.TmpDir, *CommonCmdData.HomeDir); err != nil {
		return fmt.Errorf("initialization error: %s", err)
	}

	if err := shluz.Init(filepath.Join(werf.GetServiceDir(), "locks")); err != nil {
		return err
	}

	if err := docker_registry.Init(docker_registry.Options{InsecureRegistry: *CommonCmdData.InsecureRegistry, SkipTlsVerifyRegistry: *CommonCmdData.SkipTlsVerifyRegistry}); err != nil {
		return err
	}

	if err := docker.Init(*CommonCmdData.DockerConfig, *CommonCmdData.LogVerbose, *CommonCmdData.LogDebug); err != nil {
		return err
	}

	projectDir, err := common.GetProjectDir(&CommonCmdData)
	if err != nil {
		return fmt.Errorf("getting project dir failed: %s", err)
	}

	common.ProcessLogProjectDir(&CommonCmdData, projectDir)

	werfConfig, err := common.GetWerfConfig(projectDir)
	if err != nil {
		return fmt.Errorf("bad config: %s", err)
	}

	projectName := werfConfig.Meta.Project

	_, err = common.GetStagesStorage(&CommonCmdData)
	if err != nil {
		return err
	}

	_, err = common.GetStagesStorageLock(&CommonCmdData)
	if err != nil {
		return err
	}

	stagesPurgeOptions := cleaning.StagesPurgeOptions{
		ProjectName:                   projectName,
		DryRun:                        *CommonCmdData.DryRun,
		RmContainersThatUseWerfImages: CmdData.Force,
	}

	logboek.LogOptionalLn()
	if err := cleaning.StagesPurge(stagesPurgeOptions); err != nil {
		return err
	}

	return nil
}
