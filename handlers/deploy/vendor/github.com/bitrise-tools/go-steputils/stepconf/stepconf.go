package stepconf

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/parseutil"
)

// ErrNotStructPtr indicates a type is not a pointer to a struct.
var ErrNotStructPtr = errors.New("must be a pointer to a struct")

// ParseError occurs when a struct field cannot be set.
type ParseError struct {
	Field string
	Value string
	Err   error
}

// Error implements builtin errors.Error.
func (e *ParseError) Error() string {
	segments := []string{e.Field}
	if e.Value != "" {
		segments = append(segments, e.Value)
	}
	segments = append(segments, e.Err.Error())
	return strings.Join(segments, ": ")
}

// Secret variables are not shown in the printed output.
type Secret string

const secret = "*****"

// String implements fmt.Stringer.String.
// When a Secret is printed, it's masking the underlying string with asterisks.
func (s Secret) String() string {
	if s == "" {
		return ""
	}
	return secret
}

// Print the name of the struct in blue color followed by a newline,
// then print all fields formatted as '- field name: field value` separated by newline.
func Print(config interface{}) {
	v := reflect.ValueOf(config)
	t := reflect.TypeOf(config)

	log.Infof("%s:", t.Name())
	for i := 0; i < t.NumField(); i++ {
		fmt.Printf("- %s: %v\n", t.Field(i).Name, v.Field(i).Interface())
	}
}

// parseTag splits a struct field's env tag into its name and option.
func parseTag(tag string) (string, string) {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx], tag[idx+1:]
	}
	return tag, ""
}

// Parse populates a struct with the retrieved values from environment variables
// described by struct tags and applies the defined validations.
func Parse(conf interface{}) error {
	c := reflect.ValueOf(conf)
	if c.Kind() != reflect.Ptr {
		return ErrNotStructPtr
	}
	c = c.Elem()
	if c.Kind() != reflect.Struct {
		return ErrNotStructPtr
	}
	t := c.Type()

	var errs []*ParseError
	for i := 0; i < c.NumField(); i++ {
		tag, ok := t.Field(i).Tag.Lookup("env")
		if !ok {
			continue
		}
		key, constraint := parseTag(tag)
		value := os.Getenv(key)

		if err := setField(c.Field(i), value, constraint); err != nil {
			errs = append(errs, &ParseError{t.Field(i).Name, value, err})
		}
	}
	if len(errs) > 0 {
		errorString := "failed to parse config:"
		for _, err := range errs {
			errorString += fmt.Sprintf("\n- %s", err)
		}
		return errors.New(errorString)
	}

	return nil
}

func setField(field reflect.Value, value, constraint string) error {
	switch constraint {
	case "":
		break
	case "required":
		if value == "" {
			return errors.New("required variable is not present")
		}
	case "file", "dir":
		if err := checkPath(value, constraint == "dir"); err != nil {
			return err
		}
	// TODO: use FindStringSubmatch to distinguish no match and match for empty string.
	case regexp.MustCompile(`^opt\[.*\]$`).FindString(constraint):
		if !contains(value, constraint) {
			// TODO: print only the value options, not the whole string.
			return fmt.Errorf("value is not in value options (%s)", constraint)
		}
	default:
		return fmt.Errorf("invalid constraint (%s)", constraint)
	}

	if value == "" {
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Bool:
		b, err := parseutil.ParseBool(value)
		if err != nil {
			return errors.New("can't convert to bool")
		}
		field.SetBool(b)
	case reflect.Int:
		n, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return errors.New("can't convert to int")
		}
		field.SetInt(n)
	case reflect.Slice:
		field.Set(reflect.ValueOf(strings.Split(value, "|")))
	default:
		return fmt.Errorf("type is not supported (%s)", field.Kind())
	}
	return nil
}

func checkPath(path string, dir bool) error {
	file, err := os.Stat(path)
	if err != nil {
		// TODO: check case when file exist but os.Stat fails.
		return os.ErrNotExist
	}
	if dir && !file.IsDir() {
		return errors.New("not a directory")
	}
	return nil
}

// contains reports whether s is within the value options, where value options
// are parsed from opt, which format's is opt[item1,item2,item3]. If an option
// contains commas, it should be single quoted (eg. opt[item1,'item2,item3']).
func contains(s, opt string) bool {
	opt = strings.TrimSuffix(strings.TrimPrefix(opt, "opt["), "]")
	var valueOpts []string
	if strings.Contains(opt, "'") {
		// The single quotes separate the options with comma and without comma
		// Eg. "a,b,'c,d',e" will results "a,b," "c,d" and ",e" strings.
		for _, s := range strings.Split(opt, "'") {
			switch {
			case s == "," || s == "":
			case !strings.HasPrefix(s, ",") && !strings.HasSuffix(s, ","):
				// If a string doesn't starts nor ends with a comma it means it's an option which
				// contains comma, so we just append it to valueOpts as it is. Eg. "c,d" from above.
				valueOpts = append(valueOpts, s)
			default:
				// If a string starts or ends with comma it means that it contains options without comma.
				// So we split the string at commas to get the options. Eg. "a,b," and ",e" from above.
				valueOpts = append(valueOpts, strings.Split(strings.Trim(s, ","), ",")...)
			}
		}
	} else {
		valueOpts = strings.Split(opt, ",")
	}
	for _, valOpt := range valueOpts {
		if valOpt == s {
			return true
		}
	}
	return false
}
