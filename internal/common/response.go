package common

type APIResponse struct {
	Status  string      `json:"status"`          // Success or error status
	Message string      `json:"message"`         // A message to describe the result
	Data    interface{} `json:"data,omitempty"`  // The actual data (if any), it can be any type
	Error   string      `json:"error,omitempty"` // Error details (if any)
}
