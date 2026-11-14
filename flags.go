package cli

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/spf13/pflag"
	"github.com/urfave/sflags"
	"github.com/urfave/sflags/gen/gpflag"
)

const (
	// configKeyAnnotation is the pflag annotation holding the config key a
	// flag sets: the dotted path of the config struct field's mapstructure
	// names, which is where the config is decoded from.
	configKeyAnnotation = "dioad-cli-config-key"
	// aliasOfAnnotation marks a deprecated alias flag, and names the flag
	// it is an alias of.
	aliasOfAnnotation = "dioad-cli-alias-of"
)

// registerConfigFlags adds a flag to dst for every field of cfg, a pointer to
// the config struct.
//
// sflags names a flag from the Go field path, but the config is decoded from
// the fields' mapstructure keys, and the two differ for a renamed field, a
// named struct squashed into its parent, or a nested block. A flag named after
// the wrong key sets a value nothing decodes. So each flag is named from the
// field's mapstructure key path ("log-level" for log.level), and records that
// path in an annotation so InitConfig binds it to the right key.
//
// A field whose own flag tag, or an ancestor's, names the flag keeps that name:
// the author chose it. A flag whose name changed is also registered under its
// previous sflags name, hidden and deprecated, so existing command lines keep
// working.
func registerConfigFlags(dst *pflag.FlagSet, cfg any, opts ...sflags.OptFunc) error {
	generated := pflag.NewFlagSet("generated", pflag.ContinueOnError)
	if err := gpflag.ParseTo(cfg, generated, opts...); err != nil {
		return err
	}

	keys := map[string]configFlag{}
	walkConfigFlags(reflect.TypeOf(cfg), "", nil, false, generated, opts, keys)

	var err error
	generated.VisitAll(func(f *pflag.Flag) {
		if err != nil {
			return
		}
		err = addFlag(dst, f, keys[f.Name])
	})

	return err
}

// configFlag is what walkConfigFlags learns about one generated flag.
type configFlag struct {
	key  string
	name string
}

// walkConfigFlags records, for each flag in generated that belongs to a field
// of t, the config key it sets and the name it should have.
func walkConfigFlags(t reflect.Type, flagPrefix string, keyPath []string, explicit bool, generated *pflag.FlagSet, opts []sflags.OptFunc, out map[string]configFlag) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}

	for i := range t.NumField() {
		field := t.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}

		if name, _, _ := strings.Cut(field.Tag.Get("flag"), ","); name == "-" {
			continue
		}
		var flagName string
		if !field.Anonymous {
			var ok bool
			if flagName, ok = sflagsName(field, flagPrefix, opts); !ok {
				continue
			}
		}
		msName, squash := mapstructureName(field)
		fieldExplicit := explicit || hasExplicitFlagName(field)

		fieldKey := keyPath
		if !squash && (!field.Anonymous || msName != "") {
			if msName == "" {
				msName = strings.ToLower(field.Name)
			}
			fieldKey = append(append([]string{}, keyPath...), msName)
		}

		if flagName != "" && generated.Lookup(flagName) != nil {
			name := flagName
			if !fieldExplicit {
				name = strings.Join(fieldKey, "-")
			}
			out[flagName] = configFlag{key: strings.Join(fieldKey, "."), name: name}

			continue
		}

		nextPrefix := flagName + "-"
		if field.Anonymous {
			nextPrefix = flagPrefix
		}
		walkConfigFlags(field.Type, nextPrefix, fieldKey, fieldExplicit, generated, opts, out)
	}
}

// sflagsName returns the name sflags gives field's flag under prefix, by asking
// sflags itself so the two cannot disagree. It reports false when the field has
// no flag (`flag:"-"`).
func sflagsName(field reflect.StructField, prefix string, opts []sflags.OptFunc) (string, bool) {
	probe := reflect.StructOf([]reflect.StructField{{
		Name: field.Name,
		Type: reflect.TypeFor[string](),
		Tag:  field.Tag,
	}})
	flags, err := sflags.ParseStruct(reflect.New(probe).Interface(), append(append([]sflags.OptFunc{}, opts...), sflags.Prefix(prefix))...)
	if err != nil || len(flags) != 1 {
		return "", false
	}

	return flags[0].Name, true
}

// mapstructureName returns field's mapstructure key and whether it is squashed.
func mapstructureName(field reflect.StructField) (string, bool) {
	parts := strings.Split(field.Tag.Get("mapstructure"), ",")
	if slices.Contains(parts[1:], "squash") {
		return parts[0], true
	}

	return parts[0], false
}

// hasExplicitFlagName reports whether field's flag tag gives the flag a name,
// as opposed to only options such as hidden.
func hasExplicitFlagName(field reflect.StructField) bool {
	name, _, _ := strings.Cut(field.Tag.Get("flag"), ",")

	return name != "" && name != "-"
}

// addFlag registers f on dst under the name info chose, with an alias under
// f's previous name if it changed.
func addFlag(dst *pflag.FlagSet, f *pflag.Flag, info configFlag) error {
	name := f.Name
	if info.name != "" && dst.Lookup(info.name) == nil {
		name = info.name
	}

	added := dst.VarPF(f.Value, name, f.Shorthand, f.Usage)
	added.NoOptDefVal = f.NoOptDefVal
	added.Hidden = f.Hidden
	added.Deprecated = f.Deprecated
	if info.key != "" {
		added.Annotations = map[string][]string{configKeyAnnotation: {info.key}}
	}

	if name == f.Name || dst.Lookup(f.Name) != nil {
		return nil
	}

	alias := dst.VarPF(f.Value, f.Name, "", f.Usage)
	alias.NoOptDefVal = f.NoOptDefVal
	alias.Hidden = true
	alias.Deprecated = fmt.Sprintf("use --%s", name)
	alias.Annotations = map[string][]string{aliasOfAnnotation: {name}}

	return nil
}
