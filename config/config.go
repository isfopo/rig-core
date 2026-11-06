// Package config provides shared configuration loading helpers for rig apps.
//
// Each app defines its own schema on top of this package. Configuration is
// TOML by convention; files live at ~/.config/rig/<app>.toml unless an
// explicit path is given.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Dir returns the per-user rig configuration directory (~/.config/rig).
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("config: home directory: %w", err)
	}
	return filepath.Join(home, ".config", "rig"), nil
}

// Path returns the conventional config file path for an app.
func Path(app string) (string, error) {
	if app == "" {
		return "", errors.New("config: app name is required")
	}
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, app+".toml"), nil
}

// Load decodes the TOML file at path into v. A missing file is reported as
// os.ErrNotExist.
func Load(path string, v any) error {
	if path == "" {
		return errors.New("config: path is required")
	}
	if _, err := toml.DecodeFile(path, v); err != nil {
		return fmt.Errorf("config: decode %s: %w", path, err)
	}
	return nil
}

// LoadApp decodes the conventional config file for app into v and returns
// its path. A missing file is reported as os.ErrNotExist.
func LoadApp(app string, v any) (string, error) {
	path, err := Path(app)
	if err != nil {
		return "", err
	}
	if err := Load(path, v); err != nil {
		return path, err
	}
	return path, nil
}

// Save encodes v as TOML and writes it to path, creating parent directories
// as needed. The write is atomic (temp file + rename).
func Save(path string, v any) error {
	if path == "" {
		return errors.New("config: path is required")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("config: create %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".config-*")
	if err != nil {
		return fmt.Errorf("config: temp file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	if err := toml.NewEncoder(tmp).Encode(v); err != nil {
		tmp.Close()
		cleanup()
		return fmt.Errorf("config: encode: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("config: close: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return fmt.Errorf("config: rename: %w", err)
	}
	return nil
}

// SaveApp encodes v to the conventional config file for app.
func SaveApp(app string, v any) (string, error) {
	path, err := Path(app)
	if err != nil {
		return "", err
	}
	if err := Save(path, v); err != nil {
		return path, err
	}
	return path, nil
}

// Validator is implemented by config structs that can validate themselves.
type Validator interface {
	Validate() error
}

// LoadValidated decodes path into v and calls Validate if v implements
// Validator.
func LoadValidated(path string, v any) error {
	if err := Load(path, v); err != nil {
		return err
	}
	if val, ok := v.(Validator); ok {
		return val.Validate()
	}
	return nil
}

// CheckNonEmpty returns an error if value is empty.
func CheckNonEmpty(name, value string) error {
	if value == "" {
		return fmt.Errorf("config: %s is required", name)
	}
	return nil
}

// CheckPositive returns an error if value is not greater than zero.
func CheckPositive(name string, value float64) error {
	if value <= 0 {
		return fmt.Errorf("config: %s must be > 0", name)
	}
	return nil
}

// CheckRange returns an error if value is outside [min,max].
func CheckRange(name string, value, min, max float64) error {
	if value < min || value > max {
		return fmt.Errorf("config: %s must be in [%g,%g]", name, min, max)
	}
	return nil
}
