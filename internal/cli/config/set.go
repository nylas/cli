package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a specific configuration value using dot notation.

The configuration file will be created if it doesn't exist.`,
		Example: `  # Set API timeout
  nylas config set api.timeout 120s

  # Set default grant ID
  nylas config set default_grant grant_abc123

  # Set output format
  nylas config set output.format json

  # Set output color mode
  nylas config set output.color never

  # Set GPG default signing key
  nylas config set gpg.default_key 601FEE9B1D60185F

  # Enable auto-sign for all emails
  nylas config set gpg.auto_sign true`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := configStore.Load()
			if err != nil {
				return common.WrapLoadError("configuration", err)
			}

			key := args[0]
			value := args[1]

			if err := setConfigValue(cfg, key, value); err != nil {
				return err
			}

			if err := configStore.Save(cfg); err != nil {
				return common.WrapSaveError("configuration", err)
			}

			fmt.Printf("%s Configuration updated: %s = %s\n", common.Green.Sprint("✓"), key, value)
			fmt.Printf("Config file: %s\n", configStore.Path())
			return nil
		},
	}
}

func setConfigValue(cfg any, key, value string) error {
	parts := strings.Split(key, ".")

	v := reflect.ValueOf(cfg)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	for i, part := range parts {
		if v.Kind() != reflect.Struct {
			return fmt.Errorf("cannot access field %s", part)
		}

		fieldName := snakeToPascal(part)
		field := v.FieldByName(fieldName)

		if !field.IsValid() {
			return fmt.Errorf("unknown config key: %s", key)
		}

		// If this is the last part, set the value
		if i == len(parts)-1 {
			return setFieldValue(field, value)
		}

		// If field is a pointer, dereference or initialize
		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				// Initialize the struct
				field.Set(reflect.New(field.Type().Elem()))
			}
			field = field.Elem()
		}

		v = field
	}

	return nil
}

func setFieldValue(field reflect.Value, value string) error {
	if !field.CanSet() {
		return fmt.Errorf("cannot set field")
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int64:
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer value: %s", value)
		}
		field.SetInt(i)
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value: %s (use true/false)", value)
		}
		field.SetBool(b)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}

	return nil
}
