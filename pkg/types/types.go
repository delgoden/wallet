package types

// Money amount of money in minimum currency units (cents, rubles, dirhams, etc.)
type Money int64

// PaymentCategory the category in which the payment was made (cars, pharmacies, food, etc.)
type PaymentCategory string

// PaymentStatus payment status
type PaymentStatus string

// Predefined payment statuses
const (
	PaymentStatusOK         PaymentStatus = "OK"
	PaymentStatusFail       PaymentStatus = "FAIL"
	PaymentStatusInProgress PaymentStatus = "INPROGRESS"
)

// Payment payment information
type Payment struct {
	ID        string
	AccountID int64
	Amount    Money
	Category  PaymentCategory
	Status    PaymentStatus
}

// Phone user phone
type Phone string

// Account present info for user account
type Account struct {
	ID      int64
	Phone   Phone
	Balance Money
}

// Favorite present favorite payments user
type Favorite struct {
	ID        string
	AccountID int64
	Name      string
	Amount    Money
	Category  PaymentCategory
}