package config

import (
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spf13/viper"
	"strings"
)

const (
	configKey  = "post"
	envPrefix  = "SM_POST"
	configPath = "config-path"
)

type postConfig struct {
	DataFolder      string `mapstructure:"data-folder"`
	LogEveryXLabels uint64 `mapstructure:"log-every-x-labels"`
}

var Post = postConfig{
	// Default config values (overwritten in init() if config file detected):
	DataFolder:      "~/.spacemesh-data/post-data",
	LogEveryXLabels: 5000000,
}

func init() {
	// TODO @noam read flags: https://github.com/spf13/viper#working-with-flags

	viper.SetEnvPrefix(envPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	err := viper.BindEnv(configPath)
	viper.AutomaticEnv()

	viper.SetConfigName("config")
	viper.AddConfigPath(viper.GetString(configPath))
	// TODO @noam finalize directories to search for config
	// The following makes sense in a *nix env when running a production binary:
	viper.AddConfigPath("/etc/spacemesh/")  // path to look for the config file in
	viper.AddConfigPath("$HOME/.spacemesh") // call multiple times to add many search paths
	viper.AddConfigPath(".")                // optionally look for config in the working directory
	// Need to see what makes sense on Windows.
	err = viper.ReadInConfig()
	if err != nil {
		log.Warning("failed to load config, falling back to defaults: %v", err)
	} else {
		err = viper.UnmarshalKey(configKey, &Post)
		if err != nil {
			log.Warning("failed to load config, falling back to defaults: %v", err)
		}
	}
}
