package reporting

import (
	"fmt"
	"testing"
	"time"

	"github.com/f-taxes/german_tax_report/fifo"
	"github.com/f-taxes/german_tax_report/proto"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSpot(t *testing.T) {
	D := decimal.RequireFromString

	// var wg sync.WaitGroup
	// wg.Add(1)
	toTime := func(str string) *timestamppb.Timestamp {
		v, err := time.Parse(time.RFC3339, str)
		if err != nil {
			panic(err)
		}
		return timestamppb.New(v)
	}

	// A expected result
	type er struct {
		Pnl string
		Fee string
	}

	// A expected entry
	type ee struct {
		Units       string
		UnitsLeft   string
		UnitCostEur string
		FeeEur      string
	}

	assertEntry := func(entry fifo.Entry, test ee) {
		if test.Units != "" {
			v := D(test.Units)
			assert.True(t, entry.Units.Equal(v), "Expected %s units, got %s instead.", v, entry.Units)
		}

		if test.UnitsLeft != "" {
			v := D(test.UnitsLeft)
			assert.True(t, entry.UnitsLeft.Equal(v), "Expected %s units left, got %s instead.", v, entry.UnitsLeft)
		}

		if test.FeeEur != "" {
			v := D(test.FeeEur)
			assert.True(t, entry.UnitFeeCostEur.Mul(entry.Units).Equal(v), "Expected %s€ fee, got %s instead.", v, entry.UnitFeeCostEur.Mul(entry.Units))
		}
	}

	assertResult := func(result ConversionResult, test er) {
		if test.Pnl != "" {
			v := D(test.Pnl)
			assert.True(t, result.PnlEur.Equal(v), "Expected %s€ pnl, got %s instead.", v, result.PnlEur)
		}

		if test.Fee != "" {
			v := D(test.Fee)
			assert.True(t, result.FeePayedEur.Equal(v), "Expected %s€ fee, got %s instead.", v, result.FeePayedEur)
		}
	}

	g := NewGenerator()

	g.processTransfer(&proto.Transfer{TxID: "1", Ts: toTime("2024-01-20T15:00:00Z"), Account: "Test1", Destination: "Test1", Asset: "EUR", Amount: "5000", Fee: "0", FeeC: "0", FeePriceC: "0", Action: proto.TransferAction_DEPOSIT})
	assertEntry(g.accounts.Get("Test1").Read("EUR").Entries[0], ee{Units: "5000", UnitCostEur: "1", FeeEur: "0"})

	g.processTrade(&proto.Trade{TxID: "2", Ts: toTime("2024-01-20T15:10:00Z"), Account: "Test1", Asset: "EUR", Quote: "USD", Amount: "5000", Price: "0.8", PriceC: "1.25", QuotePriceC: "1.25", Value: "4000", ValueC: "5000", Fee: "2", FeeC: "2.5", FeePriceC: "1.25", FeeCurrency: "USD", Action: proto.TxAction_SELL})

	// Expect 3200$, 2.5€ fee
	assertEntry(g.accounts.Get("Test1").Read("USD").Entries[0], ee{UnitsLeft: "3998", UnitCostEur: "1.25", FeeEur: "2.5"})

	// Should have left 790$ after exchanging 3200$ for 10 BTC and paying a 8$ fee.
	g.processTrade(&proto.Trade{TxID: "3", Ts: toTime("2024-01-20T15:20:00Z"), Account: "Test1", Asset: "BTC", Quote: "USD", Amount: "10", Price: "320", PriceC: "300", Value: "3200", ValueC: "3000", Fee: "8", FeeC: "10", FeePriceC: "0.8", FeeCurrency: "USD", Action: proto.TxAction_BUY})
	assertEntry(g.accounts.Get("Test1").Read("USD").Entries[0], ee{UnitsLeft: "790", UnitCostEur: "1.25", FeeEur: "2.5"})

	// Should own 10 BTC worth 300€ each.
	assertEntry(g.accounts.Get("Test1").Read("BTC").Entries[0], ee{UnitsLeft: "10", UnitCostEur: "300", FeeEur: "10"})

	conv := g.processTrade(&proto.Trade{TxID: "4", Ts: toTime("2024-01-20T15:30:00Z"), Account: "Test1", Asset: "ETH", Quote: "BTC", Amount: "40", Price: "0.05", PriceC: "20", Value: "2", ValueC: "800", Fee: "0.02", FeeC: "8", FeePriceC: "400", FeeCurrency: "BTC", Action: proto.TxAction_BUY})
	assertResult(conv.Result, er{Pnl: "190", Fee: "8"})
	fmt.Printf("%+v\n", conv)

	// Should have left 7.98 BTC worth 300€ each after exchanging 2 BTC for 40 ETH and paying a 0.2 BTC fee.
	assertEntry(g.accounts.Get("Test1").Read("BTC").Entries[0], ee{UnitsLeft: "7.98", UnitCostEur: "300", FeeEur: "10"})

	// Should own 40 ETH worth 20€ each at a price of 300€.
	assertEntry(g.accounts.Get("Test1").Read("ETH").Entries[0], ee{UnitsLeft: "40", UnitCostEur: "20", FeeEur: "8"})

	fmt.Printf("%+v\n", g.accounts.Get("Test1").Print())

	// assert.Equal(t, "\nFIFO queue for EUR:\n\n2024-01-20 15:00:00 +0000 UTC | 4000 x 1€\n\nFIFO queue for BTC:\n\n2024-01-20 15:10:00 +0000 UTC | 10 x 301€", g.accounts.Get("Test1").Print(), "Should be same")
	// wg.Done()
}
