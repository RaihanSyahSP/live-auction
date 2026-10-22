package auction_request

type SpecRequest struct {
	Label string `json:"label" validate:"required"`
	Value string `json:"value" validate:"required"`
}

type CreateAuctionRequest struct {
	Title           string        `json:"title" validate:"required,min=3,max=120"`
	Subtitle        string        `json:"subtitle" validate:"omitempty,max=120"`
	Description     string        `json:"description" validate:"required,min=10"`
	Condition       string        `json:"condition" validate:"required"`
	Includes        []string      `json:"includes" validate:"omitempty,dive,min=1"`
	Specs           []SpecRequest `json:"specs" validate:"omitempty,dive"`
	ImageURLs       []string      `json:"image_urls" validate:"omitempty,dive,url"`
	StartingPrice   int64         `json:"starting_price" validate:"required,gt=0"`
	MinIncrement    int64         `json:"min_increment" validate:"required,gt=0"`
	DurationMinutes int           `json:"duration_minutes" validate:"required,min=1,max=60"`
}
