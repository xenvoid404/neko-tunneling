package model

type TrialRequest struct {
	Expired int `json:"expired" form:"expired" validate:"required,min=1,max=1440"`
}

type CreateRequest struct {
	Username   string  `json:"username" form:"username" validate:"required,min=3,max=20,alphanum"`
	Password   *string `json:"password" form:"password" validate:"omitempty,min=3,max=20,alphanum"`
	LimitIP    *int    `json:"limit_ip" form:"limit_ip" validate:"omitempty,gte=0"`
	LimitQuota *int    `json:"limit_quota" form:"limit_quota" validate:"omitempty,gte=0"`
	Expired    int     `json:"expired" form:"expired" validate:"required,min=1"`
}

type RenewRequest struct {
	Expired int `json:"expired" form:"expired" validate:"required,min=1"`
}
