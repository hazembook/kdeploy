package ui

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// NotEmpty rejects blank values.
func NotEmpty(label string) func(string) error {
	return func(value string) error {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", label)
		}
		return nil
	}
}

// PositiveInt accepts a base-10 integer greater than zero.
func PositiveInt(label string) func(string) error {
	return func(value string) error {
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || n <= 0 {
			return fmt.Errorf("%s must be a positive integer", label)
		}
		return nil
	}
}

// HTTPURL accepts an absolute http or https URL.
func HTTPURL(label string) func(string) error {
	return func(value string) error {
		u, err := url.Parse(strings.TrimSpace(value))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("%s must be an http or https URL", label)
		}
		return nil
	}
}

// NoWhitespace rejects values containing spaces or tabs.
func NoWhitespace(label string) func(string) error {
	return func(value string) error {
		if strings.ContainsAny(value, " \t\r\n") {
			return fmt.Errorf("%s must not contain whitespace", label)
		}
		return nil
	}
}
