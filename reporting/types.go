package reporting

import (
	"time"

	"github.com/f-taxes/german_tax_report/fifo"
	. "github.com/f-taxes/german_tax_report/global"
	"github.com/f-taxes/german_tax_report/proto"
	"github.com/shopspring/decimal"
	d "github.com/shopspring/decimal"
)

type Deposit struct {
	Ts          time.Time
	Type        string
	Account     string
	Asset       string
	Amount      decimal.Decimal
	Fee         decimal.Decimal
	FeeEur      decimal.Decimal
	Source      string
	Entries     fifo.EntryList
	QueueBefore []fifo.Asset
	QueueAfter  []fifo.Asset
	Error       string
	Warning     string
}

type Withdrawal struct {
	RecID       string
	Ts          time.Time
	Type        string
	Account     string
	Asset       string
	Amount      decimal.Decimal
	Fee         decimal.Decimal
	FeeEur      decimal.Decimal
	Destination string
	Entries     fifo.EntryList
	QueueBefore []fifo.Asset
	QueueAfter  []fifo.Asset
	Error       string
	Warning     string
}

type Conversion struct {
	Ts               time.Time
	RecID            string
	Type             string
	Account          string
	From             string    // = Quote asset for buy, base asset for sells.
	FromDecimals     int32     // = Number of decimals of either the quote or base asset. Depending on the direction of the trade.
	To               string    // = Base asset for bus, quote asset for sells.
	ToDecimals       int32     // = Number of decimals of either the quote or base asset. Depending on the direction of the trade.
	ToAmount         d.Decimal // = Amount
	ToAmountNet      d.Decimal // = Amount - Fee
	ToAmountEur      d.Decimal // = Amount * QuotePriceC
	ToAmountNetEur   d.Decimal // = Amount * QuotePriceC - QuoteFeeC
	FromAmount       d.Decimal // = Value
	FromAmountNet    d.Decimal // = Value - FeeC
	FromAmountEur    d.Decimal // = ValueC
	FromAmountNetEur d.Decimal // = ValueC - FeeC
	FromEntries      []fifo.Asset
	QueueBefore      []fifo.Asset
	QueueAfter       []fifo.Asset
	Price            d.Decimal // = Price
	PriceEur         d.Decimal // = PriceC (only in there for informational purposes)
	FeeCurrency      string    // = FeeCurrency
	Fee              d.Decimal // = Fee
	FeeDecimals      int32     // = Number of decimals of the fee currency.
	QuoteFeeCurrency string    // = QuoteFeeCurrency
	QuoteFee         d.Decimal // = QuoteFee
	QuoteFeeDecimals int32     // = Number of decimals of the quote fee currency.
	FeeEur           d.Decimal // = FeeC + QuoteFeeC
	Result           ConversionResult
	Warning          string
	Error            string
	IsDerivative     bool
	IsMarginTrade    bool
	IsPhysical       bool
}

type ConversionResult struct {
	CostEur  d.Decimal // Original cost in EUR of the assets to convert from.
	ValueEur d.Decimal // EUR value of the asset when it was obtained.
	// Pnl      d.Decimal
	PnlEur      d.Decimal
	FeePayedEur d.Decimal
}

// Create a conversion from a trade. Depending on the direction of the trade, assets and prices are assigned accordingly.
func (r *Generator) TradeToConversion(trade *proto.Trade) *Conversion {
	if trade.Action == proto.TxAction_BUY {
		toFee := D(trade.Fee.Amount).Abs()
		toFeeC := D(trade.Fee.AmountC).Abs()
		fromFee := D(trade.QuoteFee.Amount).Abs()
		fromFeeC := D(trade.QuoteFee.AmountC).Abs()

		return &Conversion{
			RecID:            trade.TxID,
			Ts:               trade.Ts.AsTime(),
			Type:             "conversion",
			Account:          trade.Account,
			To:               trade.Asset,
			ToDecimals:       trade.AssetDecimals,
			From:             trade.Quote,
			FromDecimals:     trade.QuoteDecimals,
			Price:            D(trade.Price).Round(IfThen(trade.QuoteDecimals > 0, trade.QuoteDecimals, 8)),
			PriceEur:         D(trade.PriceC).Round(4),
			ToAmount:         D(trade.Amount).Round(IfThen(trade.AssetDecimals > 0, trade.AssetDecimals, 8)),
			ToAmountNet:      D(trade.Amount).Sub(toFee).Round(IfThen(trade.AssetDecimals > 0, trade.AssetDecimals, 8)),
			ToAmountEur:      D(trade.Amount).Mul(D(trade.PriceC)).Round(4),
			ToAmountNetEur:   D(trade.Amount).Mul(D(trade.PriceC)).Sub(toFeeC).Round(4),
			FromAmount:       D(trade.Value).Round(IfThen(trade.QuoteDecimals > 0, trade.QuoteDecimals, 8)),
			FromAmountNet:    D(trade.Value).Sub(fromFee).Round(IfThen(trade.QuoteDecimals > 0, trade.QuoteDecimals, 8)),
			FromAmountEur:    D(trade.ValueC).Round(4),
			FromAmountNetEur: D(trade.ValueC).Sub(fromFeeC).Round(4),
			FeeCurrency:      trade.Fee.Currency,
			Fee:              D(trade.Fee.Amount).Round(IfThen(trade.Fee.Decimals > 0, trade.Fee.Decimals, 8)),
			FeeDecimals:      trade.Fee.Decimals,
			QuoteFeeCurrency: trade.QuoteFee.Currency,
			QuoteFee:         D(trade.QuoteFee.Amount).Round(IfThen(trade.QuoteFee.Decimals > 0, trade.QuoteFee.Decimals, 8)),
			QuoteFeeDecimals: trade.QuoteFee.Decimals,
			FeeEur:           D(trade.Fee.AmountC).Add(D(trade.QuoteFee.AmountC)).Round(4),
			IsDerivative:     trade.Props.IsDerivative,
			IsMarginTrade:    trade.Props.IsMarginTrade,
			IsPhysical:       trade.Props.IsPhysical,
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
		fromFee := D(trade.Fee.Amount)
		fromFeeC := D(trade.Fee.AmountC)
		toFee := D(trade.QuoteFee.Amount).Abs()
		toFeeC := D(trade.QuoteFee.AmountC).Abs()
		// toFee := D(trade.QuoteFee).Abs()
		// toFeeC := D(trade.QuoteFeeC).Abs()

		c := &Conversion{
			RecID:          trade.TxID,
			Ts:             trade.Ts.AsTime(),
			Type:           "conversion",
			Account:        trade.Account,
			To:             trade.Quote,
			ToDecimals:     trade.QuoteDecimals,
			From:           trade.Asset,
			FromDecimals:   trade.AssetDecimals,
			Price:          price,
			PriceEur:       priceC.Round(4),
			ToAmount:       toAmount.Round(IfThen(trade.QuoteDecimals > 0, trade.QuoteDecimals, 8)),
			ToAmountNet:    toAmount.Sub(toFee).Round(IfThen(trade.QuoteDecimals > 0, trade.QuoteDecimals, 8)),
			ToAmountEur:    toAmount.Mul(D(trade.QuotePriceC)).Round(4),
			ToAmountNetEur: toAmount.Mul(D(trade.QuotePriceC)).Sub(toFeeC).Round(4),
			// ToAmount:         toAmount.Sub(toFee).Round(IfThen(trade.QuoteDecimals > 0, trade.QuoteDecimals, 8)),
			// ToAmountEur:      toAmount.Mul(D(trade.QuotePriceC)).Sub(toFeeC).Round(4),
			FromAmount:       value,
			FromAmountNet:    value.Sub(fromFee),
			FromAmountEur:    value.Round(4),
			FromAmountNetEur: value.Sub(fromFeeC).Round(4),
			FeeCurrency:      trade.Fee.Currency,
			Fee:              D(trade.Fee.Amount),
			FeeDecimals:      trade.Fee.Decimals,
			QuoteFeeCurrency: trade.QuoteFee.Currency,
			QuoteFee:         D(trade.QuoteFee.Amount),
			QuoteFeeDecimals: trade.QuoteFee.Decimals,
			FeeEur:           D(trade.Fee.AmountC).Add(D(trade.QuoteFee.AmountC)).Round(4),
			IsDerivative:     trade.Props.IsDerivative,
			IsMarginTrade:    trade.Props.IsMarginTrade,
			IsPhysical:       trade.Props.IsPhysical,
		}

		if priceC.GreaterThan(d.Zero) && trade.Asset != "EUR" {
			c.FromAmountEur = toAmount.Div(priceC).Round(4)
		}

		// d, _ = json.MarshalIndent(c, "", "  ")
		// fmt.Printf("%s\n", d)
		return c
	}

	return nil
}
