package op

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
)

// Normalizable is an interface for entities that can be normalized.
// Useful for example for trimming strings, add custom fields, etc.
// Can also raise an error if the entity is not valid.
type Normalizable interface {
	Normalize() error // Normalizes the entity.
}

var ReadOptions = readOptions{
	DisallowUnknownFields: true,
	MaxBodySize:           maxBodySize,
}

// ReadJSON reads the request body as JSON.
// Can be used independantly from op! framework.
// Customisable by modifying ReadOptions.
func ReadJSON[B any](input io.ReadCloser) (B, error) {
	return readJSON[B](input, ReadOptions)
}

// readJSON reads the request body as JSON.
// Can be used independantly from framework using ReadJSON,
// or as a method of Context.
// It will also read strings.
func readJSON[B any](input io.ReadCloser, options readOptions) (B, error) {
	var body *B
	switch any(new(B)).(type) {
	case *string:
		// Read the request body.
		readBody, err := io.ReadAll(input)
		if err != nil {
			return *new(B), fmt.Errorf("cannot read request body: %w", err)
		}
		// c.body = (*B)(unsafe.Pointer(&body))
		s := string(readBody)
		body = any(&s).(*B)
		slog.Info("Read body", "body", *body)
	default:
		// Deserialize the request body.
		dec := json.NewDecoder(input)
		if true {
			dec.DisallowUnknownFields()
		}
		err := dec.Decode(&body)
		if err != nil {
			return *new(B), fmt.Errorf("cannot decode request body: %w", err)
		}
		slog.Info("Decoded body", "body", *body)

		// Validation
		err = validate(*body)
		if err != nil {
			return *body, fmt.Errorf("cannot validate request body: %w", err)
		}
	}

	// Normalize input if possible.
	if normalizableBody, ok := any(body).(Normalizable); ok {
		err := normalizableBody.Normalize()
		if err != nil {
			return *body, fmt.Errorf("cannot normalize request body: %w", err)
		}
		body, ok = any(normalizableBody).(*B)
		if !ok {
			return *body, fmt.Errorf("cannot retype request body: %w",
				fmt.Errorf("normalized body is not of type %T but should be", *new(B)))
		}

		slog.Info("Normalized body", "body", *body)
	}

	return *body, nil
}
