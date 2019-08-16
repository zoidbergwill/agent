package clicommand

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/buildkite/agent/agent"
	"github.com/buildkite/agent/api"
	"github.com/buildkite/agent/experiments"
	"github.com/buildkite/agent/logger"
	"github.com/oleiade/reflections"
	"github.com/urfave/cli"
)

const (
	DefaultEndpoint = "https://agent.buildkite.com/v3"
)

var AgentAccessTokenFlag = cli.StringFlag{
	Name:   "agent-access-token",
	Value:  "",
	Usage:  "The access token used to identify the agent",
	EnvVar: "BUILDKITE_AGENT_ACCESS_TOKEN",
}

var AgentRegisterTokenFlag = cli.StringFlag{
	Name:   "token",
	Value:  "",
	Usage:  "Your account agent token",
	EnvVar: "BUILDKITE_AGENT_TOKEN",
}

var EndpointFlag = cli.StringFlag{
	Name:   "endpoint",
	Value:  DefaultEndpoint,
	Usage:  "The Agent API endpoint",
	EnvVar: "BUILDKITE_AGENT_ENDPOINT",
}

var NoHTTP2Flag = cli.BoolFlag{
	Name:   "no-http2",
	Usage:  "Disable HTTP2 when communicating with the Agent API.",
	EnvVar: "BUILDKITE_NO_HTTP2",
}

var DebugFlag = cli.BoolFlag{
	Name:   "debug",
	Usage:  "Enable debug logging output",
	EnvVar: "BUILDKITE_AGENT_DEBUG",
}

var ProfileFlag = cli.StringFlag{
	Name:   "profile",
	Usage:  "Enable a profiling mode, either cpu, memory, mutex or block",
	EnvVar: "BUILDKITE_AGENT_PROFILE",
}

var DebugHTTPFlag = cli.BoolFlag{
	Name:   "debug-http",
	Usage:  "Enable HTTP debug mode, which dumps all request and response bodies to the log",
	EnvVar: "BUILDKITE_AGENT_DEBUG_HTTP",
}

var NoColorFlag = cli.BoolFlag{
	Name:   "no-color",
	Usage:  "Don't show colors in logging",
	EnvVar: "BUILDKITE_AGENT_NO_COLOR",
}

var ExperimentsFlag = cli.StringSliceFlag{
	Name:   "experiment",
	Value:  &cli.StringSlice{},
	Usage:  "Enable experimental features within the buildkite-agent",
	EnvVar: "BUILDKITE_AGENT_EXPERIMENT",
}

var LogFormatFlag = cli.StringFlag{
	Name:   "log-format",
	Usage:  "The format to use for the logger output",
	EnvVar: "BUILDKITE_LOG_FORMAT",
	Value:  "text",
}

var SilentFlag = cli.BoolFlag{
	Name:   "silent",
	Usage:  "Suppress all logging output",
	EnvVar: "BUILDKITE_AGENT_DEBUG",
}

func CreateLogger(cfg interface{}) logger.Logger {
	// If silent, just return a discard logger
	silent, _ := reflections.GetField(cfg, "Silent")
	if silent == true {
		return logger.Discard
	} 

	var l logger.Logger
	logFormat := `text`

	// Check the LogFormat config field
	if logFormatCfg, err := reflections.GetField(cfg, "LogFormat"); err == nil {
		if logFormatString, ok := logFormatCfg.(string); ok {
			logFormat = logFormatString
		}
	}

	// Create a logger based on the type
	switch logFormat {
	case `text`, ``:
		printer := logger.NewTextPrinter(os.Stderr)

		// Show agent fields as a prefix
		printer.IsPrefixFn = func(field logger.Field) bool {
			return field.Key() == `agent`
		}

		// Turn off color if a NoColor option is present
		noColor, err := reflections.GetField(cfg, "NoColor")
		if noColor == true && err == nil {
			printer.Colors = false
		} else {
			printer.Colors = true
		}

		l = logger.NewConsoleLogger(printer, os.Exit)
	case `json`:
		l = logger.NewConsoleLogger(logger.NewJSONPrinter(os.Stdout), os.Exit)
	default:
		fmt.Printf("Unknown log-format of %q, try text or json\n", logFormat)
		os.Exit(1)
	}

	return l
}

func HandleProfileFlag(l logger.Logger, cfg interface{}) func() {
	// Enable profiling a profiling mode if Profile is present
	modeField, _ := reflections.GetField(cfg, "Profile")
	if mode, ok := modeField.(string); ok && mode != "" {
		return Profile(l, mode)
	}
	return func() {}
}

func HandleGlobalFlags(l logger.Logger, cfg interface{}) func() {
	// Enable debugging if a Debug option is present
	debug, _ := reflections.GetField(cfg, "Debug")
	if debug == true {
		l.SetLevel(logger.DEBUG)
	} else {
		l.SetLevel(logger.NOTICE)
	}

	// Enable experiments
	experimentNames, err := reflections.GetField(cfg, "Experiments")
	if err == nil {
		experimentNamesSlice, ok := experimentNames.([]string)
		if ok {
			for _, name := range experimentNamesSlice {
				experiments.Enable(name)
				l.Debug("Enabled experiment `%s`", name)
			}
		}
	}

	// Handle profiling flag
	return HandleProfileFlag(l, cfg)
}

func UnsetConfigFromEnvironment(c *cli.Context) {
	flags := append(c.App.Flags, c.Command.Flags...)
	for _, fl := range flags {
		// use golang reflection to find EnvVar values on flags
		r := reflect.ValueOf(fl)
		f := reflect.Indirect(r).FieldByName(`EnvVar`)
		// split comma delimited env
		if envVars := f.String(); envVars != `` {
			for _, env := range strings.Split(envVars, ",") {
				os.Unsetenv(env)
			}
		}
	}
}

func loadAPIClientConfig(cfg interface{}, tokenField string) api.Config {
	conf := api.Config{
		UserAgent: agent.UserAgent(),
	}

	// Enable HTTP debugging
	debugHTTP, err := reflections.GetField(cfg, "DebugHTTP")
	if debugHTTP == true && err == nil {
		conf.DebugHTTP = true
	}

	endpoint, err := reflections.GetField(cfg, "Endpoint")
	if endpoint != "" && err == nil {
		conf.Endpoint = endpoint.(string)
	}

	token, err := reflections.GetField(cfg, tokenField)
	if token != "" && err == nil {
		conf.Token = token.(string)
	}

	noHTTP2, err := reflections.GetField(cfg, "NoHTTP2")
	if err == nil {
		conf.DisableHTTP2 = noHTTP2.(bool)
	}

	return conf
}
