package fifo

import (
	"errors"
	"fmt"
	"sort"
	"time"

	g "github.com/f-taxes/german_tax_report/global"
	"github.com/f-taxes/german_tax_report/proto"
	d "github.com/shopspring/decimal"
)

// type baseDir int

// const (
// 	BASE_DIR_NONE = baseDir(iota)
// 	BASE_DIR_LONG
// 	BASE_DIR_SHORT
// )

type Entry struct {
	Amount    d.Decimal
	Price     d.Decimal
	PriceC    d.Decimal
	Value     d.Decimal
	ValueC    d.Decimal
	Fee       d.Decimal
	FeeC      d.Decimal
	AssetType proto.AssetType
	Ts        time.Time
}

func (e *Entry) Copy() Entry {
	return Entry{
		Amount:    e.Amount.Copy(),
		Price:     e.Price.Copy(),
		PriceC:    e.PriceC.Copy(),
		Value:     e.Value.Copy(),
		ValueC:    e.ValueC.Copy(),
		Fee:       e.Fee.Copy(),
		FeeC:      e.FeeC.Copy(),
		AssetType: e.AssetType,
		Ts:        e.Ts,
	}
}

type EntryList []Entry

func (e EntryList) TotalAmount() d.Decimal {
	s := d.Zero

	for i := range e {
		s = s.Add(e[i].Amount)
	}

	return s
}

func NewEntryFromTrade(tx *proto.Trade) Entry {
	return Entry{
		Amount:    g.D(tx.Amount, d.Zero),
		Price:     g.D(tx.Price, d.Zero),
		PriceC:    g.D(tx.PriceC, d.Zero),
		Fee:       g.D(tx.Fee, d.Zero),
		FeeC:      g.D(tx.FeeC, d.Zero),
		Value:     g.D(tx.Value, d.Zero),
		ValueC:    g.D(tx.ValueC, d.Zero),
		AssetType: tx.AssetType,
		Ts:        tx.Ts.AsTime(),
	}
}

func NewEntryFromTransfer(tx *proto.Transfer) Entry {
	price := d.Zero
	priceC := d.Zero

	if tx.Action == proto.TransferAction_DEPOSIT || tx.Action == proto.TransferAction_WITHDRAWAL {
		price = g.D("1")
		priceC = g.D("1")
	}

	return Entry{
		Amount: g.D(tx.Amount, d.Zero),
		Fee:    g.D(tx.Fee, d.Zero),
		FeeC:   g.D(tx.FeeC, d.Zero),
		Value:  g.D(tx.Account, d.Zero),
		Price:  price,
		PriceC: priceC,
		Ts:     tx.Ts.AsTime(),
	}
}

func (e EntryList) Sort() {
	sort.Slice(e, func(i, j int) bool {
		return e[i].Ts.Before(e[j].Ts)
	})
}

type asset struct {
	name    string
	entries EntryList
	// baseDir baseDir
}

type Fifo struct {
	assets map[string]asset
}

func NewFifo() *Fifo {
	return &Fifo{
		assets: map[string]asset{},
	}
}

func (f *Fifo) Add(assetName string, e Entry) {
	if _, ok := f.assets[assetName]; !ok {
		f.assets[assetName] = asset{
			name:    assetName,
			entries: EntryList{},
			// baseDir: BASE_DIR_NONE,
		}
	}

	a := f.assets[assetName]
	a.entries = append(a.entries, e)
	a.entries.Sort()
	f.assets[assetName] = a
}

func (f *Fifo) Take(assetName string, amount d.Decimal) (EntryList, error) {
	a, ok := f.assets[assetName]

	if !ok {
		return nil, errors.New("no entry for this asset")
	}

	result := EntryList{}
	rest := amount.Copy()

	for {
		if len(a.entries) == 0 {
			if rest.GreaterThan(d.Zero) {
				return result, errors.New("incomplete")
			}
			break
		}

		oldest := a.entries[0]

		if rest.LessThanOrEqual(oldest.Amount) {
			r := oldest.Copy()
			r.Fee = r.Fee.Div(r.Amount).Mul(rest)
			r.FeeC = r.FeeC.Div(r.Amount).Mul(rest)
			r.Amount = rest
			r.Value = r.Amount.Mul(r.Price)
			r.ValueC = r.Amount.Mul(r.PriceC)
			fmt.Printf("%+v\n", oldest)
			fmt.Printf("%+v\n", r)
			result = append(result, r)

			newAmount := oldest.Amount.Sub(rest)
			oldest.Fee = oldest.Fee.Div(oldest.Amount).Mul(newAmount)
			oldest.Amount = newAmount
			a.entries[0] = oldest

			if oldest.Amount.Equal(d.Zero) {
				a.entries = a.entries[1:]
			}
			return result, nil
		} else {
			rest = rest.Sub(oldest.Amount)
			result = append(result, oldest.Copy())
			a.entries = a.entries[1:]
		}
	}

	return result, nil
}
