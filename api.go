package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type Big struct {
	Whole   int
	Decimal int
}

func NewBig(number int, decimal int) Big {
	return Big{Whole: number, Decimal: decimal}
}

func (big *Big) Add(amount Big) {
	big.Whole += amount.Whole
	big.Decimal += amount.Decimal

	if big.Decimal >= 100 {
		big.Whole++
		big.Decimal -= 100
	}
}

func (big *Big) Sub(amount Big) {
	big.Whole -= amount.Whole
	big.Decimal -= amount.Decimal

	if big.Decimal < 0 {
		big.Whole--
		big.Decimal += 100
	}
}

func (big *Big) UnmarshalJSON(data []byte) error {
	// Zero is just one digit, so we don't have to do anything
	if len(data) == 1 {
		return nil
	}

	// The fractional amount is always three digits
	decimalBytes := data[len(data)-3 : len(data)-1]
	decimal, _ := strconv.Atoi(string(decimalBytes))
	wholeBytes := data[:len(data)-3]
	whole, _ := strconv.Atoi(string(wholeBytes))

	big.Whole = whole
	big.Decimal = decimal

	return nil
}

func (big Big) String() string {
	return fmt.Sprintf("%d.%d", big.Whole, big.Decimal)
}

type Budget struct {
	Id   string
	Name string
}

type Category struct {
	Id       string
	Name     string
	Balance  Big
	Budgeted Big
}

type Account struct {
	Id      string
	Name    string
	Balance Big
}

type Ynab struct {
	client *http.Client
	bearer string
}

func NewYnab() *Ynab {
	return &Ynab{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (y *Ynab) Do(request string, method string) (*http.Response, bool) {
	req, _ := http.NewRequest(method, request, nil)
	req.Header.Add("Authorization", y.bearer)
	response, err := y.client.Do(req)
	if err != nil {
		return nil, true
	}
	return response, false
}

// ValidateAndSetCode validates the code and sets the bearer token if the code is valid
func (y *Ynab) ValidateAndSetCode(code string) bool {
	req, _ := http.NewRequest("GET", "https://api.ynab.com/v1/user", nil)
	bearer := "Bearer " + code
	req.Header.Add("Authorization", bearer)
	response, _ := y.client.Do(req)

	if response == nil || response.StatusCode != 200 {
		return false
	}

	y.bearer = bearer
	return true
}

func readAndUnmarshal(response *http.Response, data interface{}) error {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, data)
}

func (y *Ynab) GetBudgets() []Budget {
	response, failed := y.Do("https://api.ynab.com/v1/budgets", "GET")

	if failed {
		return nil
	}

	var data struct {
		Data struct {
			Budgets       []Budget
			DefaultBudget Budget
		}
	}
	err := readAndUnmarshal(response, &data)
	if err != nil {
		return nil
	}

	return data.Data.Budgets
}

func (y *Ynab) GetAccounts(budgetId string) []Account {
	response, failed := y.Do("https://api.ynab.com/v1/budgets/"+budgetId+"/accounts", "GET")

	if failed {
		return nil
	}

	var data struct {
		Data struct {
			Accounts []Account
		}
	}
	err := readAndUnmarshal(response, &data)
	if err != nil {
		return nil
	}

	return data.Data.Accounts
}

func (y *Ynab) GetCategories(budgetId string) []Category {
	response, failed := y.Do("https://api.ynab.com/v1/budgets/"+budgetId+"/categories", "GET")

	if failed {
		return nil
	}

	var data struct {
		Data struct {
			CategoryGroups []struct {
				Categories []Category
			} `json:"category_groups"`
		}
	}
	err := readAndUnmarshal(response, &data)
	if err != nil {
		return nil
	}

	categories := make([]Category, 1)
	for _, group := range data.Data.CategoryGroups {
		categories = append(categories, group.Categories...)
	}

	return categories
}
