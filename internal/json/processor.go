package json_processor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
)

type JSONProcessor struct {
	targetKey string
	keys      []string
	values    map[string]bool
}

func NewJSONProcessor(keys []string) *JSONProcessor {
	return &JSONProcessor{
		keys:   keys,
		values: map[string]bool{},
	}
}

func (jp *JSONProcessor) FetchAndProcessFromURL(url, filename string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("HTTP request failed: %s", resp.Status)
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return jp.extractDecoder(json.NewDecoder(io.TeeReader(resp.Body, file)))
}

func (jp *JSONProcessor) FetchAndProcess(reader io.ReadCloser, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	dec := json.NewDecoder(io.TeeReader(reader, file))
	return jp.extractDecoder(dec)
}

func (jp *JSONProcessor) extractStream(r io.Reader) error {
	return jp.extractDecoder(json.NewDecoder(r))
}

func (jp *JSONProcessor) extractDecoder(dec *json.Decoder) error {
	var walk func(string) error
	walk = func(key string) error {
		t, err := dec.Token()
		if err != nil {
			return err
		}
		switch v := t.(type) {
		case json.Delim:
			if v == '{' {
				for dec.More() {
					k, err := dec.Token()
					if err != nil {
						return err
					}
					ks := k.(string)
					if err := walk(ks); err != nil {
						return err
					}
				}
				_, err = dec.Token()
				return err
			}
			if v == '[' {
				for dec.More() {
					if err := walk(""); err != nil {
						return err
					}
				}
				_, err = dec.Token()
				return err
			}
		case string:
			for _, target := range jp.keys {
				if key == target && v != "" {
					jp.values[v] = true
				}
			}
		}
		return nil
	}
	return walk("")
}

func (jp *JSONProcessor) extract(data any) {
	switch v := data.(type) {
	case map[string]any:
		for k, val := range v {
			// is this key one we're looking for?
			for _, targetKey := range jp.keys {
				if k == targetKey {
					if str, ok := val.(string); ok && str != "" {
						jp.values[str] = true
					}
				}
			}
			// recurse into nested structures
			jp.extract(val)
		}
	case []any:
		for _, item := range v {
			jp.extract(item)
		}
	}
}

func (jp *JSONProcessor) Values() []string {
	results := []string{}
	for key := range jp.values {
		results = append(results, key)
	}
	sort.Strings(results)
	return results
}
