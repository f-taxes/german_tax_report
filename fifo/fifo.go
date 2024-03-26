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

type entry struct {
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

func (e *entry) Copy() entry {
	return entry{
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

type entryList []entry

func NewEntryFromTx(tx *proto.Trade) entry {
	return entry{
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

func (e entryList) Sort() {
	sort.Slice(e, func(i, j int) bool {
		return e[i].Ts.Before(e[j].Ts)
	})
}

type asset struct {
	name    string
	entries entryList
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

func (f *Fifo) Add(assetName string, e entry) {
	if _, ok := f.assets[assetName]; !ok {
		f.assets[assetName] = asset{
			name:    assetName,
			entries: entryList{},
			// baseDir: BASE_DIR_NONE,
		}
	}

	a := f.assets[assetName]
	a.entries = append(a.entries, e)
	a.entries.Sort()
	f.assets[assetName] = a
}

func (f *Fifo) Take(assetName string, amount d.Decimal) (entryList, error) {
	a, ok := f.assets[assetName]

	if !ok {
		return nil, errors.New("no entry for this asset")
	}

	result := entryList{}
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
