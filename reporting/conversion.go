package reporting

import (
	"time"

	"github.com/f-taxes/german_tax_report/fifo"
	. "github.com/f-taxes/german_tax_report/global"
	"github.com/f-taxes/german_tax_report/proto"
	d "github.com/shopspring/decimal"
)

type Conversion struct {
	Ts            time.Time
	Account       string
	From          string    // = Quote asset for buy, base asset for sells.
	To            string    // = Base asset for bus, quote asset for sells.
	ToAmount      d.Decimal // = Amount
	FromAmount    d.Decimal // = Value
	FromAmountEur d.Decimal // = ValueC
	FromEntries   fifo.EntryList
	Price         d.Decimal // = Price
	PriceEur      d.Decimal // = PriceC (only in there for informational purposes)
	FeeCurrency   string    // = FeeCurrency
	Fee           d.Decimal // = Fee
	FeeEur        d.Decimal // = FeeC
	Result        ConversionResult
}

type ConversionResult struct {
	CostEur     d.Decimal // Original cost in EUR of the assets to convert from.
	ValueEur    d.Decimal // EUR value of the asset when it was obtained.
	PnlEur      d.Decimal
	FeePayedEur d.Decimal
}

// Create a conversion from a trade. Depending on the direction of the trade, assets and prices are assigned accordingly.
func (r *Generator) TradeToConversion(trade *proto.Trade) *Conversion {
	if trade.Action == proto.TxAction_BUY {
		return &Conversion{
			Ts:            trade.Ts.AsTime(),
			Account:       trade.Account,
			To:            trade.Asset,
			From:          trade.Quote,
			Price:         D(trade.Price),
			PriceEur:      D(trade.PriceC),
			ToAmount:      D(trade.Amount),
			FromAmount:    D(trade.Value),
			FromAmountEur: D(trade.ValueC),
			FeeCurrency:   trade.FeeCurrency,
			Fee:           D(trade.Fee),
			FeeEur:        D(trade.FeeC),
		}
	}

	if trade.Action == proto.TxAction_SELL {
		// a := &Conversion{
		// 	Ts:            trade.Ts.AsTime(),
		// 	Account:       trade.Account,
		// 	To:            trade.Asset,
		// 	From:          trade.Quote,
		// 	Price:         D(trade.Price),
		// 	PriceEur:      D(trade.PriceC),
		// 	ToAmount:      D(trade.Amount),
		// 	FromAmount:    D(trade.Value),
		// 	FromAmountEur: D(trade.ValueC),
		// 	FeeCurrency:   trade.FeeCurrency,
		// 	Fee:           D(trade.Fee),
		// 	FeeEur:        D(trade.FeeC),
		// }

		// d, _ := json.MarshalIndent(a, "", "  ")
		// fmt.Printf("%s\n", d)

		price := D("1").Div(D(trade.Price))
		priceC := D(trade.QuotePriceC)
		value := D(trade.Amount)
		toAmount := D(trade.Value)

		c := &Conversion{
			Ts:            trade.Ts.AsTime(),
			Account:       trade.Account,
			To:            trade.Quote,
			From:          trade.Asset,
			Price:         price,
			PriceEur:      priceC,
			ToAmount:      toAmount,
			FromAmount:    value,
			FromAmountEur: toAmount.Div(priceC),
			FeeCurrency:   trade.FeeCurrency,
			Fee:           D(trade.Fee),
			FeeEur:        D(trade.FeeC),
		}

		// d, _ = json.MarshalIndent(c, "", "  ")
		// fmt.Printf("%s\n", d)
		return c
	}

	return nil
}
