package agentcli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/neuroplastio/neio-agent/hidapi/hiddesc"
	"github.com/neuroplastio/neio-agent/internal/hidsvc"
	"github.com/neuroplastio/neio-agent/pkg/agent"
	"github.com/spf13/cobra"
)

func Main(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) error {
	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	cmd := NewRootCmd(filepath.Join(dir, "neio"))
	cmd.SetArgs(args)
	cmd.SetIn(in)
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	return cmd.ExecuteContext(ctx)
}

type agentProvider func() *agent.Agent

func NewRootCmd(configDir string) *cobra.Command {
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
	var a *agent.Agent
	agentProvider := func() *agent.Agent {
		return a
	}
	agentCmd.PersistentFlags().StringVar(&cfg.DataDir, "data-dir", cfg.DataDir, "data directory")
	agentCmd.PersistentFlags().StringVar(&cfg.FlowConfig, "flow-config", cfg.FlowConfig, "flow config file")
	agentCmd.PersistentFlags().StringVar(&cfg.UhidConfig, "uhid-config", cfg.UhidConfig, "uhid config file")
	agentCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		var err error
		a, err = agent.NewAgent(cfg)
		return err
	}
	agentCmd.AddCommand(NewRun(agentProvider))
	agentCmd.AddCommand(NewListDevices(agentProvider))
	agentCmd.AddCommand(NewGetReportDescriptor(agentProvider))
	return agentCmd
}

func NewRun(agent agentProvider) *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run the Neuroplast.io Agent",
		Long:  `The Neuroplast.io Agent is a daemon that runs the core logic of the Neuroplast.io project.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return agent().Run(cmd.Context())
		},
	}
}

func NewListDevices(agent agentProvider) *cobra.Command {
	return &cobra.Command{
		Use:   "list-devices",
		Short: "List HID devices",
		Long:  `List HID devices connected to the system.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			devices, err := agent().HID().ListInputDevices()
			if err != nil {
				return err
			}
			jsonB, err := json.MarshalIndent(devices, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(jsonB))
			return nil
		},
	}
}

func NewGetReportDescriptor(agent agentProvider) *cobra.Command {
	var raw bool
	cmd := &cobra.Command{
		Use:   "get-report-descriptor",
		Short: "Get report descriptor",
		Long:  `Get report descriptor of a HID device.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("usage: get-report-descriptor <addr>")
			}
			addr, err := hidsvc.ParseAddress(args[0])
			if err != nil {
				return err
			}
			dev, err := agent().HID().GetInputDevice(addr)
			if err != nil {
				return err
			}
			if raw {
				cmd.OutOrStdout().Write(dev.BackendDevice.ReportDescriptor)
				return nil
			}
			desc, err := hiddesc.Decode(dev.BackendDevice.ReportDescriptor)
			if err != nil {
				return err
			}
			jsonB, err := json.MarshalIndent(desc, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(jsonB))
			return nil
		},
	}
	cmd.Flags().BoolVar(&raw, "raw", false, "print raw report descriptor")
	return cmd
}
