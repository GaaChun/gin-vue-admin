package request

// ReserveMasterCodeInput allocates a suggestion without preventing manual code edits.
type ReserveMasterCodeInput struct {
	Resource string `json:"resource" binding:"required,oneof=brands categories materials"`
}
