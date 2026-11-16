package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/dioad/config/configcheck"
)

// ReferenceChecker is implemented by a config that refers to other things it
// can only be fully checked against, such as plugin manifests or files it
// names. ValidateConfigFile calls CheckReferences after Validate.
type ReferenceChecker interface {
	// CheckReferences adds problems with the config itself that need its
	// references to find, such as an entry whose type nothing provides, to
	// findings. It returns problems inside a referenced file as a
	// *FileError, joined with errors.Join if there are several.
	CheckReferences(findings *configcheck.Collector) error
}

// FileError is a problem inside a file a config refers to, reported under
// that file's path.
type FileError struct {
	Path string
	Err  error
}

// Error returns the file's path followed by its problems.
func (e *FileError) Error() string {
	return e.Path + ":\n" + e.Err.Error()
}

// Unwrap returns the problems inside the file.
func (e *FileError) Unwrap() error {
	return e.Err
}

// ValidateConfigFileOpt is a functional option for ValidateConfigFile.
type ValidateConfigFileOpt func(*validateConfigFileOptions)

type validateConfigFileOptions struct {
	envPrefix string
}

// WithEnvOverrides makes ValidateConfigFile apply the environment variables
// that override the file's settings when InitConfig loads it for appName, so
// the result is for the configuration the service would start with here and
// not for the file alone.
//
// As in InitConfig, a variable overrides only a setting the file lists, and an
// empty variable overrides nothing. A variable that matches no setting is not
// a problem: the environment is shared with everything else on the machine.
func WithEnvOverrides(appName string) ValidateConfigFileOpt {
	return func(o *validateConfigFileOptions) { o.envPrefix = appName }
}

// mode says whether the environment was applied, for the result line.
func (o validateConfigFileOptions) mode() string {
	if o.envPrefix == "" {
		return "environment overrides not applied"
	}

	return "environment overrides applied"
}

// ValidateConfigFile checks the config file at path as the service would at
// startup, without starting it, and writes the result to w: every problem, one
// per line under the path of the file it is in, or a line saying the file is
// valid. The file is decoded strictly into cfg, so a key that matches no field
// is a problem, and cfg's Validate method is then called, so a setting it
// rejects is too. If cfg is a ReferenceChecker, what it refers to is checked
// as well. All are reported together. It returns an error if the file is
// invalid or cannot be read.
//
// Only the file is checked unless WithEnvOverrides is given. The result line
// and the returned error say which.
func ValidateConfigFile(w io.Writer, path string, cfg Validator, opts ...ValidateConfigFileOpt) error {
	var options validateConfigFileOptions
	for _, opt := range opts {
		opt(&options)
	}

	err := decodeConfigFile(path, cfg, options.envPrefix)
	var decodeFindings *configcheck.Error
	if err != nil && !errors.As(err, &decodeFindings) {
		return err
	}

	findings := configcheck.NewCollector()
	findings.Add(err)
	findings.Add(cfg.Validate())

	var referenceErr error
	if rc, ok := cfg.(ReferenceChecker); ok {
		referenceErr = rc.CheckReferences(findings)
	}

	report := newValidationReport(path, findings.Findings(), referenceErr)
	if report.problems == 0 {
		_, err = fmt.Fprintf(w, "%s is valid (%s)\n", path, options.mode())

		return err
	}

	if _, err = io.WriteString(w, report.text); err != nil {
		return err
	}

	return fmt.Errorf("%s: %d configuration problem(s) found (%s)", path, report.problems, options.mode())
}

type validationReport struct {
	text     string
	problems int
}

// newValidationReport lays out the problems of the config file at path, one
// section per file: the config's own findings, then those referenceErr holds
// for each file the config refers to. Anything else in referenceErr is a
// problem with the config.
func newValidationReport(path string, findings []configcheck.Finding, referenceErr error) validationReport {
	var files []string
	lines := map[string][]string{}
	add := func(file string, problems []configcheck.Finding) {
		if _, seen := lines[file]; !seen && len(problems) > 0 {
			files = append(files, file)
		}
		for _, f := range problems {
			lines[file] = append(lines[file], f.String())
		}
	}

	add(path, findings)
	for _, leaf := range errorLeaves(referenceErr) {
		file, problems := path, leaf
		if fileErr, ok := errors.AsType[*FileError](leaf); ok {
			file, problems = fileErr.Path, fileErr.Err
		}

		inFile := configcheck.NewCollector()
		inFile.Add(problems)
		add(file, inFile.Findings())
	}

	var text strings.Builder
	var problems int
	for _, file := range files {
		text.WriteString(file + ":\n")
		for _, line := range lines[file] {
			text.WriteString("  " + line + "\n")
			problems++
		}
	}

	return validationReport{text: text.String(), problems: problems}
}

// errorLeaves returns the errors joined into err, or err itself. A nil err has
// none.
func errorLeaves(err error) []error {
	if err == nil {
		return nil
	}
	joined, ok := err.(interface{ Unwrap() []error }) //nolint:errorlint // inspects err's own Unwrap, not the chain
	if !ok {
		return []error{err}
	}

	var leaves []error
	for _, e := range joined.Unwrap() {
		leaves = append(leaves, errorLeaves(e)...)
	}

	return leaves
}
