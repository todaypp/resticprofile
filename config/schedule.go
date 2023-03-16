package config

import "time"

// Schedule contains the information from the schedule profile in the configuration file.
// The structure is a direct mapping of a schedule section (configuration file v2+)
type Schedule struct {
	config     *Config
	Name       string
	Group      string        `mapstructure:"group"`
	Profiles   []string      `mapstructure:"profiles"`
	Command    string        `mapstructure:"run"`
	Schedule   []string      `mapstructure:"schedule"`
	Permission string        `mapstructure:"permission"`
	Log        string        `mapstructure:"log"`
	Priority   string        `mapstructure:"priority"`
	LockMode   string        `mapstructure:"lock-mode"`
	LockWait   time.Duration `mapstructure:"lock-wait"`
}

// NewSchedule instantiates a new blank schedule
func NewSchedule(c *Config, name string) *Schedule {
	return &Schedule{
		Name:   name,
		config: c,
	}
}

func (s *Schedule) GetScheduleConfig() *ScheduleConfig {
	title := s.Name
	subTitle := ""
	if title == "" {
		if len(s.Profiles) > 0 {
			title = s.Profiles[0]
		}
		subTitle = s.Command
	}
	return &ScheduleConfig{
		Title:      title,
		SubTitle:   subTitle,
		Schedules:  s.Schedule,
		Permission: s.Permission,
		Priority:   s.Priority,
		Log:        s.Log,
		LockMode:   s.LockMode,
		LockWait:   s.LockWait,
		ConfigFile: s.config.configFile,
	}
}
