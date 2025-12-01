package repository

import (
	"context"
	"milestone3/be/internal/dto"
	"milestone3/be/internal/entity"
	"os"

	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/coreapi"
	"gorm.io/gorm"
)

type PaymentRepo struct {
	db *gorm.DB
	ctx context.Context
}

func NewPaymentRepository(db *gorm.DB, ctx context.Context) *PaymentRepo {
	return &PaymentRepo{db: db, ctx: ctx}
}

func (pr *PaymentRepo) Create(payment *entity.Payment) (error) {
	if err := pr.db.WithContext(pr.ctx).Create(payment).Error; err != nil {
		return err
	}

	return nil
}

func (pr *PaymentRepo) CreateMidtrans(payment entity.Payment) (res dto.PaymentResponse, err error) {
	serverKey := os.Getenv("MIDTRANS_SERVER_KEY")
	c := coreapi.Client{}
	c.New(serverKey, midtrans.Sandbox)
	chargeReq := &coreapi.ChargeReq{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID: payment.OrderId,
			GrossAmt: int64(payment.GrossAmount),
		},
		CustomerDetails: &midtrans.CustomerDetails{
			FName: payment.Name,
			Email: payment.Email,
			Phone: payment.NoHp,
		},
	}
	coreApiResp, err := c.ChargeTransaction(chargeReq)
	if err != nil {
		return dto.PaymentResponse{}, err
	}

	resp := dto.PaymentResponse{
		PaymentLinkUrl: coreApiResp.RedirectURL,
		TransactionId: coreApiResp.TransactionID,
		ExpiryTime: coreApiResp.ExpiryTime,
	}


	return resp, nil
}