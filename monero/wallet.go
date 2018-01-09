package monero

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sort"

	digest "github.com/delphinus/go-digest-request"
)

// Wallet describes a wallet.
type Wallet struct {
	// The monero-wallet-rpc endpoint.
	URL string

	// The username and password to use for authentication.
	Username string
	Password string
}

// NewWallet creates a new wallet.
func NewWallet(url, u, p string) *Wallet {
	return &Wallet{url, u, p}
}

// request requests and parses JSON from the Monero wallet RPC client into a
// specified interface.
func (w *Wallet) request(m string, p, t interface{}) error {
	c := digest.New(context.Background(), w.Username, w.Password)

	dat := encodeRequest(m, p)
	req, err := http.NewRequest("POST", w.URL, bytes.NewBuffer(dat))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Length", (string)(len(dat)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := c.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("request: Returned invalid statuscode %d",
			res.StatusCode)
	}

	return decodeResponse(res.Body, t)
}

// Address returns the wallet's address, a 95-character hex address string of
// the monero-wallet-rpc in session.
func (w *Wallet) Address() (string, error) {
	var t = struct {
		Address string `json:"address"`
	}{}
	if err := w.request("getaddress", nil, &t); err != nil {
		return "", err
	}

	return t.Address, nil
}

// Balance represents the values returned by `Balance`.
type Balance struct {
	// The total balance of the current monero-wallet-rpc in session.
	Balance float64

	// Funds that are sufficiently deep enough in the Monero blockchain to be
	// considered safe to spend.
	UnBalance float64
}

// Balance returns the wallet's balance.
func (w *Wallet) Balance() (Balance, error) {
	var t = struct {
		Balance   uint64 `json:"balance"`
		UnBalance uint64 `json:"unlocked_balance"`
	}{}

	if err := w.request("getbalance", nil, &t); err != nil {
		return Balance{}, err
	}

	return Balance{
		float64(t.Balance) / 1.e+12,
		float64(t.UnBalance) / 1.e+12,
	}, nil
}

// Height returns the wallet's current block height.
func (w *Wallet) Height() (int64, error) {
	var t = struct {
		Height int64 `json:"height"`
	}{}

	if err := w.request("getheight", nil, &t); err != nil {
		return 0, err
	}

	return t.Height, nil
}

// Transfer represents the values returned by `incoming_transfers`.
type Transfer struct {
	// The transaction ID of the transfer.
	ID string

	// The amount of the transfer.
	Amount float64

	// The fee of the transfer.
	Fee float64

	// The status of the transfer, can be "incoming", "outgoing", "pending" and
	// "failed".
	Status string

	// The transfer timestamp.
	Timestamp uint64
}

// Transfers returns a list of incoming transfers to the wallet.
// TODO: Add Destinations.
// TODO: Add pool.
// TODO: God damn, simplify this.
func (w *Wallet) Transfers(in, out, pending, failed bool) ([]Transfer,
	error) {
	type tt = struct {
		TXID      string `json:"txid"`
		PaymentID string `json:"payment_id"`
		Height    uint64 `json:"height"`
		Timestamp uint64 `json:"timestamp"`
		Amount    uint64 `json:"amount"`
		Fee       uint64 `json:"fee"`
		Note      string `json:"note"`
	}
	var t = struct {
		In      []tt `json:"in"`
		Out     []tt `json:"out"`
		Pending []tt `json:"pending"`
		Failed  []tt `json:"failed"`
	}{}

	if err := w.request("get_transfers", struct {
		In      bool `json:"in"`
		Out     bool `json:"out"`
		Pending bool `json:"pending"`
		Failed  bool `json:"failed"`
	}{
		in, out, pending, failed,
	}, &t); err != nil {
		return []Transfer{}, err
	}

	var tr []Transfer
	for _, p := range t.In {
		tr = append(tr, Transfer{
			p.TXID,
			float64(p.Amount) / 1.e+12,
			float64(p.Fee) / 1.e+12,
			"incoming",
			p.Timestamp,
		})
	}
	for _, p := range t.Out {
		tr = append(tr, Transfer{
			p.TXID,
			float64(p.Amount) / 1.e+12,
			float64(p.Fee) / 1.e+12,
			"outgoing",
			p.Timestamp,
		})
	}
	for _, p := range t.Pending {
		tr = append(tr, Transfer{
			p.TXID,
			float64(p.Amount) / 1.e+12,
			float64(p.Fee) / 1.e+12,
			"pending",
			p.Timestamp,
		})
	}
	for _, p := range t.Failed {
		tr = append(tr, Transfer{
			p.TXID,
			float64(p.Amount) / 1.e+12,
			float64(p.Fee) / 1.e+12,
			"failed",
			p.Timestamp,
		})
	}
	sort.Slice(tr, func(i, j int) bool {
		return tr[i].Timestamp > tr[j].Timestamp
	})

	return tr, nil
}
