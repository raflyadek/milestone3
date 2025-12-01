package routes

import (
	"milestone3/be/api/middleware"
	"milestone3/be/internal/controller"
)

func (r *EchoRouter) RegisterPaymentRoutes(paymentCtrl *controller.PaymentController) {
	paymentRoutes := r.echo.Group("/payments")
	paymentRoutes.Use(middleware.JWTMiddleware)

	//payment endpoint
	paymentRoutes.POST("", paymentCtrl.CreatePayment)
}