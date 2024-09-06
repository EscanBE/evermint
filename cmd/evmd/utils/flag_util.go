package utils

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// UpdateRegisteredGasAdjustmentFlags takes input is command, walking recursive sub-commands
// and try to update all gas adjustment flags with minimumValue if it is less than minimumValue.
func UpdateRegisteredGasAdjustmentFlags(cmd *cobra.Command, minimumValue float64) {
	flagModifier := func(flag *pflag.Flag) {
		if flag == nil {
			return
		}

		if flag.Name == flags.FlagGasAdjustment {
			tryUpdateGasAdjustmentFlag(flag, minimumValue)
		}
	}

	cmd.PersistentFlags().VisitAll(flagModifier)

	cmd.Flags().VisitAll(flagModifier)

	subCommands := cmd.Commands()
	if len(subCommands) > 0 {
		for _, subCommand := range subCommands {
			UpdateRegisteredGasAdjustmentFlags(subCommand, minimumValue)
		}
	}
}

// tryUpdateGasAdjustmentFlag will update the flag value if it is less than minimumValue.
func tryUpdateGasAdjustmentFlag(flag *pflag.Flag, minimumValue float64) {
	if flag.Name != flags.FlagGasAdjustment {
		panic(fmt.Sprintf("not --%s flag", flags.FlagGasAdjustment))
	}

	value := flag.Value
	if value == nil {
		return // un-expected, ignore
	}

	currentGasAdjustment, err := strconv.ParseFloat(value.String(), 64)
	if err != nil {
		return // ignore
	}

	if currentGasAdjustment >= minimumValue {
		return
	}

	minimumStrValue := strconv.FormatFloat(minimumValue, 'g', -1, 64) // use the same conversion as lib
	err = value.Set(minimumStrValue)
	if err != nil {
		// ignore error
		return
	}

	// updated
	flag.DefValue = minimumStrValue
}
