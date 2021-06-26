package server

// NOTE: PoST RPC server is currently disabled.

/*
import (
	"fmt"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/smutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"path/filepath"
	"reflect"
)

const (
	defaultConfigFileName = "config.toml"
	defaultDataDirName    = "data"
	defaultLogDebug       = false
)

var (
	defaultHomeDir      = filepath.Join(smutil.GetUserHomeDirectory(), "post")
	defaultDataDir      = filepath.Join(defaultHomeDir, defaultDataDirName)
	defaultConfigFile   = filepath.Join(defaultHomeDir, defaultConfigFileName)
	defaultLogDir       = defaultHomeDir
	defaultRPCListener  = "localhost:50001"
	defaultRESTListener = "localhost:8080"
)

type Config struct {
	ServerCfg *ServerConfig  `mapstructure:"server"`
	PostCfg   *config.Config `mapstructure:"post"`
}

func defaultConfig() *Config {
	return &Config{
		ServerCfg: defaultServerConfig(),
		PostCfg:   config.DefaultConfig(),
	}
}

type ServerConfig struct {
	HomeDir      string `mapstructure:"homedir"`
	ConfigFile   string `mapstructure:"config"`
	LogDir       string `mapstructure:"logdir"`
	LogDebug     bool   `mapstructure:"logdebug"`
	RPCListener  string `mapstructure:"rpclisten"`
	RESTListener string `mapstructure:"restlisten"`
}

func defaultServerConfig() *ServerConfig {
	return &ServerConfig{
		HomeDir:      defaultHomeDir,
		ConfigFile:   defaultConfigFile,
		LogDir:       defaultLogDir,
		LogDebug:     defaultLogDebug,
		RPCListener:  defaultRPCListener,
		RESTListener: defaultRESTListener,
	}
}

func loadConfig(cmd *cobra.Command) (*Config, error) {
	// Read in default config if passed as param using viper.
	fileLocation := smutil.GetCanonicalPath(viper.GetString("config"))
	vip := viper.New()

	_ = loadConfigFile(fileLocation, vip)

	// Load config if it was loaded to our viper.
	cfg := defaultConfig()
	err := vip.Unmarshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	// Ensure cli args are higher priority than the config file.
	ensureCLIFlags(cmd, cfg)

	cfg.ServerCfg.HomeDir = smutil.GetCanonicalPath(cfg.ServerCfg.HomeDir)
	cfg.ServerCfg.LogDir = smutil.GetCanonicalPath(cfg.ServerCfg.LogDir)

	// If the provided home directory is not the default, we'll modify the
	// path to all of the files and directories that will live within it.
	if cfg.ServerCfg.HomeDir != defaultHomeDir {
		cfg.ServerCfg.LogDir = cfg.ServerCfg.HomeDir
		cfg.PostCfg.DataDir = filepath.Join(cfg.ServerCfg.HomeDir, defaultDataDirName)
	}

	return cfg, nil
}

func loadConfigFile(fileLocation string, vip *viper.Viper) (err error) {
	if fileLocation == "" {
		fileLocation = defaultConfigFile
	}

	vip.SetConfigFile(fileLocation)
	err = vip.ReadInConfig()
	if err != nil {
		if fileLocation != defaultConfigFile {
			fmt.Printf("failed loading %v, trying %v", fileLocation, defaultConfigFileName) // WARNING
			vip.SetConfigFile(defaultConfigFile)
			err = vip.ReadInConfig()
		}

		// We modified err so check again.
		if err != nil {
			return fmt.Errorf("failed to read config file: %v", err)
		}
	}

	return nil
}

func setFlags(cmd *cobra.Command, cfg *Config) {
	flags := cmd.PersistentFlags()

	// Server config.

	flags.StringVar(&cfg.ServerCfg.ConfigFile, "config",
		cfg.ServerCfg.ConfigFile, "Path to configuration file")

	flags.StringVar(&cfg.ServerCfg.HomeDir, "homedir",
		cfg.ServerCfg.HomeDir, "The directory that contains the data, logs, configuration file, etc.")

	flags.StringVar(&cfg.ServerCfg.LogDir, "logdir",
		cfg.ServerCfg.LogDir, "Directory to log output")

	flags.BoolVar(&cfg.ServerCfg.LogDebug, "logdebug",
		cfg.ServerCfg.LogDebug, "Whether to enable debug logging")

	flags.StringVar(&cfg.ServerCfg.RPCListener, "rpclisten",
		cfg.ServerCfg.RPCListener, "The interface/port/socket to listen for RPC connections")

	flags.StringVar(&cfg.ServerCfg.RESTListener, "restlisten",
		cfg.ServerCfg.RESTListener, "The interface/port/socket to listen for REST connections")

	// POST config.
	// TODO(moshababo): add usage desc

	flags.UintVar(&cfg.PostCfg.NumFiles, "post-numfiles",
		cfg.PostCfg.NumFiles, "")

	flags.Uint64Var(&cfg.PostCfg.NumLabels, "post-numlabels",
		cfg.PostCfg.NumLabels, "")

	flags.UintVar(&cfg.PostCfg.LabelSize, "post-labelsize",
		cfg.PostCfg.LabelSize, "")

	flags.UintVar(&cfg.PostCfg.K1, "post-k1",
		cfg.PostCfg.K1, "")

	flags.UintVar(&cfg.PostCfg.K2, "post-k2",
		cfg.PostCfg.K2, "")

	err := viper.BindPFlags(flags)
	if err != nil {
		panic(err)
	}
}

func ensureCLIFlags(cmd *cobra.Command, cfg *Config) {
	assignFields := func(p reflect.Type, elem reflect.Value, name string) {
		for i := 0; i < p.NumField(); i++ {
			if p.Field(i).Tag.Get("mapstructure") == name {
				var val interface{}
				switch p.Field(i).Type.String() {
				case "bool":
					val = viper.GetBool(name)
				case "string":
					val = viper.GetString(name)
				case "int", "int8", "int16":
					val = viper.GetInt(name)
				case "int32":
					val = viper.GetInt32(name)
				case "int64":
					val = viper.GetInt64(name)
				case "uint", "uint8", "uint16":
					val = viper.GetUint(name)
				case "uint32":
					val = viper.GetUint32(name)
				case "uint64":
					val = viper.GetUint64(name)
				case "float64":
					val = viper.GetFloat64(name)
				default:
					val = viper.Get(name)
				}

				elem.Field(i).Set(reflect.ValueOf(val))
				return
			}
		}
	}

	// this is ugly but we have to do this because viper can't handle nested structs when deserialize
	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			name := f.Name

			ff := reflect.TypeOf(*cfg.ServerCfg)
			elem := reflect.ValueOf(cfg.ServerCfg).Elem()
			assignFields(ff, elem, name)

			ff = reflect.TypeOf(*cfg.PostCfg)
			elem = reflect.ValueOf(cfg.PostCfg).Elem()
			assignFields(ff, elem, name)
		}
	})
}
*/
