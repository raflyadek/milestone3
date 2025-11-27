package dto

type DonationDTO struct {
	ID        uint    `validate:"omitempty"`
	Amount    float64 `validate:"required,gt=0"`
	DonorName string  `validate:"required,min=2,max=100"`
}
