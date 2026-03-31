// Package config provides struct-based configuration for Cobra CLI commands.
//
// It allows defining configuration using struct tags, with automatic flag registration
// and value loading that respects the priority: flag > environment variable > default.
//
// Example usage:
//
//	type ServerConfig struct {
//	    Addr    string        `flag:"addr,a" env:"ADDR" default:":8080" desc:"server listen address"`
//	    Timeout time.Duration `flag:"timeout" env:"TIMEOUT" default:"30s" desc:"request timeout"`
//	}
//
//	var cfg ServerConfig
//
//	func init() {
//	    config.RegisterFlags(cmd, &cfg)
//	}
//
//	func run(cmd *cobra.Command) error {
//	    if err := config.Load(cmd, &cfg); err != nil {
//	        return err
//	    }
//	    // use cfg.Addr, cfg.Timeout, etc.
//	}
package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// fieldInfo holds parsed struct tag information for a config field.
type fieldInfo struct {
	flagName   string
	shorthand  string
	envKey     string
	defaultVal string
	desc       string
}

// RegisterFlags registers Cobra flags for each tagged field in the config struct.
// The cfg parameter must be a pointer to a struct.
//
// Supported struct tags:
//   - flag:"name" or flag:"name,shorthand" - the flag name and optional single-character shorthand
//   - env:"VAR_NAME" - environment variable name to check
//   - default:"value" - default value (parsed according to field type)
//   - desc:"description" - flag description shown in --help
//
// Supported field types: string, int, bool, time.Duration.
func RegisterFlags(cmd *cobra.Command, cfg any) error {
	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("config must be a pointer to a struct, got %T", cfg)
	}

	structVal := v.Elem()
	structType := structVal.Type()

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		info := parseFieldTags(&field)
		if info.flagName == "" {
			continue // skip fields without flag tag
		}

		if err := registerFlag(cmd, field.Type, &info); err != nil {
			return fmt.Errorf("registering flag for field %s: %w", field.Name, err)
		}
	}

	return nil
}

// Load populates the config struct with values using the priority:
// 1. Flag value (if explicitly set on command line)
// 2. Environment variable (if set and non-empty)
// 3. Default value from tag (or Go zero value)
//
// The cfg parameter must be a pointer to the same struct passed to RegisterFlags.
func Load(cmd *cobra.Command, cfg any) error {
	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("config must be a pointer to a struct, got %T", cfg)
	}

	structVal := v.Elem()
	structType := structVal.Type()

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		info := parseFieldTags(&field)
		if info.flagName == "" {
			continue // skip fields without flag tag
		}

		fieldVal := structVal.Field(i)
		if err := loadFieldValue(cmd, field.Type, fieldVal, &info); err != nil {
			return fmt.Errorf("loading value for field %s: %w", field.Name, err)
		}
	}

	return nil
}

// parseFieldTags extracts configuration from struct field tags.
func parseFieldTags(field *reflect.StructField) fieldInfo {
	info := fieldInfo{}

	flagTag := field.Tag.Get("flag")
	if flagTag == "" {
		return info
	}

	parts := strings.SplitN(flagTag, ",", 2)
	info.flagName = parts[0]
	if len(parts) > 1 {
		info.shorthand = parts[1]
	}

	info.envKey = field.Tag.Get("env")
	info.defaultVal = field.Tag.Get("default")
	info.desc = field.Tag.Get("desc")

	return info
}

// registerFlag registers a single flag based on field type.
func registerFlag(cmd *cobra.Command, fieldType reflect.Type, info *fieldInfo) error {
	switch fieldType.Kind() {
	case reflect.String:
		registerStringFlag(cmd, info)
		return nil
	case reflect.Int:
		return registerIntFlag(cmd, info)
	case reflect.Bool:
		return registerBoolFlag(cmd, info)
	case reflect.Int64:
		return registerInt64Flag(cmd, fieldType, info)
	default:
		return fmt.Errorf("unsupported field type: %s", fieldType)
	}
}

func registerStringFlag(cmd *cobra.Command, info *fieldInfo) {
	if info.shorthand != "" {
		cmd.Flags().StringP(info.flagName, info.shorthand, info.defaultVal, info.desc)
	} else {
		cmd.Flags().String(info.flagName, info.defaultVal, info.desc)
	}
}

func registerIntFlag(cmd *cobra.Command, info *fieldInfo) error {
	defaultInt := 0
	if info.defaultVal != "" {
		var err error
		defaultInt, err = strconv.Atoi(info.defaultVal)
		if err != nil {
			return fmt.Errorf("parsing default int value %q: %w", info.defaultVal, err)
		}
	}
	if info.shorthand != "" {
		cmd.Flags().IntP(info.flagName, info.shorthand, defaultInt, info.desc)
	} else {
		cmd.Flags().Int(info.flagName, defaultInt, info.desc)
	}
	return nil
}

func registerBoolFlag(cmd *cobra.Command, info *fieldInfo) error {
	defaultBool := false
	if info.defaultVal != "" {
		var err error
		defaultBool, err = strconv.ParseBool(info.defaultVal)
		if err != nil {
			return fmt.Errorf("parsing default bool value %q: %w", info.defaultVal, err)
		}
	}
	if info.shorthand != "" {
		cmd.Flags().BoolP(info.flagName, info.shorthand, defaultBool, info.desc)
	} else {
		cmd.Flags().Bool(info.flagName, defaultBool, info.desc)
	}
	return nil
}

func registerInt64Flag(cmd *cobra.Command, fieldType reflect.Type, info *fieldInfo) error {
	if fieldType != reflect.TypeOf(time.Duration(0)) {
		return fmt.Errorf("unsupported field type: %s", fieldType)
	}
	defaultDuration := time.Duration(0)
	if info.defaultVal != "" {
		var err error
		defaultDuration, err = time.ParseDuration(info.defaultVal)
		if err != nil {
			return fmt.Errorf("parsing default duration value %q: %w", info.defaultVal, err)
		}
	}
	if info.shorthand != "" {
		cmd.Flags().DurationP(info.flagName, info.shorthand, defaultDuration, info.desc)
	} else {
		cmd.Flags().Duration(info.flagName, defaultDuration, info.desc)
	}
	return nil
}

// loadFieldValue loads a value into the field using flag > env > default priority.
func loadFieldValue(cmd *cobra.Command, fieldType reflect.Type, fieldVal reflect.Value, info *fieldInfo) error {
	flagChanged := cmd.Flags().Changed(info.flagName)

	switch fieldType.Kind() {
	case reflect.String:
		val, err := getStringValue(cmd, info, flagChanged)
		if err != nil {
			return err
		}
		fieldVal.SetString(val)

	case reflect.Int:
		val, err := getIntValue(cmd, info, flagChanged)
		if err != nil {
			return err
		}
		fieldVal.SetInt(int64(val))

	case reflect.Bool:
		val, err := getBoolValue(cmd, info, flagChanged)
		if err != nil {
			return err
		}
		fieldVal.SetBool(val)

	case reflect.Int64:
		if fieldType == reflect.TypeOf(time.Duration(0)) {
			val, err := getDurationValue(cmd, info, flagChanged)
			if err != nil {
				return err
			}
			fieldVal.SetInt(int64(val))
		} else {
			return fmt.Errorf("unsupported field type: %s", fieldType)
		}

	default:
		return fmt.Errorf("unsupported field type: %s", fieldType)
	}

	return nil
}

// getStringValue returns the string value with flag > env > default priority.
func getStringValue(cmd *cobra.Command, info *fieldInfo, flagChanged bool) (string, error) {
	if flagChanged {
		val, err := cmd.Flags().GetString(info.flagName)
		if err != nil {
			return "", fmt.Errorf("getting flag %s: %w", info.flagName, err)
		}
		return val, nil
	}
	if info.envKey != "" {
		if envVal := os.Getenv(info.envKey); envVal != "" {
			return envVal, nil
		}
	}
	val, err := cmd.Flags().GetString(info.flagName)
	if err != nil {
		return "", fmt.Errorf("getting flag %s: %w", info.flagName, err)
	}
	return val, nil
}

// getIntValue returns the int value with flag > env > default priority.
func getIntValue(cmd *cobra.Command, info *fieldInfo, flagChanged bool) (int, error) {
	if flagChanged {
		val, err := cmd.Flags().GetInt(info.flagName)
		if err != nil {
			return 0, fmt.Errorf("getting flag %s: %w", info.flagName, err)
		}
		return val, nil
	}
	if info.envKey != "" {
		if envVal := os.Getenv(info.envKey); envVal != "" {
			parsed, err := strconv.Atoi(envVal)
			if err != nil {
				return 0, fmt.Errorf("parsing env var %s=%q as int: %w", info.envKey, envVal, err)
			}
			return parsed, nil
		}
	}
	val, err := cmd.Flags().GetInt(info.flagName)
	if err != nil {
		return 0, fmt.Errorf("getting flag %s: %w", info.flagName, err)
	}
	return val, nil
}

// getBoolValue returns the bool value with flag > env > default priority.
func getBoolValue(cmd *cobra.Command, info *fieldInfo, flagChanged bool) (bool, error) {
	if flagChanged {
		val, err := cmd.Flags().GetBool(info.flagName)
		if err != nil {
			return false, fmt.Errorf("getting flag %s: %w", info.flagName, err)
		}
		return val, nil
	}
	if info.envKey != "" {
		if envVal := os.Getenv(info.envKey); envVal != "" {
			parsed, err := strconv.ParseBool(envVal)
			if err != nil {
				return false, fmt.Errorf("parsing env var %s=%q as bool: %w", info.envKey, envVal, err)
			}
			return parsed, nil
		}
	}
	val, err := cmd.Flags().GetBool(info.flagName)
	if err != nil {
		return false, fmt.Errorf("getting flag %s: %w", info.flagName, err)
	}
	return val, nil
}

// getDurationValue returns the duration value with flag > env > default priority.
func getDurationValue(cmd *cobra.Command, info *fieldInfo, flagChanged bool) (time.Duration, error) {
	if flagChanged {
		val, err := cmd.Flags().GetDuration(info.flagName)
		if err != nil {
			return 0, fmt.Errorf("getting flag %s: %w", info.flagName, err)
		}
		return val, nil
	}
	if info.envKey != "" {
		if envVal := os.Getenv(info.envKey); envVal != "" {
			parsed, err := time.ParseDuration(envVal)
			if err != nil {
				return 0, fmt.Errorf("parsing env var %s=%q as duration: %w", info.envKey, envVal, err)
			}
			return parsed, nil
		}
	}
	val, err := cmd.Flags().GetDuration(info.flagName)
	if err != nil {
		return 0, fmt.Errorf("getting flag %s: %w", info.flagName, err)
	}
	return val, nil
}
