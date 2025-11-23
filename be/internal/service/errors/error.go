package errors

import "errors"

var (
	// general errors
	ErrBadRequest             = errors.New("bad request")
	ErrUnauthorized           = errors.New("unauthorized access")
	ErrForbidden              = errors.New("forbidden action")
	ErrInternalServer         = errors.New("internal server error")
	ErrEmailAlreadyExists     = errors.New("email already registered")
	ErrPaymentFailed          = errors.New("payment failed")
	ErrMailNotificationFailed = errors.New("mail sending failed")

	// user errors
	ErrDonationItemInvalidInput    = errors.New("invalid input for donation item")
	ErrDonationItemUpdateForbidden = errors.New("only admin can update donation item")
	ErrDonationItemMailFailed      = errors.New("failed to send mail notification for donation item")

	// donation money errors
	ErrDonationMoneyInvalidInput    = errors.New("invalid input for donation money")
	ErrDonationMoneyUpdateForbidden = errors.New("only admin can update donation money")
	ErrDonationMoneyPaymentFailed   = errors.New("failed to process payment for donation money")

	// article errors
	ErrArticleInvalidInput    = errors.New("invalid input for article")
	ErrArticleUpdateForbidden = errors.New("only admin can update article")
)
