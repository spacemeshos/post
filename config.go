package main

import (
	"fmt"
	"github.com/btcsuite/btcutil"
	"github.com/jessevdk/go-flags"
	"github.com/spacemeshos/post/shared"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

const (
	defaultConfigFilename = "post.conf"
	defaultDataDirname    = "data"
	defaultLogLevel       = "debug"
	defaultLabelsLogRate  = 5000000
	defaultLogDirname     = "logs"
	defaultLogFilename    = "post.log"
	defaultMaxLogFiles    = 3
	defaultMaxLogFileSize = 10
	defaultRPCPort        = 50001
	defaultRESTPort       = 8080
)

var (
	defaultPostDir    = btcutil.AppDataDir("post", false)
	defaultConfigFile = filepath.Join(defaultPostDir, defaultConfigFilename)
	defaultDataDir    = filepath.Join(defaultPostDir, defaultDataDirname)
	defaultLogDir     = filepath.Join(defaultPostDir, defaultLogDirname)
)

// config defines the configuration options for post.
//
// See loadConfig for further details regarding the
// configuration loading+parsing process.
type config struct {
	PostDir         string `long:"postdir" description:"The base directory that contains the PoST's data, logs, configuration file, etc."`
	ConfigFile      string `short:"c" long:"configfile" description:"Path to configuration file"`
	DataDir         string `short:"b" long:"datadir" description:"The directory to store post's data within"`
	LogDir          string `long:"logdir" description:"Directory to log output."`
	MaxLogFiles     int    `long:"maxlogfiles" description:"Maximum logfiles to keep (0 for no rotation)"`
	MaxLogFileSize  int    `long:"maxlogfilesize" description:"Maximum logfile size in MB"`
	RawRPCListener  string `short:"r" long:"rpclisten" description:"The interface/port/socket to listen for RPC connections"`
	RawRESTListener string `short:"w" long:"restlisten" description:"The interface/port/socket to listen for REST connections"`
	RPCListener     net.Addr
	RESTListener    net.Addr

	LogLevel      string `long:"loglevel" description:"Logging level for all subsystems {trace, debug, info, warn, error, critical}"`
	LabelsLogRate uint64 `long:"lograte" description:"Labels construction progress log rate"`
	CPUProfile    string `long:"cpuprofile" description:"Write CPU profile to the specified file"`
	Profile       string `long:"profile" description:"Enable HTTP profiling on given port -- NOTE port must be between 1024 and 65535"`

	Params *shared.Params `group:"params"`
}

// loadConfig initializes and parses the config using a config file and command
// line options.
//
// The configuration proceeds as follows:
// 	1) Start with a default config with sane settings
// 	2) Pre-parse the command line to check for an alternative config file
// 	3) Load configuration file overwriting defaults with any specified options
// 	4) Parse CLI options and overwrite/add any specified options
func loadConfig() (*config, error) {
	defaultCfg := config{
		Params:          shared.DefaultParams(),
		PostDir:         defaultPostDir,
		ConfigFile:      defaultConfigFile,
		DataDir:         defaultDataDir,
		LogDir:          defaultLogDir,
		LogLevel:        defaultLogLevel,
		LabelsLogRate:   defaultLabelsLogRate,
		MaxLogFiles:     defaultMaxLogFiles,
		MaxLogFileSize:  defaultMaxLogFileSize,
		RawRPCListener:  fmt.Sprintf("localhost:%d", defaultRPCPort),
		RawRESTListener: fmt.Sprintf("localhost:%d", defaultRESTPort),
	}

	// Pre-parse the command line options to pick up an alternative config
	// file.
	preCfg := defaultCfg
	if _, err := flags.Parse(&preCfg); err != nil {
		return nil, err
	}

	appName := filepath.Base(os.Args[0])
	appName = strings.TrimSuffix(appName, filepath.Ext(appName))
	usageMessage := fmt.Sprintf("Use %s -h to show usage", appName)

	// If the config file path has not been modified by the user, then we'll
	// use the default config file path. However, if the user has modified
	// their postdir, then we should assume they intend to use the config
	// file within it.

	preCfg.PostDir = cleanAndExpandPath(preCfg.PostDir)
	preCfg.ConfigFile = cleanAndExpandPath(preCfg.ConfigFile)
	if preCfg.PostDir != defaultPostDir {
		if preCfg.ConfigFile == defaultConfigFile {
			preCfg.ConfigFile = filepath.Join(
				preCfg.PostDir, defaultConfigFilename,
			)
		}
	}

	// Next, load any additional configuration options from the file.
	var configFileError error
	cfg := preCfg
	if err := flags.IniParse(preCfg.ConfigFile, &cfg); err != nil {
		// If it's a parsing related error, then we'll return
		// immediately, otherwise we can proceed as possibly the config
		// file doesn't exist which is OK.
		if _, ok := err.(*flags.IniError); ok {
			return nil, err
		}

		configFileError = err
	}

	// Finally, parse the remaining command line options again to ensure
	// they take precedence.
	if _, err := flags.Parse(&cfg); err != nil {
		return nil, err
	}

	// If the provided PoST directory is not the default, we'll modify the
	// path to all of the files and directories that will live within it.
	if cfg.PostDir != defaultPostDir {
		cfg.DataDir = filepath.Join(cfg.PostDir, defaultDataDirname)
		cfg.LogDir = filepath.Join(cfg.PostDir, defaultLogDirname)
	}

	// Create the post directory if it doesn't already exist.
	funcName := "loadConfig"
	if err := os.MkdirAll(cfg.PostDir, 0700); err != nil {
		// Show a nicer error message if it's because a symlink is
		// linked to a directory that does not exist (probably because
		// it's not mounted).
		if e, ok := err.(*os.PathError); ok && os.IsExist(err) {
			if link, lerr := os.Readlink(e.Path); lerr == nil {
				str := "is symlink %s -> %s mounted?"
				err = fmt.Errorf(str, e.Path, link)
			}
		}

		str := "%s: Failed to create post directory: %v"
		err := fmt.Errorf(str, funcName, err)
		_, _ = fmt.Fprintln(os.Stderr, err)
		return nil, err
	}

	// As soon as we're done parsing configuration options, ensure all paths
	// to directories and files are cleaned and expanded before attempting
	// to use them later on.
	cfg.DataDir = cleanAndExpandPath(cfg.DataDir)
	cfg.LogDir = cleanAndExpandPath(cfg.LogDir)

	// Initialize logging at the default logging level.
	initLogRotator(
		filepath.Join(cfg.LogDir, defaultLogFilename),
		cfg.MaxLogFileSize, cfg.MaxLogFiles,
	)

	if !validLogLevel(cfg.LogLevel) {
		_, _ = fmt.Fprintln(os.Stderr, usageMessage)
		err := fmt.Errorf("the specified log level (%v) is invalid", cfg.LogLevel)
		return nil, err
	}

	// Change the logging level for all subsystems.
	setLogLevels(cfg.LogLevel)

	// Resolve the RPC listener
	addr, err := net.ResolveTCPAddr("tcp", cfg.RawRPCListener)
	if err != nil {
		return nil, err
	}
	cfg.RPCListener = addr

	// Resolve the REST listener
	addr, err = net.ResolveTCPAddr("tcp", cfg.RawRESTListener)
	if err != nil {
		return nil, err
	}
	cfg.RESTListener = addr

	// Warn about missing config file only after all other configuration is
	// done. This prevents the warning on help messages and invalid
	// options. Note this should go directly before the return.
	if configFileError != nil {
		postLog.Warnf("%v", configFileError)
	}

	return &cfg, nil
}

// cleanAndExpandPath expands environment variables and leading ~ in the
// passed path, cleans the result, and returns it.
// This function is taken from https://github.com/btcsuite/btcd
func cleanAndExpandPath(path string) string {
	if path == "" {
		return ""
	}

	// Expand initial ~ to OS specific home directory.
	if strings.HasPrefix(path, "~") {
		var homeDir string
		user, err := user.Current()
		if err == nil {
			homeDir = user.HomeDir
		} else {
			homeDir = os.Getenv("HOME")
		}

		path = strings.Replace(path, "~", homeDir, 1)
	}

	// NOTE: The os.ExpandEnv doesn't work with Windows-style %VARIABLE%,
	// but the variables can still be expanded via POSIX-style $VARIABLE.
	return filepath.Clean(os.ExpandEnv(path))
}

// validLogLevel returns whether or not logLevel is a valid debug log level.
func validLogLevel(logLevel string) bool {
	switch logLevel {
	case "trace":
		fallthrough
	case "debug":
		fallthrough
	case "info":
		fallthrough
	case "warn":
		fallthrough
	case "error":
		fallthrough
	case "critical":
		fallthrough
	case "off":
		return true
	}
	return false
}
