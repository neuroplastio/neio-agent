package agentcli

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/neuroplastio/neio-agent/pkg/agent"
	"github.com/spf13/cobra"
)

func NewAgentCmd(configDir string) *cobra.Command {
	cfg := agent.Config{
		DataDir:    filepath.Join(configDir, "data"),
		FlowConfig: filepath.Join(configDir, "flow.yml"),
		UhidConfig: filepath.Join(configDir, "uhid.yml"),
	}
	agentCmd := &cobra.Command{
		Use:   "neio-agent",
		Short: "Neuroplast.io Agent",
		Long:  `The Neuroplast.io Agent is a daemon that runs the core logic of the Neuroplast.io project.`,
	}
	agentCmd.PersistentFlags().StringVar(&cfg.DataDir, "data-dir", cfg.DataDir, "data directory")
	agentCmd.PersistentFlags().StringVar(&cfg.FlowConfig, "flow-config", cfg.FlowConfig, "flow config file")
	agentCmd.PersistentFlags().StringVar(&cfg.UhidConfig, "uhid-config", cfg.UhidConfig, "uhid config file")
	agentCmd.AddCommand(NewRunCmd(&cfg))
	return agentCmd
}

func NewRunCmd(cfg *agent.Config) *cobra.Command {
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run the Neuroplast.io Agent",
		Long:  `The Neuroplast.io Agent is a daemon that runs the core logic of the Neuroplast.io project.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return agent.NewAgent(*cfg).Run(cmd.Context())
		},
	}
	return runCmd
}

func Main(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) error {
	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	cmd := NewAgentCmd(filepath.Join(dir, "neio"))
	cmd.SetArgs(args)
	cmd.SetIn(in)
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	return cmd.ExecuteContext(ctx)
}
