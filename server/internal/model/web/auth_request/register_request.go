package auth_request

type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name,omitempty" validate:"omitempty,min=3,max=50"`
}
