package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/plaid/plaid-go/v20/plaid"
)

var f *finDB
var plaidClient *plaid.APIClient
var err error

func createLinkTokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	user := plaid.LinkTokenCreateRequestUser{
		ClientUserId: "USER_ID_FROM_YOUR_DB",
	}
	request := plaid.NewLinkTokenCreateRequest(
		"Plaid Test",
		"en",
		[]plaid.CountryCode{plaid.COUNTRYCODE_US},
		user,
	)

	request.SetProducts([]plaid.Products{plaid.PRODUCTS_AUTH, plaid.PRODUCTS_TRANSACTIONS})
	request.SetLinkCustomizationName("default")
	resp, httpResp, err := plaidClient.PlaidApi.LinkTokenCreate(context.TODO()).LinkTokenCreateRequest(*request).Execute()
	if err != nil {
		fmt.Println(err)
		fmt.Println(httpResp)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(createLinkTokenResp{LinkToken: ""})
		return
	}

	linkToken := resp.GetLinkToken()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createLinkTokenResp{LinkToken: linkToken})
}

func exchangePublicTokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req exchangePublicTokenReq
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	exchangePublicTokenReq := plaid.NewItemPublicTokenExchangeRequest(req.PublicToken)
	exchangePublicTokenResp, _, err := plaidClient.PlaidApi.ItemPublicTokenExchange(context.TODO()).ItemPublicTokenExchangeRequest(
		*exchangePublicTokenReq,
	).Execute()
	if err != nil {
		fmt.Println(err)
	}

	accessToken := exchangePublicTokenResp.GetAccessToken()

	_, err = f.db.Exec(`INSERT INTO tokens (access_token, item_id, request_id) VALUES (?, ?, ?);`, exchangePublicTokenResp.AccessToken, exchangePublicTokenResp.ItemId, exchangePublicTokenResp.RequestId)
	if err != nil {
		fmt.Println(err)
	}

	balanceRequest := plaid.NewAccountsBalanceGetRequest(
		accessToken,
	)
	balancesResp, _, err := plaidClient.PlaidApi.AccountsBalanceGet(context.TODO()).AccountsBalanceGetRequest(*balanceRequest).Execute()
	if err != nil {
		fmt.Println(err)
	}

	for _, acc := range balancesResp.Accounts {
		_, err = f.db.Exec(`INSERT INTO accounts (account_id, balance, name) VALUES (?, ?, ?);`, acc.AccountId, *(acc.Balances.Available.Get()), acc.Name)
		if err != nil {
			fmt.Println(err)
		}
	}

	request := plaid.NewTransactionsSyncRequest(
		accessToken,
	)
	transactionsResp, _, err := plaidClient.PlaidApi.TransactionsSync(context.TODO()).TransactionsSyncRequest(*request).Execute()
	if err != nil {
		fmt.Println(err)
	}

	for _, added := range transactionsResp.Added {
		_, err := f.db.Exec(`INSERT INTO transactions (amount, account, merchant, date, category) VALUES (?, ?, ?, ?, ?)`, added.Amount, added.AccountId, added.MerchantName.Get(), added.Date, added.PersonalFinanceCategory.Get().Primary)
		if err != nil {
			fmt.Println(err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func accountsListHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := f.db.Query(`SELECT account_id, balance, name FROM accounts`)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var accounts []account
	var totalBalance float64
	for rows.Next() {
		var currentAccount account
		rows.Scan(&currentAccount.AccountID, &currentAccount.Balance, &currentAccount.Name)
		accounts = append(accounts, currentAccount)
		totalBalance += currentAccount.Balance
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accountsResponse{Balance: totalBalance, Accounts: accounts})
}

func categoriesHandler(w http.ResponseWriter, r *http.Request) {
	categories := map[string]float64{
		"BANK_FEES":                 0,
		"ENTERTAINMENT":             0,
		"FOOD_AND_DRINK":            0,
		"GENERAL_MERCHANDISE":       0,
		"GENERAL_SERVICES":          0,
		"GOVERNMENT_AND_NON_PROFIT": 0,
		"HOME_IMPROVEMENT":          0,
		"INCOME":                    0,
		"LOAN_PAYMENTS":             0,
		"MEDICAL":                   0,
		"PERSONAL_CARE":             0,
		"RENT_AND_UTILITIES":        0,
		"TRANSFER_IN":               0,
		"TRANSFER_OUT":              0,
		"TRANSPORTATION":            0,
		"TRAVEL":                    0,
	}

	rows, err := f.db.Query(`SELECT category, amount FROM transactions`)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var category string
	var amount float64
	for rows.Next() {
		rows.Scan(&category, &amount)
		categories[category] += amount
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}

func transactionsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := f.db.Query(`SELECT amount, account, merchant, date, category FROM transactions`)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var transactions []transaction
	var currTransaction transaction
	for rows.Next() {
		rows.Scan(&currTransaction.Amount, &currTransaction.AccountID, &currTransaction.MerchantName, &currTransaction.Date, &currTransaction.Category)

		accountNameRow, err := f.db.Query(`SELECT name FROM accounts WHERE account_id = ? LIMIT 1`, currTransaction.AccountID)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		for accountNameRow.Next() {
			accountNameRow.Scan(&currTransaction.Account)
		}
		transactions = append(transactions, currTransaction)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transactionResponse{NumberOfTransactions: len(transactions), Transactions: transactions})
}

func main() {
	godotenv.Load()

	// DB init
	f, err = NewFinDB()
	if err != nil {
		panic(err)
	}
	defer f.db.Close()

	// Plaid client init
	plaidConfig := plaid.NewConfiguration()
	plaidConfig.AddDefaultHeader("PLAID-CLIENT-ID", os.Getenv("PLAID_CLIENT_ID"))
	plaidConfig.AddDefaultHeader("PLAID-SECRET", os.Getenv("PLAID_SECRET"))
	plaidConfig.UseEnvironment(plaid.Environment(os.Getenv("PLAID_ENV")))
	plaidClient = plaid.NewAPIClient(plaidConfig)

	// HTTP handler
	fs := http.FileServer(http.Dir("./build"))
	http.Handle("/", fs)

	http.HandleFunc("/api/create_link_token", createLinkTokenHandler)
	http.HandleFunc("/api/exchange_public_token", exchangePublicTokenHandler)
	http.HandleFunc("/api/accountsList", accountsListHandler)
	http.HandleFunc("/api/categorySpending", categoriesHandler)
	http.HandleFunc("/api/transactions", transactionsHandler)

	port := ":5050"
	fmt.Println("Listening on port ", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
