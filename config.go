package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds the application configuration.
type Config struct {
	OrgID                     string
	SlackToken                string
	SlackChannelID            string
	MaxNotificationAgeSeconds int64
	Debug                     bool
}

const (
	safetyMarginMaxNotificationAgeSeconds = 600
)

// getMandatoryEnvVar retrieves the value of an environment variable.
// It returns an error if the variable is not set or is empty.
func getMandatoryEnvVar(varName string) (string, error) {
	value, ok := os.LookupEnv(varName)
	if !ok || value == "" {
		return "", fmt.Errorf("%s environment variable is not defined or empty", varName)
	}
	return value, nil
}

// LoadConfig loads configuration from environment variables.
// It returns an error if any required variable is missing.
func LoadConfig() (*Config, error) {
	orgID, err := getMandatoryEnvVar("ADV_NOTIF_ORG_ID")
	if err != nil {
		return nil, err
	}

	slackToken, err := getMandatoryEnvVar("ADV_NOTIF_SLACK_TOKEN")
	if err != nil {
		return nil, err
	}

	slackChannelID, err := getMandatoryEnvVar("ADV_NOTIF_SLACK_CHANNEL_ID")
	if err != nil {
		return nil, err
	}

	maxNotificationAgeSecondsStr, err := getMandatoryEnvVar("ADV_NOTIF_MAX_NOTIFICATION_AGE_SECONDS")
	if err != nil {
		return nil, err
	}

	maxNotificationAgeSeconds, err := strconv.ParseInt(maxNotificationAgeSecondsStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("ADV_NOTIF_MAX_NOTIFICATION_AGE_SECONDS must be a valid integer: %v", err)
	}

	isDebug, _ := os.LookupEnv("ADV_NOTIF_DEBUG")

	return &Config{
		OrgID:                     orgID,
		SlackToken:                slackToken,
		SlackChannelID:            slackChannelID,
		MaxNotificationAgeSeconds: maxNotificationAgeSeconds + safetyMarginMaxNotificationAgeSeconds,
		Debug:                     strings.ToUpper(isDebug) == "TRUE",
	}, nil
}
