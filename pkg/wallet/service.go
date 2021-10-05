package wallet

import (
	"bufio"
	"errors"
	"github.com/delgoden/wallet/pkg/types"
	"github.com/google/uuid"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
)

type Service struct {
	nextAccountID int64
	accounts      []*types.Account
	payments      []*types.Payment
	favorites     []*types.Favorite
}

var (
	ErrPhoneRegistered      = errors.New("phone already registered")
	ErrAmountMustBePositive = errors.New("amount must be greater than zero")
	ErrAccountNotFound      = errors.New("account not found")
	ErrNotEnoughBalance     = errors.New("not enough balance")
	ErrPaymentNotFound      = errors.New("payment not found")
	ErrPaymentsNotFound = errors.New("payments not found")
	ErrFavoriteNotFound = errors.New("favorite not found")
)

// RegisterAccount provides a method for adding new accounts
func (s *Service) RegisterAccount(phone types.Phone) (*types.Account, error) {
	for _, account := range s.accounts {
		if account.Phone == phone {
			return nil, ErrPhoneRegistered
		}
	}

	s.nextAccountID++
	account := &types.Account{
		ID:      s.nextAccountID,
		Phone:   phone,
		Balance: 0,
	}
	s.accounts = append(s.accounts, account)

	return account, nil
}

// Deposit provides a method to process balance replenishment
func (s *Service) Deposit(accountID int64, amount types.Money) error {
	if amount <= 0 {
		return ErrAmountMustBePositive
	}

	var account *types.Account
	for _, acc := range s.accounts {
		if acc.ID == accountID {
			account = acc
			break
		}
	}

	if account == nil {
		return ErrAccountNotFound
	}

	account.Balance += amount
	return nil
}

// Pay provides a payment processing method
func (s *Service) Pay(accountID int64, amount types.Money, category types.PaymentCategory) (*types.Payment, error) {
	if amount <= 0 {
		return nil, ErrAmountMustBePositive
	}

	var account *types.Account
	for _, acc := range s.accounts {
		if acc.ID == accountID {
			account = acc
			break
		}
	}

	if account == nil {
		return nil, ErrAccountNotFound
	}

	if account.Balance < amount {
		return nil, ErrNotEnoughBalance
	}

	account.Balance -= amount
	paymentID := uuid.New().String()
	payment := &types.Payment{
		ID:        paymentID,
		AccountID: accountID,
		Amount:    amount,
		Category:  category,
		Status:    types.PaymentStatusInProgress,
	}
	s.payments = append(s.payments, payment)

	return payment, nil
}

// FindAccountByID search for an account by ID
func (s *Service) FindAccountByID(accountID int64) (*types.Account, error) {
	var account *types.Account
	for _, acc := range s.accounts {
		if acc.ID == accountID {
			account = acc
			break
		}
	}

	if account == nil {
		return nil, ErrAccountNotFound
	}

	return account, nil
}

// FindPaymentByID search for a payment by ID
func (s *Service) FindPaymentByID(paymentID string) (*types.Payment, error) {
	var payment *types.Payment
	for _, pay := range s.payments {
		if pay.ID == paymentID {
			payment = pay
			break
		}
	}

	if payment == nil {
		return nil, ErrPaymentNotFound
	}

	return payment, nil
}

// Reject cancels the payment and returns the money to the balance
func (s *Service) Reject(paymentID string) error {
	payment, err := s.FindPaymentByID(paymentID)
	if err != nil {
		return err
	}

	payment.Status = types.PaymentStatusFail

	account, err1 := s.FindAccountByID(payment.AccountID)
	if err != nil {
		return err1
	}

	account.Balance += payment.Amount
	return nil
}

// Repeat allows the ID to repeat the payment
func (s *Service) Repeat(paymentID string) (*types.Payment, error) {
	payment, err := s.FindPaymentByID(paymentID)
	if err != nil {
		return nil, err
	}

	repeatPayment, err := s.Pay(payment.AccountID, payment.Amount, payment.Category)
	if err != nil {
		return nil, err
	}

	return repeatPayment, nil
}

// FavoritePayment creates favorites from a specific payment
func (s *Service) FavoritePayment(paymentID string, name string) (*types.Favorite, error) {
	payment, err := s.FindPaymentByID(paymentID)
	if err != nil {
		return nil, err
	}

	favoriteID := uuid.New().String()
	favorite := &types.Favorite{
		ID:        favoriteID,
		AccountID: payment.AccountID,
		Name:      name,
		Amount:    payment.Amount,
		Category:  payment.Category,
	}
	s.favorites = append(s.favorites, favorite)

	return favorite, nil
}

// PayFromFavorite makes a payment from a specific favorite
func (s *Service) PayFromFavorite(favoriteID string) (*types.Payment, error) {
	favorite, err := s.FindFavoriteByID(favoriteID)
	if err != nil {
		return nil, err
	}

	paymentID := uuid.New().String()
	payment := &types.Payment{
		ID:        paymentID,
		AccountID: favorite.AccountID,
		Amount:    favorite.Amount,
		Category:  favorite.Category,
		Status:    types.PaymentStatusInProgress,
	}
	s.payments = append(s.payments, payment)
	return payment, nil
}

// FindFavoriteByID search gor a favorite by ID
func (s *Service) FindFavoriteByID(favoriteID string) (*types.Favorite, error) {
	var favorite *types.Favorite
	for _, fvr := range s.favorites {
		if fvr.ID == favoriteID {
			favorite = fvr
			break
		}
	}

	if favorite == nil {
		return nil, ErrFavoriteNotFound
	}

	return favorite, nil
}

// ExportToFile exports all accounts to a file
func (s *Service) ExportToFile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Print(err)
		}
	}()

	for _, account := range s.accounts {
		accStr := strconv.FormatInt(account.ID, 10) + ";" + string(account.Phone) + ";" + strconv.FormatInt(int64(account.Balance), 10) + "|"
		_, err := file.Write([]byte(accStr))
		if err != nil {
			return err
		}
	}

	return err
}

// ImportFromFile imports all accounts from a file
func (s *Service) ImportFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Print(err)
		}
	}()

	reader := bufio.NewReader(file)

	for {
		accStr, err := reader.ReadString('|')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		accSls := strings.Split(accStr, ";")
		ID, err := strconv.Atoi(accSls[0])
		if err != nil {
			return err
		}
		Balance, err := strconv.Atoi(strings.TrimSuffix(accSls[2], "|"))
		if err != nil {
			return err
		}
		s.accounts = append(s.accounts, &types.Account{ID: int64(ID), Phone: types.Phone(accSls[1]), Balance: types.Money(Balance)})
	}
	return err
}

// Export - export of all accounts, payments and favorites to files
func (s *Service) Export(dir string) error {
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		panic(err)
	}
	if len(s.accounts) != 0 {
		accFile, err := os.Create(dir + "/accounts.dump")
		if err != nil {
			log.Print(err)
		}
		defer func() {
			if cerr := accFile.Close(); cerr != nil {
				if err == nil {
					log.Print(err)
				}
			}
		}()

		for _, account := range s.accounts {
			accStr := strconv.FormatInt(account.ID, 10) + ";" + string(account.Phone) + ";" + strconv.FormatInt(int64(account.Balance), 10) + "\n"
			_, err := accFile.Write([]byte(accStr))
			if err != nil {
				return err
			}
		}
	}

	if len(s.payments) != 0 {

		payFile, err := os.Create(dir + "/payments.dump")
		if err != nil {
			log.Print(err)
		}
		defer func() {
			if cerr := payFile.Close(); cerr != nil {
				if err == nil {
					log.Print(err)
				}
			}
		}()

		for _, payment := range s.payments {
			payStr := payment.ID + ";" + strconv.FormatInt(payment.AccountID, 10) + ";" + strconv.FormatInt(int64(payment.Amount), 10) + ";" + string(payment.Category) + ";" + string(payment.Status) + "\n"
			_, err := payFile.WriteString(payStr)
			if err != nil {
				return err
			}
		}
	}

	if len(s.favorites) != 0 {

		favFile, err := os.Create(dir + "/favorites.dump")
		if err != nil {
			log.Print(err)
		}
		defer func() {
			if cerr := favFile.Close(); cerr != nil {
				if err == nil {
					log.Print(err)
				}
			}
		}()

		for _, favorite := range s.favorites {
			favStr := favorite.ID + ";" + strconv.FormatInt(favorite.AccountID, 10) + ";" + favorite.Name + ";" + strconv.FormatInt(int64(favorite.Amount), 10) + ";" + string(favorite.Category) + "\n"
			_, err := favFile.WriteString(favStr)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Import - import of all accounts, payments and favorites from files
func (s *Service) Import(dir string) error {
	file, err := os.Open(dir + "/accounts.dump")
	if err == nil || errors.Is(err, os.ErrExist) {
		reader := bufio.NewReader(file)
		for {
			accStr, err := reader.ReadString('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			accSls := strings.Split(strings.TrimSuffix(accStr, "\n"), ";")
			ID, err := strconv.Atoi(accSls[0])
			if err != nil {
				return err
			}
			Balance, err := strconv.Atoi(accSls[2])
			if err != nil {
				return err
			}
			_, err = s.FindAccountByID(int64(ID))
			if err != nil {
				account := &types.Account{
					ID:      int64(ID),
					Phone:   types.Phone(accSls[1]),
					Balance: types.Money(Balance),
				}
				s.accounts = append(s.accounts, account)
				s.nextAccountID = int64(ID)
			}

		}
	} else {
		log.Print(err)
	}

	defer func() {
		err := file.Close()
		if err != nil {
			log.Print(err)
		}
	}()

	payFile, err := os.Open(dir + "/payments.dump")
	if err == nil || errors.Is(err, os.ErrExist) {
		reader := bufio.NewReader(payFile)
		for {
			payStr, err := reader.ReadString('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			paySls := strings.Split(strings.TrimSuffix(payStr, "\n"), ";")
			payID := paySls[0]
			if err != nil {
				return err
			}
			AccountID, err := strconv.Atoi(paySls[1])
			if err != nil {
				return err
			}
			Amount, err := strconv.Atoi(paySls[2])
			if err != nil {
				return err
			}
			_, err = s.FindPaymentByID(payID)
			if err != nil {
				payment := &types.Payment{
					ID:        payID,
					AccountID: int64(AccountID),
					Amount:    types.Money(Amount),
					Category:  types.PaymentCategory(paySls[3]),
					Status:    types.PaymentStatus(paySls[4]),
				}
				s.payments = append(s.payments, payment)
			}
		}
	} else {
		log.Print(err)
	}

	defer func() {
		err := payFile.Close()
		if err != nil {
			log.Print(err)
		}
	}()

	favFile, err := os.Open(dir + "/favorites.dump")
	if err == nil || errors.Is(err, os.ErrExist) {
		reader := bufio.NewReader(favFile)
		for {
			favStr, err := reader.ReadString('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			favSls := strings.Split(strings.TrimSuffix(favStr, "\n"), ";")
			favID := favSls[0]
			if err != nil {
				return err
			}
			AccountID, err := strconv.Atoi(favSls[1])
			if err != nil {
				return err
			}
			Amount, err := strconv.Atoi(favSls[3])
			if err != nil {
				return err
			}
			_, err = s.FindFavoriteByID(favID)
			if err != nil {
				favorite := &types.Favorite{
					ID:        favID,
					AccountID: int64(AccountID),
					Name:      favSls[2],
					Amount:    types.Money(Amount),
					Category:  types.PaymentCategory(favSls[4]),
				}
				s.favorites = append(s.favorites, favorite)
			}
		}
	} else {
		log.Print(err)
	}

	defer func() {
		err := favFile.Close()
		if err != nil {
			log.Print(err)
		}
	}()

	return nil
}

// ExportAccountHistory finds all payments for a specific account
func (s *Service) ExportAccountHistory(accountID int64) ([]types.Payment, error) {
	_, err := s.FindAccountByID(accountID)
	if err != nil {
		return nil, err
	}
	if len(s.payments) == 0 {
		return nil, ErrPaymentsNotFound
	}
	var payments []types.Payment
	for _, payment := range s.payments {
		if payment.AccountID == accountID {
			payments = append(payments, *payment)
		}
	}

	return payments, nil
}

// HistoryToFiles saves all payments of a specific account to a file

func (s *Service) HistoryToFiles(payments []types.Payment, dir string, records int) error {
	if len(payments) == 0 {
		return nil
	}

	if len(payments) <= records {
		exportPayments(payments, dir+"/payments.dump")
		return nil
	}
	var iterator int = 1
	count := 0
	for i := 1; i <= len(payments); i++ {
		strIterator := strconv.Itoa(iterator)
		fileName := dir + "/payments" + strIterator + ".dump"
		if i == len(payments) && i-count != records {
			exportPayments(payments[count:i], fileName)
		}

		if i-count == records {
			exportPayments(payments[count:i], fileName)
			count += records
			iterator++
		}
	}
	return nil
}

// exportPayments preparation of payments for export to file
func exportPayments(payments []types.Payment, path string) {
	pay := ""

	for _, payment := range payments {

		pay += payment.ID + ";"
		pay += strconv.Itoa(int(payment.AccountID)) + ";"
		pay += strconv.Itoa(int(payment.Amount)) + ";"
		pay += string(payment.Category) + ";"
		pay += string(payment.Status) + ";"
		pay += "\n"
	}
	err := WriteDump(path, pay)
	if err != nil {
		log.Print(err)

	}

}

// WriteDump recording prepared payments for export to a file
func WriteDump(path, payRec string) error {
	payFile, err := os.Create(path)
	if err != nil {
		log.Print(err)
	}
	defer func() {
		if cerr := payFile.Close(); cerr != nil {
			if err == nil {
				log.Print(err)
			}
		}
	}()

	_, err = payFile.WriteString(payRec)
	if err != nil {
		return err
	}

	return nil
}

// SumPayments summation of all payments in a competitive mode
func (s *Service) SumPayments(goroutines int) types.Money {
	result := types.Money(0)
	if goroutines == 0 || goroutines == 1 {
		for _, payment := range s.payments {
			result += payment.Amount
		}
		return result
	}

	wg := sync.WaitGroup{}

	mu := sync.Mutex{}
	paymentsOnGoroutine := len(s.payments) / goroutines
	count := 0
	for i := 0; i < len(s.payments); i++ {
		if i == len(s.payments)-1 {
			wg.Add(1)
			go func(count, i int) {
				defer wg.Done()
				tmp := types.Money(0)
				for _, payment := range s.payments[count:] {
					tmp += payment.Amount
				}
				mu.Lock()
				defer mu.Unlock()
				result += tmp
			}(count, i)
		}

		if i-count == paymentsOnGoroutine {
			wg.Add(1)
			go func(count, i int) {
				defer wg.Done()
				tmp := types.Money(0)
				for _, payment := range s.payments[count:i] {
					tmp += payment.Amount
				}
				mu.Lock()
				defer mu.Unlock()
				result += tmp
			}(count, i)
			count += paymentsOnGoroutine
		}
	}
	wg.Wait()
	return result
}

// FilterPayments account payment filter
func (s *Service) FilterPayments(accountID int64, goroutines int) ([]types.Payment, error) {

	if goroutines == 0 || goroutines == 1 {
		payments, err := s.ExportAccountHistory(accountID)
		if err != nil {
			return nil, err
		}
		return payments, nil
	}

	_, err := s.FindAccountByID(accountID)
	if err != nil {
		return nil, err
	}

	payments := []types.Payment{}
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	paymentsOnGoroutine := len(s.payments) / goroutines
	count := 0
	for i := 0; i < len(s.payments); i++ {
		if i == len(s.payments)-1 {
			wg.Add(1)
			go func(count, i int) {
				defer wg.Done()
				tmp := []types.Payment{}
				for _, payment := range s.payments[count:] {
					if payment.AccountID == accountID {
						tmp = append(tmp, *payment)
					}
				}
				mu.Lock()
				defer mu.Unlock()
				payments = append(payments, tmp...)
			}(count, i)
		}

		if i-count == paymentsOnGoroutine {
			wg.Add(1)
			go func(count, i int) {
				defer wg.Done()
				tmp := []types.Payment{}
				for _, payment := range s.payments[count:i] {
					if payment.AccountID == accountID {
						tmp = append(tmp, *payment)
					}
				}
				mu.Lock()
				defer mu.Unlock()
				payments = append(payments, tmp...)
			}(count, i)
			count += paymentsOnGoroutine
		}
	}
	wg.Wait()
	return payments, nil
}