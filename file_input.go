// https://github.com/sl1pm4t/k2tf/blob/master/pkg/file_io/input.go
// Adapted to fit to loki promehteus rules files.

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana/loki/pkg/logql/syntax"
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"

	"github.com/rs/zerolog/log"
	yaml "gopkg.in/yaml.v3"
)

func ReadYAMLInput(input string) []RuleNamespace {
	if input == "-" || input == "" {
		return readYAMLStdinInput(input)
	}
	return readYAMLFilesInput(input)
}

func readYAMLStdinInput(input string) []RuleNamespace {
	info, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}

	if info.Mode()&os.ModeCharDevice != 0 {
		log.Fatal().Msg("No data read from stdin")
	}

	reader := bufio.NewReader(os.Stdin)
	buf := &bytes.Buffer{}
	buf.ReadFrom(reader)
	parsed, errs := parseRulesBytes(buf.Bytes())

	if len(errs) > 0 {
		log.Fatal().Err(errs[0]).Msg("Could not parse stdin")
	}

	return parsed

}

func readYAMLFilesInput(input string) []RuleNamespace {
	var objs []RuleNamespace

	if _, err := os.Stat(input); os.IsNotExist(err) {
		log.Fatal().Str("file", input).Msg("input filepath does not exist")
	}

	file, err := os.Open(input)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	fs, err := file.Stat()
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	readYamlFile := func(fileName string) {
		log.Debug().Msgf("reading file: %s", fileName)
		content, err := ioutil.ReadFile(fileName)
		if err != nil {
			log.Fatal().Err(err).Msg("could not read file")
		}

		r := bytes.NewReader(content)
		buf := &bytes.Buffer{}
		buf.ReadFrom(r)
		obj, errs := parseRulesBytes(buf.Bytes())
		if len(errs) > 0 {
			log.Warn().Err(errs[0]).Msgf("could not parse file %s", fileName)
		}
		objs = append(objs, obj...)
	}

	if fs.Mode().IsDir() {
		// read directory
		log.Debug().Msgf("reading directory: %s", input)

		dirContents, err := file.Readdirnames(0)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}

		for _, f := range dirContents {
			if strings.HasSuffix(f, ".yml") || strings.HasSuffix(f, ".yaml") {
				readYamlFile(filepath.Join(input, f))
			}
		}

	} else {
		// read single file
		readYamlFile(input)

	}

	return objs
}

func ReadHCLInput(input string) []map[string]interface{} {
	if input == "-" || input == "" {
		return readHCLStdinInput(input)
	}
	return readHCLFilesInput(input)
}

func readHCLStdinInput(input string) []map[string]interface{} {
	info, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}

	if info.Mode()&os.ModeCharDevice != 0 {
		log.Fatal().Msg("No data read from stdin")
	}

	buffer := bytes.NewBuffer([]byte{})
	var stream io.Reader
	_, err = buffer.ReadFrom(stream)

	dataBytes, err := Bytes(buffer.Bytes(), "STDIN")
	if err != nil {
		log.Fatal().Err(err).Msg("Could not parse stdin")
	}
	var data map[string]interface{}
	err = json.Unmarshal(dataBytes, &data)
	if err != nil {
		log.Warn().Err(err).Msgf("could not unmarshal")
	}

	return []map[string]interface{}{data}

}

func readHCLFilesInput(input string) []map[string]interface{} {
	var objs []map[string]interface{}

	if _, err := os.Stat(input); os.IsNotExist(err) {
		log.Fatal().Str("file", input).Msg("input filepath does not exist")
	}

	file, err := os.Open(input)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	fs, err := file.Stat()
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	readHCLFile := func(fileName string) {
		log.Debug().Msgf("reading file: %s", fileName)
		content, err := ioutil.ReadFile(fileName)
		if err != nil {
			log.Fatal().Err(err).Msg("could not read file")
		}

		r := bytes.NewReader(content)
		buf := &bytes.Buffer{}
		buf.ReadFrom(r)

		dataBytes, err := Bytes(buf.Bytes(), "STDIN")
		if err != nil {
			log.Warn().Err(err).Msgf("could not parse file %s", fileName)
		}
		var obj map[string]interface{}
		err = json.Unmarshal(dataBytes, &obj)
		if err != nil {
			log.Warn().Err(err).Msgf("could not unmarshal file %s", fileName)
		}
		objs = append(objs, obj)
	}

	if fs.Mode().IsDir() {
		// read directory
		log.Debug().Msgf("reading directory: %s", input)

		dirContents, err := file.Readdirnames(0)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}

		for _, f := range dirContents {
			if strings.HasSuffix(f, ".tf") {
				readHCLFile(filepath.Join(input, f))
			}
		}

	} else {
		// read single file
		readHCLFile(input)

	}

	return objs
}

func parseRulesBytes(content []byte) ([]RuleNamespace, []error) {
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)

	var nss []RuleNamespace
	for {
		var ns RuleNamespace
		err := decoder.Decode(&ns)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, []error{err}
		}

		if errs := validateGroups(ns.Groups...); len(errs) > 0 {
			return nil, errs
		}

		nss = append(nss, ns)

	}
	return nss, nil
}

func validateGroups(grps ...rulefmt.RuleGroup) (errs []error) {
	set := map[string]struct{}{}

	for i, g := range grps {
		if g.Name == "" {
			errs = append(errs, errors.Errorf("group %d: Groupname must not be empty", i))
		}

		if _, ok := set[g.Name]; ok {
			errs = append(
				errs,
				errors.Errorf("groupname: \"%s\" is repeated in the same file", g.Name),
			)
		}

		set[g.Name] = struct{}{}

		for _, r := range g.Rules {
			if err := validateRuleNode(&r, g.Name); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errs
}

func validateRuleNode(r *rulefmt.RuleNode, groupName string) error {
	if r.Record.Value != "" && r.Alert.Value != "" {
		return errors.Errorf("only one of 'record' and 'alert' must be set")
	}

	if r.Record.Value == "" && r.Alert.Value == "" {
		return errors.Errorf("one of 'record' or 'alert' must be set")
	}

	if r.Expr.Value == "" {
		return errors.Errorf("field 'expr' must be set in rule")
	} else if _, err := syntax.ParseExpr(r.Expr.Value); err != nil {
		if r.Record.Value != "" {
			return errors.Wrapf(err, fmt.Sprintf("could not parse expression for record '%s' in group '%s'", r.Record.Value, groupName))
		} else {
			return errors.Wrapf(err, fmt.Sprintf("could not parse expression for alert '%s' in group '%s'", r.Alert.Value, groupName))
		}
	}

	if r.Record.Value != "" {
		if len(r.Annotations) > 0 {
			return errors.Errorf("invalid field 'annotations' in recording rule")
		}
		if r.For != 0 {
			return errors.Errorf("invalid field 'for' in recording rule")
		}
		if !model.IsValidMetricName(model.LabelValue(r.Record.Value)) {
			return errors.Errorf("invalid recording rule name: %s", r.Record.Value)
		}
	}

	for k, v := range r.Labels {
		if !model.LabelName(k).IsValid() || k == model.MetricNameLabel {
			return errors.Errorf("invalid label name: %s", k)
		}

		if !model.LabelValue(v).IsValid() {
			return errors.Errorf("invalid label value: %s", v)
		}
	}

	for k := range r.Annotations {
		if !model.LabelName(k).IsValid() {
			return errors.Errorf("invalid annotation name: %s", k)
		}
	}

	return nil
}

type RuleNamespace struct {
	Groups    []rulefmt.RuleGroup `yaml:"groups"`
	Namespace string              `yaml:"namespace,omitempty"`
}
