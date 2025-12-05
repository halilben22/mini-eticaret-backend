package enums

type OrderStatus string

const (
	OrderStatusPending        OrderStatus = "pending"
	OrderStatusWaitingPayment OrderStatus = "waiting_payment"
	OrderStatusPaid           OrderStatus = "paid"
	OrderStatusShipped        OrderStatus = "shipped"
	OrderStatusCancelled      OrderStatus = "cancelled"
)
