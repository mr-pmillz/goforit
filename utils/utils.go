package utils

import (
	"bufio"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// LoadFromCommandOpts ...
type LoadFromCommandOpts struct {
	CommaInStringToSlice bool
	DefaultFlagVal       string
	Flag                 string
	IsFilePath           bool
	Prefix               string // Optional
	Opts                 interface{}
}

// ConfigureFlagOpts sets the cobra flag option to the LoadFromCommandOpts.Opts key
// it returns the parsed value of the cobra flag from LoadFromCommandOpts.Flag
//
//nolint:gocognit
func ConfigureFlagOpts(cmd *cobra.Command, lfcOpts *LoadFromCommandOpts) (interface{}, error) {
	cmdFlag, err := cmd.Flags().GetString(fmt.Sprintf("%s%s", lfcOpts.Prefix, lfcOpts.Flag))
	if err != nil {
		return nil, err
	}

	switch cmdFlag {
	case "":
		flagToUpperConfig := strings.ToUpper(strings.ReplaceAll(fmt.Sprintf("%s%s", lfcOpts.Prefix, lfcOpts.Flag), "-", "_"))
		configVal := viper.GetString(flagToUpperConfig)
		envVal, ok := os.LookupEnv(configVal)
		configSliceVal := viper.GetStringSlice(flagToUpperConfig)
		if ok {
			if lfcOpts.IsFilePath {
				fileExists, err := Exists(envVal)
				if err != nil {
					return nil, err
				}
				if fileExists {
					absVal, err := ResolveAbsPath(envVal)
					if err != nil {
						return nil, err
					}
					lfcOpts.Opts = absVal
				} else {
					lfcOpts.Opts = envVal
				}
			} else {
				lfcOpts.Opts = envVal
			}
		} else {
			switch {
			case len(configSliceVal) > 1 && strings.Contains(configVal, "\n"):
				lfcOpts.Opts = configSliceVal
			case configVal != "":
				if lfcOpts.IsFilePath {
					if exists, err := Exists(configVal); exists && err == nil {
						absConfigVal, err := ResolveAbsPath(configVal)
						if err != nil {
							return nil, err
						}
						lfcOpts.Opts = absConfigVal
					} else {
						lfcOpts.Opts = configVal
					}
				} else {
					lfcOpts.Opts = configVal
				}
			default:
				switch {
				case lfcOpts.DefaultFlagVal != "" && lfcOpts.IsFilePath:
					absDefaultVal, err := ResolveAbsPath(lfcOpts.DefaultFlagVal)
					if err != nil {
						return nil, err
					}
					_, err = os.Stat(absDefaultVal)
					if os.IsNotExist(err) {
						lfcOpts.Opts = cmdFlag
					} else {
						lfcOpts.Opts = absDefaultVal
					}
				case lfcOpts.DefaultFlagVal != "" && !lfcOpts.IsFilePath:
					lfcOpts.Opts = lfcOpts.DefaultFlagVal
				default:
					lfcOpts.Opts = cmdFlag
				}
			}
		}
	default:
		envValue, ok := os.LookupEnv(strings.ToUpper(strings.ReplaceAll(fmt.Sprintf("%s%s", lfcOpts.Prefix, lfcOpts.Flag), "-", "_")))
		if ok {
			if strings.Contains(envValue, ",") && lfcOpts.CommaInStringToSlice {
				val := strings.Split(envValue, ",")
				lfcOpts.Opts = val
				return lfcOpts.Opts, nil
			}
			lfcOpts.Opts = envValue
		} else {
			if lfcOpts.IsFilePath {
				fileExists, err := Exists(cmdFlag)
				if err != nil {
					return nil, err
				}
				if fileExists {
					absCmdFlag, err := ResolveAbsPath(cmdFlag)
					if err != nil {
						return nil, err
					}
					lfcOpts.Opts = absCmdFlag
				} else {
					if strings.Contains(cmdFlag, ",") && lfcOpts.CommaInStringToSlice {
						val := strings.Split(cmdFlag, ",")
						lfcOpts.Opts = val
						return lfcOpts.Opts, nil
					}
					lfcOpts.Opts = cmdFlag
				}
			} else {
				if strings.Contains(cmdFlag, ",") && lfcOpts.CommaInStringToSlice {
					val := strings.Split(cmdFlag, ",")
					lfcOpts.Opts = val
					return lfcOpts.Opts, nil
				}
				lfcOpts.Opts = cmdFlag
			}
		}
	}
	return lfcOpts.Opts, nil
}

// ResolveAbsPath ...
func ResolveAbsPath(path string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return path, err
	}

	dir := usr.HomeDir
	if path == "~" {
		path = dir
	} else if strings.HasPrefix(path, "~/") {
		path = filepath.Join(dir, path[2:])
	}

	path, err = filepath.Abs(path)
	if err != nil {
		return path, err
	}

	return path, nil
}

// Exists returns whether the given file or directory exists
func Exists(path string) (bool, error) {
	if path == "" {
		return false, nil
	}
	absPath, err := ResolveAbsPath(path)
	if err != nil {
		return false, err
	}
	info, err := os.Stat(absPath)
	if err == nil {
		switch {
		case info.IsDir():
			return true, nil
		case info.Size() != 0:
			return true, nil
		case info.Size() == 0:
			// file exists but it's empty
			return false, nil
		}
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// ReadLines reads a whole file into memory
// and returns a slice of its lines.
func ReadLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}
