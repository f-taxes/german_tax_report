package reporting

import (
	"context"
	"fmt"
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
	accounts *fifo.Fifo
}

func NewGenerator() *Generator {
	return &Generator{
		accounts: fifo.NewFifo(),
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
	for rec := range recordChan {
		if rec.Trade != nil {
			tx := rec.Trade

			r.accounts.Add(tx.Asset, fifo.NewEntryFromTx(tx))

			fmt.Printf("take: %s of %s\n", D(tx.Amount).Mul(D(tx.Price)), tx.Quote)
			pastEntries, err := r.accounts.Take(tx.Quote, D(tx.Amount).Mul(D(tx.Price)))
			fmt.Printf("%+v\n", pastEntries)

			if err != nil {
				golog.Errorf("Failed to get record for %s out of fifo queue: %v", tx.Quote, err)
				continue
			}

			totalCostC := d.Zero

			for _, e := range pastEntries {
				totalCostC = e.ValueC.Add(e.FeeC)
			}

			fmt.Printf("total cost: %s €\n", totalCostC)
			fee := D(tx.Fee).Mul(D(tx.FeePriceC))
			fmt.Printf("selling value: %s €\n", D(tx.ValueC).Sub(fee))

			fmt.Printf("pnl: %s €\n", D(tx.ValueC).Sub(fee).Sub(totalCostC))
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
	}

	fmt.Printf("%+v\n", r.accounts)
}
