package stepconf

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/bitrise-io/go-utils/colorstring"
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

const rangeMinimumGroupName = "min"
const rangeMaximumGroupName = "max"
const rangeMinBracketGroupName = "minbr"
const rangeMaxBracketGroupName = "maxbr"
const rangeRegex = `range(?P<` + rangeMinBracketGroupName + `>\[|\])(?P<` + rangeMinimumGroupName + `>.*?)\.\.(?P<` + rangeMaximumGroupName + `>.*?)(?P<` + rangeMaxBracketGroupName + `>\[|\])`

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

// Print the name of the struct with Title case in blue color with followed by a newline,
// then print all fields formatted as '- field name: field value` separated by newline.
func Print(config interface{}) {
	fmt.Printf(toString(config))
}

func valueString(v reflect.Value) string {
	if v.Kind() != reflect.Ptr {
		return fmt.Sprintf("%v", v.Interface())
	}

	if !v.IsNil() {
		return fmt.Sprintf("%v", v.Elem().Interface())
	}

	return ""
}

// returns the name of the struct with Title case in blue color followed by a newline,
// then print all fields formatted as '- field name: field value` separated by newline.
func toString(config interface{}) string {
	v := reflect.ValueOf(config)
	t := reflect.TypeOf(config)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	str := fmt.Sprintf(colorstring.Bluef("%s:\n", strings.Title(t.Name())))
	for i := 0; i < t.NumField(); i++ {
		str += fmt.Sprintf("- %s: %s\n", t.Field(i).Name, valueString(v.Field(i)))
	}

	return str
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

		errorString += fmt.Sprintf("\n\n%s", toString(conf))
		return errors.New(errorString)
	}

	return nil
}

func setField(field reflect.Value, value, constraint string) error {
	if err := validateConstraint(value, constraint); err != nil {
		return err
	}

	if value == "" {
		return nil
	}

	if field.Kind() == reflect.Ptr {
		// If field is a pointer type, then set its value to be a pointer to a new zero value, matching field underlying type.
		var dePtrdType = field.Type().Elem()     // get the type field can point to
		var newPtrType = reflect.New(dePtrdType) // create new ptr address for type with non-nil zero value
		field.Set(newPtrType)                    // assign value to pointer
		field = field.Elem()
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
	case reflect.Float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return errors.New("can't convert to float")
		}
		field.SetFloat(f)
	case reflect.Slice:
		field.Set(reflect.ValueOf(strings.Split(value, "|")))
	default:
		return fmt.Errorf("type is not supported (%s)", field.Kind())
	}
	return nil
}

func validateConstraint(value, constraint string) error {
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
	case regexp.MustCompile(rangeRegex).FindString(constraint):
		if err := ValidateRangeFields(value, constraint); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid constraint (%s)", constraint)
	}
	return nil
}

//ValidateRangeFields validates if the given range is proper. Ranges are optional, empty values are valid.
func ValidateRangeFields(valueStr, constraint string) error {
	if valueStr == "" {
		return nil
	}
	constraintMin, constraintMax, constraintMinBr, constraintMaxBr, err := GetRangeValues(constraint)
	if err != nil {
		return err
	}
	min, err := parseValueStr(constraintMin)
	if err != nil {
		return fmt.Errorf("failed to parse min value %s: %s", constraintMin, err)
	}
	max, err := parseValueStr(constraintMax)
	if err != nil {
		return fmt.Errorf("failed to parse max value %s: %s", constraintMax, err)
	}
	value, err := parseValueStr(valueStr)
	if err != nil {
		return fmt.Errorf("failed to parse value %s: %s", valueStr, err)
	}
	isMinInclusiveBool, err := isMinInclusive(constraintMinBr)
	if err != nil {
		return err
	}
	isMaxInclusiveBool, err := isMaxInclusive(constraintMaxBr)
	if err != nil {
		return err
	}

	if err := validateRangeFieldValues(min, max, isMinInclusiveBool, isMaxInclusiveBool, value); err != nil {
		return err
	}
	if err := validateRangeFieldTypes(min, max, value); err != nil {
		return err
	}

	return nil
}

func isMinInclusive(bracket string) (bool, error) {
	switch bracket {
	case "[":
		return true, nil
	case "]":
		return false, nil
	default:
		return false, fmt.Errorf("invalid string found for bracket: %s", bracket)
	}
}

func isMaxInclusive(bracket string) (bool, error) {
	switch bracket {
	case "[":
		return false, nil
	case "]":
		return true, nil
	default:
		return false, fmt.Errorf("invalid string found for bracket: %s", bracket)
	}
}

func validateRangeFieldValues(min interface{}, max interface{}, minInclusive bool, maxInclusive bool, value interface{}) error {
	if value == nil {
		return fmt.Errorf("value is not present")
	}
	var err error
	var valueFloat float64
	if valueFloat, err = getFloatValue(value); err != nil {
		return err
	}

	var minErr error
	var minFloat float64
	if min != nil {
		if minFloat, err = getFloatValue(min); err != nil {
			return err
		}
		minErr = validateRangeMinFieldValue(minFloat, valueFloat, minInclusive)
	}

	var maxErr error
	var maxFloat float64
	if max != nil {
		var err error
		if maxFloat, err = getFloatValue(max); err != nil {
			return err
		}
		maxErr = validateRangeMaxFieldValue(maxFloat, valueFloat, maxInclusive)
	}

	if min != nil && max != nil {
		if minFloat > maxFloat {
			return fmt.Errorf("constraint logic is wrong, minimum value %f is bigger than maximum %f", minFloat, maxFloat)
		}
		if minFloat == maxFloat {
			return fmt.Errorf("minimum value %f is equal to maximum %f, for this case use optional value", minFloat, maxFloat)
		}
	}

	if min == nil {
		return maxErr
	} else if max == nil {
		return minErr
	}
	if minErr != nil || maxErr != nil {
		return fmt.Errorf("value %f is out of range %f-%f", value, minFloat, maxFloat)
	}
	return nil
}

func validateRangeFieldTypes(min interface{}, max interface{}, value interface{}) error {
	if value == nil {
		return fmt.Errorf("value cannot be nil")
	}
	var minType string
	var maxType string
	var valueType string
	var err error

	if valueType, err = getTypeOf(value); err != nil {
		return err
	}
	if min != nil {
		if minType, err = getTypeOf(min); err != nil {
			return err
		}
	}
	if max != nil {
		if maxType, err = getTypeOf(max); err != nil {
			return err
		}
	}

	if maxType != "" && minType != "" && !hasSameContent(minType, maxType, valueType) {
		return fmt.Errorf("invalid constraint and value combination, minimum is %s, maximum is %s, value is %s, but they should be the same", minType, maxType, valueType)
	}

	if maxType != "" && !hasSameContent(maxType, valueType) {
		return fmt.Errorf("invalid constraint and value combination, maximum is %s, value is %s, but they should be the same", maxType, valueType)
	}

	if minType != "" && !hasSameContent(minType, valueType) {
		return fmt.Errorf("invalid constraint and value combination, minimum is %s, value is %s, but they should be the same", minType, valueType)
	}
	return nil
}

func hasSameContent(strs ...string) bool {
	length := len(strs)
	if length == 1 {
		return true
	}
	firstItem := strs[0]
	for i := 1; i < length; i++ {
		if strings.Compare(firstItem, strs[i]) != 0 {
			return false
		}
	}
	return true
}

func getTypeOf(v interface{}) (string, error) {
	switch v.(type) {
	case int64:
		return "int64", nil
	case float64:
		return "float64", nil
	default:
		return "unknown", fmt.Errorf("could not find type for %v", v)
	}
}

func parseValueStr(value string) (interface{}, error) {
	var err error
	var parsedInt int64
	var parsedFloat float64
	if parsedInt, err = strconv.ParseInt(value, 10, 64); err != nil {
		// Could be float
		if parsedFloat, err = strconv.ParseFloat(value, 64); err != nil {
			// It is invalid.
			return nil, fmt.Errorf("value %s is could not be parsed", value)
		}
		return parsedFloat, nil
	}
	return parsedInt, nil
}

func getFloatValue(value interface{}) (float64, error) {
	switch i := value.(type) {
	case int64:
		return float64(i), nil
	case float64:
		return i, nil
	case string:
		var parsedValue interface{}
		var err error
		if parsedValue, err = parseValueStr(i); err != nil {
			return 0, err
		}
		return getFloatValue(parsedValue)
	default:
		return 0, fmt.Errorf("not supported type %T", value)
	}
}

func validateRangeMinFieldValue(min float64, value float64, inclusive bool) error {
	if inclusive && min > value {
		return fmt.Errorf("value %f is out of range, less than minimum %f", value, min)
	} else if !inclusive && min >= value {
		return fmt.Errorf("value %f is out of range, greater or equal than maximum %f", value, min)
	}
	return nil
}

func validateRangeMaxFieldValue(max float64, value float64, inclusive bool) error {
	if inclusive && max < value {
		return fmt.Errorf("value %f is out of range, greater than maximum %f", value, max)

	} else if !inclusive && max <= value {
		return fmt.Errorf("value %f is out of range, greater or equal than maximum %f", value, max)
	}
	return nil
}

// GetRangeValues reads up the given range constraint and returns the values, or an error if the constraint is malformed or could not be parsed.
func GetRangeValues(value string) (min string, max string, minBracket string, maxBracket string, err error) {
	regex := regexp.MustCompile(rangeRegex)
	groups := regex.FindStringSubmatch(value)
	if len(groups) < 1 {
		return "", "", "", "", fmt.Errorf("value in value options is malformed (%s)", value)
	}

	groupMap := getRegexGroupMap(groups, regex)
	minStr := groupMap[rangeMinimumGroupName]
	maxStr := groupMap[rangeMaximumGroupName]
	minBr := groupMap[rangeMinBracketGroupName]
	maxBr := groupMap[rangeMaxBracketGroupName]
	if minStr == "" && maxStr == "" {
		return "", "", "", "", fmt.Errorf("constraint contains no limits")
	}
	return minStr, maxStr, minBr, maxBr, nil
}

func getRegexGroupMap(groups []string, regex *regexp.Regexp) map[string]string {
	result := make(map[string]string)
	for i, value := range regex.SubexpNames() {
		if i != 0 && value != "" {
			result[value] = groups[i]
		}
	}
	return result
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
