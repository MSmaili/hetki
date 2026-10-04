package report

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode/utf8"
)

// Decode reads exactly one bounded JSON object without unknown/duplicate keys.
// Callers must bound stdin lifetime; contexts cannot interrupt arbitrary readers.
func Decode(reader io.Reader) (Request, error) {
	data, err := io.ReadAll(io.LimitReader(reader, MaxRequestBytes+1))
	if err != nil {
		return Request{}, fmt.Errorf("read report input: %w", err)
	}
	if len(data) > MaxRequestBytes {
		return Request{}, errors.New("report input exceeds byte limit")
	}
	var request Request
	if err := decode(data, &request); err != nil {
		return Request{}, fmt.Errorf("decode report: %w", err)
	}
	if err := request.validate(); err != nil {
		return Request{}, err
	}
	return request, nil
}

func decode(data []byte, target any) error {
	if !utf8.Valid(data) {
		return errors.New("JSON must be valid UTF-8")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := jsonValue(d, 0, reflect.TypeOf(target)); err != nil {
		return err
	}
	if _, err := d.Token(); !errors.Is(err, io.EOF) {
		return errors.New("expected exactly one JSON value")
	}
	d = json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	// Do not echo decoder errors, which can contain arbitrary incoming values.
	if err := d.Decode(target); err != nil {
		return errors.New("invalid JSON fields or types")
	}
	return nil
}

func jsonValue(d *json.Decoder, depth int, shape reflect.Type) error {
	if depth > 16 {
		return errors.New("JSON exceeds nesting limit")
	}
	token, err := d.Token()
	if err != nil {
		return errors.New("invalid JSON")
	}
	if token == nil {
		return errors.New("JSON null is not supported")
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	if delim != '{' && delim != '[' {
		return errors.New("invalid JSON delimiter")
	}
	for shape != nil && shape.Kind() == reflect.Pointer {
		shape = shape.Elem()
	}
	fields := make(map[string]reflect.Type)
	required := make(map[string]bool)
	if delim == '{' && shape != nil && shape.Kind() == reflect.Struct {
		for i := 0; i < shape.NumField(); i++ {
			field := shape.Field(i)
			tag := field.Tag.Get("json")
			name := strings.Split(tag, ",")[0]
			if name != "" && name != "-" {
				fields[name] = field.Type
				required[name] = !strings.Contains(tag, ",omitempty") && !strings.Contains(tag, ",omitzero")
			}
		}
	}
	var element reflect.Type
	if delim == '[' && shape != nil && shape.Kind() == reflect.Slice {
		element = shape.Elem()
	}
	seen := make(map[string]bool)
	for d.More() {
		child := element
		if delim == '{' {
			key, err := d.Token()
			if err != nil {
				return errors.New("invalid JSON field")
			}
			name, ok := key.(string)
			if !ok || seen[name] {
				return errors.New("duplicate or invalid JSON field")
			}
			var known bool
			child, known = fields[name]
			if !known {
				return errors.New("unknown JSON field (names are case-sensitive)")
			}
			seen[name] = true
		}
		if err := jsonValue(d, depth+1, child); err != nil {
			return err
		}
	}
	end, err := d.Token()
	if err != nil || (delim == '{' && end != json.Delim('}')) || (delim == '[' && end != json.Delim(']')) {
		return errors.New("invalid JSON delimiter")
	}
	for name, mandatory := range required {
		if mandatory && !seen[name] {
			return errors.New("missing JSON field")
		}
	}
	return nil
}
