package auth_response

type AuthTokens struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"-"`
	RefreshToken string       `json:"-"`
}
