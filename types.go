package main

type (
	createLinkTokenResp struct {
		LinkToken string `json:"link_token"`
	}

	exchangePublicTokenReq struct {
		PublicToken string `json:"public_token"`
	}

	account struct {
		Name      string  `json:"name"`
		AccountID string  `json:"account_id"`
		Balance   float64 `json:"balance"`
	}

	accountsResponse struct {
		Balance  float64   `json:"balance"`
		Accounts []account `json:"accounts"`
	}

	transaction struct {
		Date         string  `json:"date"`
		Amount       float32 `json:"amount"`
		Account      string  `json:"account"`
		AccountID    string  `json:"account_id"`
		Category     string  `json:"category"`
		MerchantName string  `json:"merchant_name"`
	}

	transactionResponse struct {
		NumberOfTransactions int           `json:"number_of_transactions"`
		Transactions           []transaction `json:"transactions"`
	}
)
