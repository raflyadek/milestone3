package entity

type Payment struct {
	Id int
	OrderId string
	GrossAmount float64
	Name string
	NoHp string
	Email string
}