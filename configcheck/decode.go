package configcheck

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/go-viper/mapstructure/v2"

	"github.com/dioad/util"
)

// UnknownKeyMessage is the Message of a finding Decode reports for a key
// that matches no field.
const UnknownKeyMessage = "unknown key"

// mapstructure reports every unknown key in one struct as a single error
// whose message starts with this text, followed by the keys.
const invalidKeysPrefix = "has invalid keys: "

// DecodeHook returns the decode hook used to load configuration: it decodes
// util.MaskedString, time.Duration, net.IP and net.IPNet values from
// strings. Use it for any decode of configuration that must accept exactly
// what the cli loader accepts.
func DecodeHook() mapstructure.DecodeHookFunc {
	return mapstructure.ComposeDecodeHookFunc(
		util.MaskedStringDecodeHook,
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToIPHookFunc(),
		mapstructure.StringToIPNetHookFunc(),
	)
}

// Decode decodes input, typically a map[string]any read from a
// configuration file, into out, which must be a non-nil pointer. It decodes
// as the cli loader does, with DecodeHook and mapstructure's weakly typed
// input (so a number decodes into a string field, as viper allows), and
// additionally rejects keys that match no field.
//
// Every problem is reported, as an *Error whose findings are sorted by path
// and whose paths are relative to input: "tunnels[0].tls-server-config" for
// an unknown key, "routes[dmarc].max-size" inside a map. Fields that decode
// cleanly are set even when others fail. Any other error means out could not
// be decoded into at all, which is a programming error rather than a
// configuration problem.
func Decode(input, out any) error {
	outType := reflect.TypeOf(out)
	if outType == nil || outType.Kind() != reflect.Pointer {
		return fmt.Errorf("configcheck: decode target must be a non-nil pointer, got %T", out)
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook:  DecodeHook(),
		ErrorUnused: true,
		// viper's Unmarshal, which the cli loader uses, decodes weakly.
		WeaklyTypedInput: true,
		Result:           out,
	})
	if err != nil {
		return fmt.Errorf("configcheck: %w", err)
	}

	err = decoder.Decode(input)
	if err == nil {
		return nil
	}

	// mapstructure names an unknown key at the root after the target type
	// rather than with an empty path.
	rootName := outType.Elem().String()
	var findings []Finding
	collectFindings(err, rootName, &findings)
	slices.SortStableFunc(findings, func(a, b Finding) int {
		return cmp.Or(cmp.Compare(a.Path, b.Path), cmp.Compare(a.Message, b.Message))
	})
	return &Error{Findings: findings}
}

// collectFindings walks mapstructure's error tree: a wrapper around a join
// of per-field errors, each a *mapstructure.DecodeError naming its field.
// It must not use errors.As on the whole tree, which would stop at the
// first DecodeError.
func collectFindings(err error, rootName string, findings *[]Finding) {
	decodeErr, ok := err.(*mapstructure.DecodeError)
	if !ok {
		switch wrapped := err.(type) {
		case interface{ Unwrap() []error }:
			for _, e := range wrapped.Unwrap() {
				collectFindings(e, rootName, findings)
			}
		case interface{ Unwrap() error }:
			collectFindings(wrapped.Unwrap(), rootName, findings)
		default:
			*findings = append(*findings, Finding{Message: err.Error()})
		}
		return
	}

	path := decodeErr.Name()
	if path == rootName {
		path = ""
	}
	message := decodeErr.Unwrap().Error()

	keys, isUnknownKeys := strings.CutPrefix(message, invalidKeysPrefix)
	if !isUnknownKeys {
		*findings = append(*findings, Finding{Path: path, Message: message})
		return
	}
	for key := range strings.SplitSeq(keys, ", ") {
		*findings = append(*findings, Finding{Path: JoinPath(path, key), Message: UnknownKeyMessage})
	}
}
