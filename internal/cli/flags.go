package cli

import (
	"strconv"
	"strings"
)

type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type stringFlag struct {
	value string
	set   bool
}

func (s *stringFlag) String() string {
	return s.value
}

func (s *stringFlag) Set(value string) error {
	s.value = value
	s.set = true
	return nil
}

type boolFlag struct {
	value bool
	set   bool
}

func (b *boolFlag) String() string {
	if b == nil {
		return "false"
	}
	if b.value {
		return "true"
	}
	return "false"
}

func (b *boolFlag) Set(value string) error {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	b.value = parsed
	b.set = true
	return nil
}

func (b *boolFlag) IsBoolFlag() bool {
	return true
}
