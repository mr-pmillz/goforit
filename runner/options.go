package runner

import (
	"github.com/mr-pmillz/goforit/utils"
	"github.com/spf13/cobra"
	"reflect"
)

type Options struct {
	Target     interface{}
	Verbose    bool
	Output     string
	StreamNmap bool
}

// ConfigureCommand ...
func ConfigureCommand(cmd *cobra.Command) error {
	cmd.PersistentFlags().StringP("target", "t", "", "target to scan")
	cmd.PersistentFlags().BoolP("verbose", "v", false, "toggle verbosity")
	cmd.PersistentFlags().BoolP("stream-nmap", "", false, "run nmap and stream results in real time")
	cmd.PersistentFlags().StringP("output", "o", "", "directory to store all generated output")
	return nil
}

// LoadFromCommand ...
func (opts *Options) LoadFromCommand(cmd *cobra.Command) error {
	// ConfigureCommand has already run by the time LoadFromCommand gets called.
	target, err := utils.ConfigureFlagOpts(cmd, &utils.LoadFromCommandOpts{
		Flag:                 "target",
		IsFilePath:           false,
		Opts:                 opts.Target,
		CommaInStringToSlice: true,
	})
	if err != nil {
		return err
	}
	rt := reflect.TypeOf(target)
	switch rt.Kind() {
	case reflect.Slice:
		opts.Target = target.([]string)
	case reflect.String:
		opts.Target = target.(string)
	}

	verbose, err := cmd.Flags().GetBool("verbose")
	if err != nil {
		return err
	}
	opts.Verbose = verbose

	nmapStream, err := cmd.Flags().GetBool("stream-nmap")
	if err != nil {
		return err
	}
	opts.StreamNmap = nmapStream

	output, err := utils.ConfigureFlagOpts(cmd, &utils.LoadFromCommandOpts{
		Flag:       "output",
		IsFilePath: true,
		Opts:       opts.Output,
	})
	if err != nil {
		return err
	}
	opts.Output = output.(string)

	return nil
}
