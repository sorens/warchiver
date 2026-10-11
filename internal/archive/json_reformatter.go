package archive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	walog "github.com/sorens/warchiver/internal/log"
)

type NotJson struct {
	err error
}

func (e *NotJson) Error() string {
	return fmt.Sprintf("invalid JSON data: %s", e.err)
}

func IsNotJson(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*NotJson)
	return ok
}

func ReformatJsonFile(ctx context.Context, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	tmp, err := os.CreateTemp("", "wa.*.json")
	if err != nil {
		return fmt.Errorf("failed to create tmp file: %w", err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	err = ReformatJson(ctx, file, tmp)
	if IsNotJson(err) {
		// the data is not json, return early, no error
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to reformat JSON: %w", err)
	}

	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("failed to replace file: %w", err)
	}

	return nil
}

func ReformatJson(ctx context.Context, r io.Reader, w io.Writer) error {
	var data any
	if err := decodeSingleJSON(r, &data); err != nil {
		walog.Ctx(ctx).Debug("failed to decode JSON: %s", err)
		return &NotJson{err: err}
	}
	return encodeIndentedJSON(data, w)
}

func ReformatJsonToBuffer(ctx context.Context, r io.Reader) (*bytes.Buffer, error) {
	var data any
	if err := decodeSingleJSON(r, &data); err != nil {
		walog.Ctx(ctx).Debug("failed to decode JSON: %s", err)
		return nil, &NotJson{err: err}
	}

	var buf bytes.Buffer
	if err := encodeIndentedJSON(data, &buf); err != nil {
		return nil, err
	}
	return &buf, nil
}

func decodeSingleJSON(r io.Reader, data *any) error {
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(data); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return fmt.Errorf("trailing content after JSON value: %w", err)
	}
	return nil
}

func encodeIndentedJSON(data any, w io.Writer) error {
	data = canonicalize(data)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode indented JSON: %w", err)
	}
	return nil
}

// canonicalize walks decoded JSON; arrays preserve order, object keys sort via encoding/json.
func canonicalize(v any) any {
	switch val := v.(type) {
	case map[string]any:
		for k, child := range val {
			val[k] = canonicalize(child)
		}
		return val
	case []any:
		for i, child := range val {
			val[i] = canonicalize(child)
		}
		return val
	default:
		return v
	}
}
