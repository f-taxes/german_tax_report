package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/f-taxes/german_tax_report/fifo"
	. "github.com/f-taxes/german_tax_report/global"
	g "github.com/f-taxes/german_tax_report/grpc_client"
	"github.com/f-taxes/german_tax_report/proto"
	"github.com/kataras/golog"
	d "github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Generator struct {
	// desc     DescriptionMap
	accounts AccountFifo
}

func NewGenerator() *Generator {
	return &Generator{
		// desc:     DescriptionMap{},
		accounts: AccountFifo{},
	}
}

func (r *Generator) Start() error {
	recordChan := make(chan *proto.Record)

	go r.process(recordChan)

	err := g.GrpcClient.StreamRecords(context.Background(), &proto.StreamRecordsJob{
		Plugin:        Plugin.ID,
		PluginVersion: Plugin.Version,
		From:          timestamppb.New(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)),
		To:            timestamppb.Now(),
	}, recordChan)

	close(recordChan)

	return err
}

func (r *Generator) process(recordChan chan *proto.Record) {
	conversions := []*Conversion{}

	for rec := range recordChan {
		if rec.Transfer != nil {
			r.processTransfer(rec.Transfer)
		}

		if rec.Trade != nil {
			conversions = append(conversions, r.processTrade(rec.Trade))
		}
	}

	data, _ := json.MarshalIndent(conversions, "", "  ")
	os.WriteFile("./report.json", data, 0755)

	// for _, d := range r.desc {
	// 	for i := range d {
	// 		fmt.Println(d[i].String())
	// 	}
	// }

	// fmt.Printf("%+v\n", r.accounts)
}

func (r *Generator) processTransfer(transfer *proto.Transfer) {
	/*
		If the source is unknown but the asset is EUR, assume its ok to just accept the entry into the queue. Like it's coming from a fiat bank account.
		If asset isn't EUR and the source is unknown we need to error I guess.
		In that case there needs to be additional trades that show how the deposited asset came into possession using EUR in the first place.
		This is true for crypto but also foreign fiat.
	*/
	if transfer.Action == proto.TransferAction_DEPOSIT && transfer.Asset == "EUR" {
		// golog.Infof("TRANSFER %s: Add %s %s", transfer.Account, transfer.Amount, transfer.Asset)
		entry := fifo.Entry{
			Units:       D(transfer.Amount, d.Zero),
			UnitsLeft:   D(transfer.Amount, d.Zero),
			UnitCostEur: D("1"),
			Ts:          transfer.Ts.AsTime(),
		}

		r.accounts.Get(transfer.Account).Add(transfer.Asset, entry)
		// r.desc.Add(transfer.Account, move{Asset: transfer.Asset, Source: transfer.Source, Destination: transfer.Destination, Entries: fifo.EntryList{entry}, Ts: transfer.Ts.AsTime()})
	}

	if transfer.Action == proto.TransferAction_WITHDRAWAL {
		// golog.Infof("TRANSFER %s: Take %s %s (withdrawal)", transfer.Account, transfer.Amount, transfer.Asset)
		entries, err := r.accounts.Get(transfer.Account).Take(transfer.Asset, D(transfer.Amount))
		if err != nil {
			golog.Errorf("Failed to withdraw %s %s from %s: %v", transfer.Amount, transfer.Asset, transfer.Account, err)
		}

		// desc.Add(transfer.Account, convert{Asset: transfer.Asset, })
		if transfer.Destination != "" {
			for i := range entries {
				r.accounts.Get(transfer.Destination).Add(transfer.Asset, entries[i])
			}
		}
		// r.desc.Add(transfer.Account, move{Asset: transfer.Asset, Entries: entries, Ts: transfer.Ts.AsTime(), Source: transfer.Source, Destination: transfer.Destination})
	}
}

func (r *Generator) processTrade(trade *proto.Trade) *Conversion {
	if trade.Asset == "ADA" {
		fmt.Printf("%+v\n", "now")
		fmt.Printf("%s\n", r.accounts.Get(trade.Account).Print())
	}

	c := r.TradeToConversion(trade)

	r.accounts.Get(c.Account).Add(c.To, fifo.NewEntryFromTrade(c.ToAmount, c.FromAmountEur, c.FeeEur, c.Ts))

	pastEntries, err := r.accounts.Get(c.Account).Take(c.From, c.FromAmount)
	c.FromEntries = pastEntries

	if err != nil {
		golog.Errorf("Failed to take units for %s out of fifo queue (ID=%s, TS=%s): %v", c.From, trade.TxID, trade.Ts.AsTime(), err)
		return nil
	}

	// Remove fee from fifo queue.
	_, err = r.accounts.Get(c.Account).Take(c.FeeCurrency, c.Fee)
	if err != nil {
		fmt.Printf("%+v\n", err)
	}

	if trade.Asset == "ADA" {
		fmt.Printf("%+v\n", "now")
		fmt.Printf("%s\n", r.accounts.Get(trade.Account).Print())
	}

	totalCostC := d.Zero

	for _, e := range pastEntries {
		totalCostC = e.UnitCostEur.Mul(e.UnitsLeft).Add(e.UnitFeeCostEur.Mul(e.UnitsLeft))
	}

	// fmt.Printf("total cost: %s €\n", totalCostC)
	// feeC := c.FeeEur
	// feeC := D(trade.FeeC)
	// fmt.Printf("selling value: %s €\n", c.FromAmountEur.Sub(c.FeeEur))

	// fmt.Printf("pnl: %s €\n", c.FromAmountEur.Sub(c.FeeEur).Sub(totalCostC))
	// fmt.Printf("pnl: %s €\n", D(trade.ValueC).Sub(feeC).Sub(totalCostC))

	c.Result = ConversionResult{
		CostEur:     totalCostC,
		ValueEur:    c.FromAmountEur.Sub(c.FeeEur),
		PnlEur:      c.FromAmountEur.Sub(c.FeeEur).Sub(totalCostC),
		FeePayedEur: c.FeeEur,
	}

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

	return c
}
