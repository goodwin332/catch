package dto

type PrivateProfileResponse struct {
	UserID      string  `json:"user_id"`
	Email       string  `json:"email"`
	Username    string  `json:"username,omitempty"`
	DisplayName string  `json:"display_name,omitempty"`
	AvatarURL   string  `json:"avatar_url,omitempty"`
	Rating      int     `json:"rating"`
	Role        string  `json:"role"`
	BirthDate   *string `json:"birth_date,omitempty"`
	Bio         string  `json:"bio,omitempty"`
	Boat        string  `json:"boat,omitempty"`
	CountryCode string  `json:"country_code,omitempty"`
	CountryName string  `json:"country_name,omitempty"`
	CityName    string  `json:"city_name,omitempty"`
}

type PublicProfileResponse struct {
	UserID      string `json:"user_id"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	Rating      int    `json:"rating"`
	Bio         string `json:"bio,omitempty"`
	Boat        string `json:"boat,omitempty"`
	CountryName string `json:"country_name,omitempty"`
	CityName    string `json:"city_name,omitempty"`
}

type ProfileSearchResponse struct {
	Items []PublicProfileResponse `json:"items"`
}

type UpdateMyProfileRequest struct {
	Username    *string `json:"username,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	BirthDate   *string `json:"birth_date,omitempty"`
	Bio         *string `json:"bio,omitempty"`
	Boat        *string `json:"boat,omitempty"`
	CountryCode *string `json:"country_code,omitempty"`
	CountryName *string `json:"country_name,omitempty"`
	CityName    *string `json:"city_name,omitempty"`
}
