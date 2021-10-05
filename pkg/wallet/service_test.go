package wallet

import (
	"fmt"
	"github.com/delgoden/wallet/pkg/types"
	"github.com/google/uuid"
	"log"
	"reflect"
	"testing"
)

type testService struct {
	*Service
}

type testAccount struct {
	phone    types.Phone
	balance  types.Money
	payments []struct {
		amount   types.Money
		category types.PaymentCategory
	}
}

var defaultTestAccount = testAccount{
	phone:   "992000000001",
	balance: 10_000_000_000_00,
	payments: []struct {
		amount   types.Money
		category types.PaymentCategory
	}{
		{amount: 1_000_00, category: "auto"},
	},
}

func newTestService() *testService {
	return &testService{Service: &Service{}}
}

func (s *testService) addAccount(data testAccount) (*types.Account, []*types.Payment, error) {
	account, err := s.RegisterAccount(data.phone)
	if err != nil {
		return nil, nil, fmt.Errorf("can't register account, error = %v", err)
	}

	err = s.Deposit(account.ID, data.balance)
	if err != nil {
		return nil, nil, fmt.Errorf("can't deposity account, error = %v", err)
	}

	payments := make([]*types.Payment, len(data.payments))
	for i, payment := range data.payments {
		payments[i], err = s.Pay(account.ID, payment.amount, payment.category)
		if err != nil {
			return nil, nil, fmt.Errorf("can't make payment, error = %v", err)
		}
	}
	return account, payments, nil
}

func TestService_FindAccountByID_success(t *testing.T) {
	s := newTestService()
	account, _, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}
	got, err := s.FindAccountByID(account.ID)
	if err != nil {
		t.Errorf("FindAccountByID(): error = %v", err)
		return
	}

	if !reflect.DeepEqual(account, got) {
		t.Errorf("FindAccountByID(): wrong account returned = %v", err)
		return
	}
}

func TestService_FindAccountByID_failed(t *testing.T) {
	s := newTestService()
	_, _, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}
	_, err = s.FindAccountByID(10)
	if err != ErrAccountNotFound {
		t.Errorf("FindAccountByID(): error = %v", err)
		return
	}

}

func TestService_FindPaymentByID_success(t *testing.T) {
	s := newTestService()
	_, payments, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	payment := payments[0]
	got, err := s.FindPaymentByID(payment.ID)
	if err != nil {
		t.Errorf("FindPaymentByID(): error = %v", err)
		return
	}

	if !reflect.DeepEqual(payment, got) {
		t.Errorf("FindPaymentByID(): wrong payment returned = %v", err)
		return
	}

}

func TestService_FindPaymentByID_fail(t *testing.T) {
	s := newTestService()
	_, _, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	_, err = s.FindPaymentByID(uuid.New().String())
	if err == nil {
		t.Errorf("FindPaymentByID(): must return error, returned nil")
		return
	}

	if err != ErrPaymentNotFound {
		t.Errorf("FindPaymentByID(): must return ErrPaymentNotFound, returned = %v", err)
		return
	}

}

func Test_Reject_success(t *testing.T) {
	s := newTestService()
	_, payments, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	payment := payments[0]
	err = s.Reject(payment.ID)
	if err != nil {
		t.Errorf("Reject(): error = %v", err)
		return
	}

	savedPayment, err := s.FindPaymentByID(payment.ID)
	if err != nil {
		t.Errorf("Reject(): can't find payment by id, error = %v", err)
		return
	}

	if savedPayment.Status != types.PaymentStatusFail {
		t.Errorf("Reject(): status didn't changed, payment = %v", savedPayment)
		return
	}

	savedAccount, err := s.FindAccountByID(payment.AccountID)
	if err != nil {
		t.Errorf("Reject(): can't find account by id, error = %v", err)
		return
	}

	if savedAccount.Balance != defaultTestAccount.balance {
		t.Errorf("Reject(): balance didn't changed, account = %v", savedAccount)
		return
	}
}

func Test_Reject_fail(t *testing.T) {
	s := newTestService()
	_, _, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	err = s.Reject(uuid.New().String())
	if err == nil {
		t.Errorf("Reject():  must return error, returned nil")
		return
	}
	if err != ErrPaymentNotFound {
		t.Errorf("FindPaymentByID(): must return ErrPaymentNotFound, returned = %v", err)
		return
	}

}

func Test_Repeat_success(t *testing.T) {
	s := newTestService()
	_, payments, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	payment := payments[0]
	repeatPayment, err := s.Repeat(payment.ID)
	if err != nil {
		t.Errorf("Repeat(): error = %v", err)
		return
	}

	if repeatPayment.AccountID != payment.AccountID {
		t.Errorf("Repeat(): accounts do not match payment = %v, repeatPayment = %v", payment.AccountID, repeatPayment.AccountID)
		return
	}

	if repeatPayment.Amount != payment.Amount {
		t.Errorf("Repeat(): ammounts do not match payment = %v, repeatPayment = %v", payment.Amount, repeatPayment.Amount)
		return
	}

	if repeatPayment.Category != payment.Category {
		t.Errorf("Repeat(): categories do not match payment = %v, repeatPayment = %v", payment.Category, repeatPayment.Category)
		return
	}
}

func Test_Repeat_fail(t *testing.T) {
	s := newTestService()
	_, _, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	_, err = s.Repeat(uuid.New().String())
	if err == nil {
		t.Errorf("Repeat(): error = %v", err)
		return
	}
}

func Test_FavoritePayment_success(t *testing.T) {
	s := newTestService()
	_, payments, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	payment := payments[0]
	favorite, err := s.FavoritePayment(payment.ID, "new")
	if err != nil {
		t.Errorf("FavoritePayment(): error = %v", err)
		return
	}

	if favorite.AccountID != payment.AccountID {
		t.Errorf("FavoritePayment(): accounts do not match payment = %v, favorite = %v", payment.AccountID, favorite.AccountID)
		return
	}

	if favorite.Amount != payment.Amount {
		t.Errorf("FavoritePayment(): ammounts do not match payment = %v, favorite = %v", payment.Amount, favorite.Amount)
		return
	}

	if favorite.Category != payment.Category {
		t.Errorf("FavoritePayment(): categories do not match payment = %v, favorite = %v", payment.Category, favorite.Category)
		return
	}
}

func Test_FavoritePayment_fail(t *testing.T) {
	s := newTestService()
	_, _, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	paymentID := uuid.New().String()
	_, err = s.FavoritePayment(paymentID, "new")
	if err == nil {
		t.Errorf("FavoritePayment(): error = %v", err)
		return
	}
}

func Test_PayFromFavorite_success(t *testing.T) {
	s := newTestService()
	_, payments, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	payment := payments[0]
	favorite, err := s.FavoritePayment(payment.ID, "new")
	if err != nil {
		t.Errorf("FavoritePayment(): error = %v", err)
		return
	}

	paymentNew, err := s.PayFromFavorite(favorite.ID)
	if err != nil {
		t.Errorf("PayFromFavorite(): error = %v", err)
		return
	}

	if favorite.AccountID != paymentNew.AccountID {
		t.Errorf("PayFromFavorite(): accounts do not match paymentNew = %v, favorite = %v", paymentNew.AccountID, favorite.AccountID)
		return
	}

	if favorite.Amount != paymentNew.Amount {
		t.Errorf("PayFromFavorite(): ammounts do not match paymentNew = %v, favorite = %v", paymentNew.Amount, favorite.Amount)
		return
	}

	if favorite.Category != paymentNew.Category {
		t.Errorf("PayFromFavorite(): categories do not match paymentNew = %v, favorite = %v", paymentNew.Category, favorite.Category)
		return
	}

}

func Test_PayFromFavorite_fail(t *testing.T) {
	s := newTestService()
	_, payments, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	payment := payments[0]
	_, err = s.FavoritePayment(payment.ID, "new")
	if err != nil {
		t.Errorf("FavoritePayment(): error = %v", err)
		return
	}

	favoriteID := uuid.New().String()
	_, err = s.PayFromFavorite(favoriteID)
	if err == nil {
		t.Errorf("FavoritePayment(): error = %v", err)
		return
	}
}

func TestExport(t *testing.T) {
	s := newTestService()
	_, payments, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	payment := payments[0]
	_, err = s.FavoritePayment(payment.ID, "new")
	if err != nil {
		t.Errorf("FavoritePayment(): error = %v", err)
		return
	}

	err = s.Export("02877c15-eda1-4d1e-879a-2f2083b8514f")

	if err != nil {
		log.Print(err)
	}
}

func TestImport(t *testing.T) {
	s := newTestService()
	account, payments, err := s.addAccount(defaultTestAccount)
	if err != nil {
		t.Error(err)
		return
	}

	payment := payments[0]
	_, err = s.FavoritePayment(payment.ID, "new")
	if err != nil {
		t.Errorf("FavoritePayment(): error = %v", err)
		return
	}

	_ = s.Export("data")

	err = s.Import("data")

	if !reflect.DeepEqual(account, s.accounts[0]) {
		t.Errorf("ImportF(): wrong account returned = %v", err)
		return
	}
}

func BenchmarkSumPayments(b *testing.B) {
	s := newTestService()
	account, _, err := s.addAccount(defaultTestAccount)
	if err != nil {
		b.Error(err)
		return
	}
	for j := 0; j < 11; j++ {
		_, err := s.Pay(account.ID, 1_000_00, "mobile")
		if err != nil {
			b.Errorf(("Pay(): wrong = %v"), err)
		}
	}
	want := types.Money(0)
	for _, pay := range s.payments {
		want += pay.Amount
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := s.SumPayments(2)
		b.StopTimer()
		if result != want {
			b.Fatalf("invalid result got: %v; want: %v", result, want)
		}
		b.StartTimer()
	}
}