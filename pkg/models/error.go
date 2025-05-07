package models

type APIError struct {
	APIMessage string `json:"message"`
	Errors     []struct {
		Message string `json:"message"`
		Field   string `json:"field"`
	} `json:"errors"`
}

// Implement the uhttp.ErrorResponse interface.
func (e *APIError) Message() string {
	if len(e.Errors) > 0 {
		msg := e.Errors[0].Message
		if e.Errors[0].Field != "" {
			msg += " (field: " + e.Errors[0].Field + ")"
		}
		return msg
	}
	if e.APIMessage != "" {
		return e.APIMessage
	}
	return "unknown error from Greenhouse API"
}
