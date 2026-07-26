package dto

type TrialReq struct {
	Expired int `json:"expired" form:"expired" validate:"required,min=1"`
}
