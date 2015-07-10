// cfgp - go configuration file parser package
// Copyright (c) 2015 Andrea Masi

// Package cfgp is a configuration parser fo Go.
//
// Just define a struct with needed configurations. Values are then taken from multiple source
// in this order of precendece:
//
// 	- env variables
// 	- command line arguments (which are automagically created and parsed)
// 	- configuration file
//
// Tags
//
// Default is to use field names in struct to create flags,
// search for env variables and configuration into files.
// Tags can be used to specify different name, flag help message
// in command line, and section in conf file.
// Format is:
//	<name>,<help message>,<section in file>
//
// For file, only INI format supported for now. Files must follows INI informal standard:
//
//	https://en.wikipedia.org/wiki/INI_file
//
// It tries to be modular and easily extendible to support different formats.
// This is a work in progress, better packages are out there.
package cfgp

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/eraclitux/stracer"
)

var ErrNeedPointer = errors.New("cfgp: pointer to struct expected")
var ErrFileFormat = errors.New("cfgp: unrecognized file format, only (ini|txt|cfg) supported")
var ErrUnknownFlagType = errors.New("cfgp: unknown flag type")

func getStructValue(confPtr interface{}) (reflect.Value, error) {
	v := reflect.ValueOf(confPtr)
	if v.Kind() == reflect.Ptr {
		return v.Elem(), nil
	}
	return reflect.Value{}, ErrNeedPointer
}

// myFlag implements Flag.Value.
// TODO is filed needed?
type myFlag struct {
	field      reflect.StructField
	fieldValue reflect.Value
	isBool     bool
}

func (s *myFlag) String() string {
	return s.field.Name
}

// IsBoolFlag istructs the command-line parser
// to makes -name equivalent to -name=true rather than
// using the next command-line argument.
func (s *myFlag) IsBoolFlag() bool {
	return s.isBool
}

func (s *myFlag) Set(arg string) error {
	stracer.Traceln("setting flag", s.field.Name)
	switch s.fieldValue.Kind() {
	case reflect.Int:
		n, err := strconv.Atoi(arg)
		if err != nil {
			return err
		}
		s.fieldValue.SetInt(int64(n))
	case reflect.String:
		s.fieldValue.SetString(arg)
	case reflect.Bool:
		b, err := strconv.ParseBool(arg)
		if err != nil {
			return err
		}
		s.fieldValue.SetBool(b)
	default:
		return ErrUnknownFlagType
	}
	return nil
}

func helpMessageFromTags(f reflect.StructField) (string, bool) {
	t := f.Tag.Get("cfgp")
	tags := strings.Split(t, ",")
	if len(tags) == 3 {
		return tags[1], true
	}
	return "", false
}

func makeHelpMessage(f reflect.StructField) string {
	var helpM string
	switch f.Type.Kind() {
	case reflect.Int:
		if m, ok := helpMessageFromTags(f); ok {
			helpM = m + ", an int value"
		} else {
			helpM = "set an int value"
		}
	case reflect.String:
		if m, ok := helpMessageFromTags(f); ok {
			helpM = m + ", a string value"
		} else {
			helpM = "set a string value"
		}
	case reflect.Bool:
		if m, ok := helpMessageFromTags(f); ok {
			helpM = m + ", a bool value"
		} else {
			helpM = "set a bool value"
		}
	default:
		helpM = "unknown flag kind"
	}
	return helpM
}

func isBool(v reflect.Value) bool {
	if v.Kind() == reflect.Bool {
		return true
	}
	return false
}

func nameFromTags(f reflect.StructField) (string, bool) {
	t := f.Tag.Get("cfgp")
	tags := strings.Split(t, ",")
	if len(tags) == 3 {
		return tags[0], true
	}
	return "", false
}

// FIXME can we semplify using structType := structValue.Type()?
func createFlag(f reflect.StructField, fieldValue reflect.Value, fs *flag.FlagSet) {
	name := strings.ToLower(f.Name)
	if n, ok := nameFromTags(f); ok {
		name = n
	}
	stracer.Traceln("creating flag:", name)
	fs.Var(&myFlag{f, fieldValue, isBool(fieldValue)}, name, makeHelpMessage(f))
}

func parseFlags(s reflect.Value) error {
	flagSet := flag.NewFlagSet("cfgp", flag.ExitOnError)
	flagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flagSet.PrintDefaults()
	}
	typeOfT := s.Type()
	for i := 0; i < s.NumField(); i++ {
		fieldValue := s.Field(i)
		if fieldValue.CanSet() {
			createFlag(typeOfT.Field(i), fieldValue, flagSet)
		}
	}
	err := flagSet.Parse(os.Args[1:])
	if err != nil {
		stracer.Traceln("this is not executed")
		return err
	}
	return nil
}

// Parse popolate passed struct (via pointer) with configuration from varoius source.
// It guesses configuration type by file extention and call specific parser.
// (.ini|.txt|.cfg) are evaluated as INI files which is to only format supported for now.
// path can be an empty string to disable file parsing.
func Parse(path string, confPtr interface{}) error {
	structValue, err := getStructValue(confPtr)
	if err != nil {
		return err
	}
	if path != "" {
		if match, _ := regexp.MatchString(`\.(ini|txt|cfg)$`, path); match {
			err := parseINI(path, structValue)
			if err != nil {
				return err
			}
		} else if match, _ := regexp.MatchString(`\.(yaml)$`, path); match {
			return errors.New("YAML not yet implemented. Want you help?")
		} else {
			return ErrFileFormat
		}
	}
	err = parseFlags(structValue)
	if err != nil {
		return err
	}
	return nil
}
