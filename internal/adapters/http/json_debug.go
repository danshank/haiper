package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// JSONError represents detailed information about JSON parsing failures
type JSONError struct {
	Type        string `json:"type"`
	Message     string `json:"message"`
	Offset      int64  `json:"offset,omitempty"`
	RawInput    string `json:"raw_input,omitempty"`
	InputLength int    `json:"input_length"`
	Context     string `json:"context,omitempty"`
}

// DecodeJSONWithDebug safely decodes JSON with comprehensive error logging
func DecodeJSONWithDebug(r *http.Request, dst interface{}, maxBytes int64) error {
	// Limit request body size
	if maxBytes > 0 {
		r.Body = http.MaxBytesReader(nil, r.Body, maxBytes)
	}

	// Read the entire body for debugging purposes
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		jsonErr := JSONError{
			Type:        "read_error",
			Message:     "Failed to read request body",
			InputLength: 0,
		}
		logJSONError(jsonErr, err)
		return fmt.Errorf("failed to read request body: %w", err)
	}

	// Create a new reader from the bytes for actual decoding
	bodyReader := bytes.NewReader(bodyBytes)
	decoder := json.NewDecoder(bodyReader)
	
	// Configure decoder for strict parsing
	decoder.DisallowUnknownFields() // Optional: reject unknown fields
	
	err = decoder.Decode(dst)
	if err != nil {
		jsonErr := analyzeJSONError(err, bodyBytes)
		logJSONError(jsonErr, err)
		return createUserFriendlyError(jsonErr)
	}

	// Check if there's additional data after the JSON
	if decoder.More() {
		jsonErr := JSONError{
			Type:        "extra_data",
			Message:     "Request body contains extra data after JSON",
			RawInput:    truncateString(string(bodyBytes), 500),
			InputLength: len(bodyBytes),
		}
		logJSONError(jsonErr, errors.New("extra data after JSON"))
		return fmt.Errorf("request body contains extra data after JSON")
	}

	return nil
}

// analyzeJSONError extracts detailed information from JSON parsing errors
func analyzeJSONError(err error, input []byte) JSONError {
	jsonErr := JSONError{
		RawInput:    truncateString(string(input), 500),
		InputLength: len(input),
	}

	var syntaxError *json.SyntaxError
	var unmarshalTypeError *json.UnmarshalTypeError
	var invalidUnmarshalError *json.InvalidUnmarshalError
	var maxBytesError *http.MaxBytesError

	switch {
	case errors.As(err, &syntaxError):
		jsonErr.Type = "syntax_error"
		jsonErr.Offset = syntaxError.Offset
		jsonErr.Message = fmt.Sprintf("JSON syntax error at position %d", syntaxError.Offset)
		jsonErr.Context = extractErrorContext(input, syntaxError.Offset)

	case errors.As(err, &unmarshalTypeError):
		jsonErr.Type = "type_error"
		jsonErr.Offset = unmarshalTypeError.Offset
		jsonErr.Message = fmt.Sprintf("Cannot unmarshal %s into Go value of type %s (field: %s)", 
			unmarshalTypeError.Value, unmarshalTypeError.Type, unmarshalTypeError.Field)
		jsonErr.Context = extractErrorContext(input, unmarshalTypeError.Offset)

	case errors.As(err, &invalidUnmarshalError):
		jsonErr.Type = "invalid_unmarshal"
		jsonErr.Message = fmt.Sprintf("Invalid unmarshal target: %s", invalidUnmarshalError.Type)

	case errors.As(err, &maxBytesError):
		jsonErr.Type = "size_limit"
		jsonErr.Message = fmt.Sprintf("Request body too large (limit: %d bytes)", maxBytesError.Limit)

	case errors.Is(err, io.EOF):
		jsonErr.Type = "empty_body"
		jsonErr.Message = "Request body is empty"

	case errors.Is(err, io.ErrUnexpectedEOF):
		jsonErr.Type = "unexpected_eof"
		jsonErr.Message = "Request body contains incomplete JSON"

	default:
		if strings.Contains(err.Error(), "unknown field") {
			jsonErr.Type = "unknown_field"
			jsonErr.Message = err.Error()
		} else {
			jsonErr.Type = "unknown_error"
			jsonErr.Message = err.Error()
		}
	}

	return jsonErr
}

// extractErrorContext extracts surrounding text around the error position
func extractErrorContext(input []byte, offset int64) string {
	if offset < 0 || offset >= int64(len(input)) {
		return ""
	}

	start := int64(0)
	end := int64(len(input))

	// Extract 50 characters before and after the error position
	if offset > 50 {
		start = offset - 50
	}
	if offset+50 < int64(len(input)) {
		end = offset + 50
	}

	context := string(input[start:end])
	
	// Mark the error position with >>> <<<
	relativeOffset := offset - start
	if relativeOffset >= 0 && relativeOffset < int64(len(context)) {
		before := context[:relativeOffset]
		after := context[relativeOffset:]
		context = fmt.Sprintf("%s>>>ERROR_HERE<<<%s", before, after)
	}

	return context
}

// logJSONError logs detailed JSON error information
func logJSONError(jsonErr JSONError, originalErr error) {
	log.Printf("🚨 JSON Parse Error Details:")
	log.Printf("   Type: %s", jsonErr.Type)
	log.Printf("   Message: %s", jsonErr.Message)
	log.Printf("   Input Length: %d bytes", jsonErr.InputLength)
	
	if jsonErr.Offset > 0 {
		log.Printf("   Error Position: %d", jsonErr.Offset)
	}
	
	if jsonErr.Context != "" {
		log.Printf("   Context: %s", jsonErr.Context)
	}
	
	log.Printf("   Raw Input: %s", jsonErr.RawInput)
	log.Printf("   Original Error: %v", originalErr)
	log.Printf("   Expected Format: {\"hook_type\": \"PreToolUse|PostToolUse|...\", \"session_id\": \"uuid\", \"tool\": \"Bash|Edit|...\", ...}")
}

// createUserFriendlyError creates user-friendly error messages
func createUserFriendlyError(jsonErr JSONError) error {
	switch jsonErr.Type {
	case "syntax_error":
		return fmt.Errorf("invalid JSON format at position %d. Please check your JSON syntax", jsonErr.Offset)
	case "type_error":
		return fmt.Errorf("incorrect data type in JSON: %s", jsonErr.Message)
	case "unknown_field":
		return fmt.Errorf("unknown field in JSON: %s. Allowed fields: hook_type, session_id, tool, cwd, data", jsonErr.Message)
	case "empty_body":
		return fmt.Errorf("request body is empty. Expected JSON with at least {\"hook_type\": \"...\"}")
	case "size_limit":
		return fmt.Errorf("request body too large: %s", jsonErr.Message)
	default:
		return fmt.Errorf("JSON parsing failed: %s", jsonErr.Message)
	}
}

// truncateString truncates a string to maxLength characters
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength] + "... [truncated]"
}

// GetExpectedJSONFormat returns a sample of the expected JSON format
func GetExpectedJSONFormat(hookType string) string {
	examples := map[string]string{
		"PreToolUse": `{
  "hook_type": "PreToolUse",
  "session_id": "123e4567-e89b-12d3-a456-426614174000",
  "tool": "Bash",
  "cwd": "/path/to/directory",
  "data": {...}
}`,
		"PostToolUse": `{
  "hook_type": "PostToolUse", 
  "session_id": "123e4567-e89b-12d3-a456-426614174000",
  "tool": "Bash",
  "success": true,
  "data": {...}
}`,
		"Notification": `{
  "hook_type": "Notification",
  "session_id": "123e4567-e89b-12d3-a456-426614174000",
  "message": "Claude needs attention"
}`,
	}

	if example, exists := examples[hookType]; exists {
		return example
	}

	return `{
  "hook_type": "PreToolUse|PostToolUse|Notification|UserPromptSubmit|Stop|SubagentStop|PreCompact",
  "session_id": "optional-uuid",
  "tool": "optional-tool-name",
  "data": "optional-data"
}`
}