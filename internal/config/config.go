package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

func ReadJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode config: %w", err)
	}
	return nil
}

func DurationSeconds(name, value string) (int64, error) {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	if duration <= 0 || duration%time.Second != 0 {
		return 0, fmt.Errorf("parse %s: duration must be a positive whole number of seconds", name)
	}
	return int64(duration / time.Second), nil
}
