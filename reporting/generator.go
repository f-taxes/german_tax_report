package reporting

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/f-taxes/german_tax_report/fifo"
	"github.com/f-taxes/german_tax_report/global"
	. "github.com/f-taxes/german_tax_report/global"
	g "github.com/f-taxes/german_tax_report/grpc_client"
	"github.com/f-taxes/german_tax_report/proto"
	"github.com/kataras/golog"
	d "github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Generator struct {
	Recs              []any
	accounts          AccountFifo
	recentWithdrawals []*Withdrawal
	marginShortTrades map[string]bool
	l                 sync.Mutex
}

func NewGenerator() *Generator {
	return &Generator{
		Recs:              []any{},
		accounts:          AccountFifo{},
		recentWithdrawals: []*Withdrawal{},
		marginShortTrades: map[string]bool{},
	}
}

func (r *Generator) Start(from, to time.Time) error {
	recordChan := make(chan *proto.Record)
	doneChan := make(chan struct{})

	go func() {
		for rec := range recordChan {
			if rec.Transfer != nil {
				if c := r.processTransfer(rec.Transfer); c != nil {
					r.Recs = append(r.Recs, c)
				}
			}

			if rec.Trade != nil {
				if c := r.processTrade(rec.Trade); c != nil {
					r.Recs = append(r.Recs, c)
				}
			}
		}

		close(doneChan)
	}()

	err := g.GrpcClient.StreamRecords(context.Background(), &proto.StreamRecordsJob{
		Plugin:        Plugin.ID,
		PluginVersion: Plugin.Version,
		From:          timestamppb.New(from),
		To:            timestamppb.New(to),
	}, recordChan)

	close(recordChan)
	<-doneChan

	return err
}

// func (r *Generator) process(recordChan chan *proto.Record) {
// 	for rec := range recordChan {
// 		if rec.Transfer != nil {
// 			c := r.processTransfer(rec.Transfer)
// 			r.Recs = append(r.Recs, c)
// 		}

// 		if rec.Trade != nil {
// 			c := r.processTrade(rec.Trade)
// 			r.Recs = append(r.Recs, c)
// 		}
// 	}
// }

func (r *Generator) processTransfer(transfer *proto.Transfer) (w *Withdrawal) {
	if transfer.Action == proto.TransferAction_DEPOSIT {
		deposit := Deposit{
			Ts:      transfer.Ts.AsTime(),
			Type:    "deposit",
			Asset:   transfer.Asset,
			Amount:  D(transfer.Amount),
			Account: transfer.Account,
			Source:  transfer.Source,
			Fee:     D(transfer.Fee),
			FeeEur:  D(transfer.FeeC),
		}

		var matching *Withdrawal

		for i, w := range r.recentWithdrawals {
			if w.Destination == transfer.Account && w.Ts.Sub(transfer.Ts.AsTime()).Abs() < time.Minute*15 && global.PercentageDelta(D(transfer.Amount), w.Entries.TotalUnitsLeft()).LessThan(D("10")) {
				matching = r.recentWithdrawals[i]
				r.recentWithdrawals = global.RemoveElementUnordered(r.recentWithdrawals, i)
				break
			}
		}

		if matching != nil {
			deposit.Entries = matching.Entries.Copy()
			r.Recs = append(r.Recs, deposit)

			for _, e := range deposit.Entries {
				r.accounts.Get(transfer.Account).Add(transfer.Asset, e.Copy())
			}
		}

		if matching == nil {
			if transfer.Asset == "EUR" {
				entry := fifo.Entry{
					Units:       D(transfer.Amount, d.Zero),
					UnitsLeft:   D(transfer.Amount, d.Zero),
					UnitCostEur: D("1"),
					Ts:          transfer.Ts.AsTime(),
				}

				deposit.Entries = append(deposit.Entries, entry)
				deposit.Warning = "No records about how this deposit came do be. Assuming bank account transfer."
				r.Recs = append(r.Recs, deposit)

				r.accounts.Get(transfer.Account).Add(transfer.Asset, entry)
			} else {
				deposit.Error = "No prior withdrawal found to explain how this deposit was possible."
				r.Recs = append(r.Recs, deposit)
			}
		}
	}

	if transfer.Action == proto.TransferAction_WITHDRAWAL {
		asset, err := r.accounts.Get(transfer.Account).Take(transfer.Asset, D(transfer.Amount), transfer.AssetDecimals)
		if err != nil {
			golog.Errorf("Failed to withdraw %s %s from %s: %v", transfer.Amount, transfer.Asset, transfer.Account, err)
		}

		if D(transfer.Fee).GreaterThan(d.Zero) {
			feeAssets, err := r.accounts.Get(transfer.Account).Take(transfer.FeeCurrency, D(transfer.Fee), transfer.FeeDecimals)
			if err != nil {
				golog.Errorf("Failed to withdraw %s %s from %s: %v", transfer.Amount, transfer.Asset, transfer.Account, err)
			}

			asset.Entries = append(asset.Entries, feeAssets.Entries...)
		}

		w = &Withdrawal{
			RecID:       transfer.TxID,
			Ts:          transfer.Ts.AsTime(),
			Type:        "withdrawal",
			Account:     transfer.Account,
			Asset:       transfer.Asset,
			Amount:      D(transfer.Amount),
			Destination: transfer.Destination,
			Entries:     asset.Entries,
			Fee:         D(transfer.Fee),
			FeeEur:      D(transfer.FeeC),
		}

		r.recentWithdrawals = append(r.recentWithdrawals, w)

		return

		// golog.Infof("TRANSFER %s: Take %s %s (withdrawal)", transfer.Account, transfer.Amount, transfer.Asset)

		// desc.Add(transfer.Account, convert{Asset: transfer.Asset, })
		// if transfer.Destination != "" {
		// 	for i := range entries {
		// 		r.accounts.Get(transfer.Destination).Add(transfer.Asset, entries[i])
		// 	}
		// }
		// r.desc.Add(transfer.Account, move{Asset: transfer.Asset, Entries: entries, Ts: transfer.Ts.AsTime(), Source: transfer.Source, Destination: transfer.Destination})
	}

	return
}

func (r *Generator) processMarginTrade(trade *proto.Trade) (c *Conversion) {
	c = r.TradeToConversion(trade)
	key := fmt.Sprintf("%s_%s", c.Account, trade.Ticker)

	defer func() {
		r.Recs = append(r.Recs, c)
	}()

	if r.accounts.Get(c.Account, "margin").HasUnits(c.To) {
		pastEntries := fifo.EntryList{}
		isShort := r.marginShortTrades[key]

		if trade.Action == proto.TxAction_BUY {
			if isShort {
				pastEntries, err := r.accounts.Get(c.Account, "margin").Take(c.To, c.ToAmount, c.ToDecimals)
				c.FromEntries = []fifo.Asset{pastEntries}

				if err != nil {
					golog.Errorf("Failed to take units for %s out of fifo queue (ID=%s, TS=%s, MARGIN=%v): %v", c.From, trade.TxID, trade.Ts.AsTime(), trade.Props.IsMarginTrade, err)
					return
				}
			} else {
				r.accounts.Get(c.Account, "margin").Add(c.To, fifo.NewEntry(c.ToAmount, c.FromAmountEur, c.FeeEur, c.Ts))
			}
		}

		if trade.Action == proto.TxAction_SELL {
			if isShort {
				r.accounts.Get(c.Account, "margin").Add(c.To, fifo.NewEntry(c.ToAmount, c.FromAmountEur, c.FeeEur, c.Ts))
			} else {
				pastEntries, err := r.accounts.Get(c.Account, "margin").Take(c.From, c.FromAmount, c.FromDecimals)
				c.FromEntries = []fifo.Asset{pastEntries}

				if err != nil {
					golog.Errorf("Failed to take units for %s out of fifo queue (ID=%s, TS=%s, MARGIN=%v): %v", c.From, trade.TxID, trade.Ts.AsTime(), trade.Props.IsMarginTrade, err)
					return
				}
			}
		}

		totalCostC := d.Zero

		for _, e := range pastEntries {
			totalCostC = e.UnitCostEur.Mul(e.UnitsLeft).Add(e.UnitFeeCostEur.Mul(e.UnitsLeft))
		}

		c.Result = ConversionResult{
			CostEur:     totalCostC,
			ValueEur:    c.FromAmountEur.Sub(c.FeeEur),
			PnlEur:      c.FromAmountEur.Sub(c.FeeEur).Sub(totalCostC),
			FeePayedEur: c.FeeEur,
		}
	} else {
		if trade.Action == proto.TxAction_SELL {
			r.accounts.Get(c.Account, "margin").Add(c.From, fifo.NewEntry(c.FromAmount, c.ToAmountEur, c.FeeEur, c.Ts))
			r.marginShortTrades[key] = true
		} else {
			r.accounts.Get(c.Account, "margin").Add(c.To, fifo.NewEntry(c.ToAmount, c.FromAmountEur, c.FeeEur, c.Ts))
		}
	}

	// Remove fee from fifo queue.
	if !c.Fee.IsZero() {
		_, err := r.accounts.Get(c.Account).Take(c.FeeCurrency, c.Fee, c.FeeDecimals)

		if err != nil {
			golog.Errorf("Failed to take units for %s out of fifo queue to pay fees (ID=%s, TS=%s): %v", c.FeeCurrency, trade.TxID, trade.Ts.AsTime(), err)
		}
	}

	if !c.QuoteFee.IsZero() {
		_, err := r.accounts.Get(c.Account).Take(c.QuoteFeeCurrency, c.QuoteFee, c.QuoteFeeDecimals)

		if err != nil {
			// fmt.Printf("%s\n", r.accounts.Get(trade.Account).Read("EUR").Entries.Print())
			golog.Errorf("Failed to take units for %s out of fifo queue to pay fees (ID=%s, TS=%s): %v", c.QuoteFeeCurrency, trade.TxID, trade.Ts.AsTime(), err)
		}
	}

	return
}

func (r *Generator) processTrade(trade *proto.Trade) (c *Conversion) {
	if trade.Props.IsMarginTrade {
		return r.processMarginTrade(trade)
	}

	if trade.Value == "3740.56051" {
		fmt.Printf("%+v\n", "now")
	}

	c = r.TradeToConversion(trade)

	if c.ToAmount.Equal(D("3740.5605")) {
		fmt.Printf("%+v\n", "now")
	}

	defer func() {
		if r.accounts.Get(c.Account).HasUnits(c.To) {
			c.QueueAfter = append(c.QueueAfter, r.accounts.Get(c.Account).Read(c.To))
		}

		if r.accounts.Get(c.Account).HasUnits(c.From) {
			c.QueueAfter = append(c.QueueAfter, r.accounts.Get(c.Account).Read(c.From))
		}
	}()

	if r.accounts.Get(c.Account).HasUnits(c.To) {
		c.QueueBefore = append(c.QueueBefore, r.accounts.Get(c.Account).Read(c.To))
	}

	if r.accounts.Get(c.Account).HasUnits(c.From) {
		c.QueueBefore = append(c.QueueBefore, r.accounts.Get(c.Account).Read(c.From))
	}

	r.accounts.Get(c.Account).Add(c.To, fifo.NewEntry(c.ToAmount, c.FromAmountEur, c.FeeEur, c.Ts))
	assetExtracted, err := r.accounts.Get(c.Account).Take(c.From, c.FromAmount, c.FromDecimals)

	c.FromEntries = []fifo.Asset{assetExtracted}

	if err != nil {
		ferr := err.(fifo.FifoError)
		c.Error = fmt.Sprintf("Not enough %s available in FIFO queue. Trying to take %s but queue only holds %s (missing %s).", c.From, ferr.RequiredAmount, D(ferr.RequiredAmount).Sub(D(ferr.MissingAmount)).String(), ferr.MissingAmount)
		golog.Errorf("Failed to take units for %s out of fifo queue (ID=%s, TS=%s, MARGIN=%v): %v", c.From, trade.TxID, trade.Ts.AsTime(), trade.Props.IsMarginTrade, err)
	}

	// Remove fee from fifo queue.
	if !c.Fee.IsZero() {
		extractedEntries, err := r.accounts.Get(c.Account).Take(c.FeeCurrency, c.Fee, c.FeeDecimals)

		if err != nil {
			golog.Errorf("Failed to take units for %s out of fifo queue to pay fees (ID=%s, TS=%s): %v", c.FeeCurrency, trade.TxID, trade.Ts.AsTime(), err)
		}

		c.FromEntries = append(c.FromEntries, extractedEntries)
	}

	if !c.QuoteFee.IsZero() {
		if trade.TxID == "TPJR2V-X7L54-Z6665S" {
			fmt.Printf("%+v\n", "now")
			// fmt.Printf("%s\n", r.accounts.Get(trade.Account).Read("EUR").Entries.Print())
		}

		fmt.Printf("%+v\n", r.accounts.Get(c.Account).Read(c.QuoteFeeCurrency).Entries.Print())
		extractedEntries, err := r.accounts.Get(c.Account).Take(c.QuoteFeeCurrency, c.QuoteFee, c.QuoteFeeDecimals)
		fmt.Printf("%+v\n", r.accounts.Get(c.Account).Read(c.QuoteFeeCurrency).Entries.Print())

		if err != nil {
			// fmt.Printf("%s\n", r.accounts.Get(trade.Account).Read("EUR").Entries.Print())
			golog.Errorf("Failed to take units for %s out of fifo queue to pay fees (ID=%s, TS=%s): %v", c.QuoteFeeCurrency, trade.TxID, trade.Ts.AsTime(), err)
		}

		c.FromEntries = append(c.FromEntries, extractedEntries)
	}

	totalCostC := d.Zero

	for _, e := range assetExtracted.Entries {
		totalCostC = totalCostC.Add(e.UnitCostEur.Mul(e.UnitsLeft).Round(4).Add(e.UnitFeeCostEur.Mul(e.UnitsLeft).Round(4))).Round(4)
	}

	// fmt.Printf("total cost: %s €\n", totalCostC)
	// feeC := c.FeeEur
	// feeC := D(trade.FeeC)
	// fmt.Printf("selling value: %s €\n", c.FromAmountEur.Sub(c.FeeEur))

	// fmt.Printf("pnl: %s €\n", c.FromAmountEur.Sub(c.FeeEur).Sub(totalCostC))
	// fmt.Printf("pnl: %s €\n", D(trade.ValueC).Sub(feeC).Sub(totalCostC))

	c.Result = ConversionResult{
		CostEur:     totalCostC,
		ValueEur:    c.FromAmountEur.Sub(c.FeeEur).Round(4),
		PnlEur:      c.FromAmountEur.Sub(c.FeeEur).Sub(totalCostC).Round(4),
		FeePayedEur: c.FeeEur,
	}

	return

	// fmt.Printf("%s\n", r.accounts.Get(trade.Account).Read("EUR").Entries.Print())

	// r.desc.Add(c.Account, convert{Asset: c.To, Quote: c.From, Entries: pastEntries, Ts: c.Ts, Pnl: D(trade.ValueC).Sub(c.FeeEur).Sub(totalCostC).String()})
	// fifoAssetMap := r.accounts.Get(tx.Account).Get(tx.Asset)

	// if fifoAssetMap.BaseDirectionInverted == BASE_DIR_NONE {
	// 	fifoAssetMap.BaseDirectionInverted = baseDirFromTxAction(tx.Action)
	// }

	// if baseDirFromTxAction(tx.Action) == fifoAssetMap.BaseDirectionInverted {
	// 	fifoAssetMap.Queue.Add(fifo.NewFifoRecord(StrToDecimal(tx.Amount), StrToDecimal(tx.Price), StrToDecimal(tx.Fee)))
	// } else {
	// 	fifoAssetMap.Queue.Take(fifo.NewFifoRecord(StrToDecimal(tx.Amount), StrToDecimal(tx.Price), StrToDecimal(tx.Fee)))
	// }
}
