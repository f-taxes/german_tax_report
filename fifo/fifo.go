package fifo

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	d "github.com/shopspring/decimal"
)

// type baseDir int

// const (
// 	BASE_DIR_NONE = baseDir(iota)
// 	BASE_DIR_LONG
// 	BASE_DIR_SHORT
// )

type Entry struct {
	Units          d.Decimal
	UnitsLeft      d.Decimal
	UnitCostEur    d.Decimal
	UnitFeeCostEur d.Decimal
	Ts             time.Time
}

func (e *Entry) Copy() Entry {
	return Entry{
		Units:          e.Units.Copy(),
		UnitsLeft:      e.UnitsLeft.Copy(),
		UnitCostEur:    e.UnitCostEur.Copy(),
		UnitFeeCostEur: e.UnitFeeCostEur.Copy(),
		Ts:             e.Ts,
	}
}

type EntryList []Entry

func (e EntryList) TotalUnits() d.Decimal {
	s := d.Zero

	for i := range e {
		s = s.Add(e[i].Units)
	}

	return s
}

func (e EntryList) TotalFeeEur() d.Decimal {
	s := d.Zero

	for i := range e {
		s = s.Add(e[i].UnitFeeCostEur)
	}

	return s
}

func NewEntryFromTrade(amount, valueC, feeC d.Decimal, ts time.Time) Entry {
	return Entry{
		Units:          amount.Copy(),
		UnitsLeft:      amount.Copy(),
		UnitCostEur:    valueC.Div(amount),
		UnitFeeCostEur: feeC.Div(amount),
		Ts:             ts,
	}
}

// func NewEntryFromTransfer(tx *proto.Transfer) Entry {
// 	price := d.Zero
// 	priceC := d.Zero

// 	if tx.Action == proto.TransferAction_DEPOSIT || tx.Action == proto.TransferAction_WITHDRAWAL {
// 		price = g.D("1")
// 		priceC = g.D("1")
// 	}

// 	return Entry{
// 		Amount: g.D(tx.Amount, d.Zero),
// 		CostEur: ,
// 		Fee:    g.D(tx.Fee, d.Zero),
// 		FeeC:   g.D(tx.FeeC, d.Zero),
// 		Price:  price,
// 		PriceC: priceC,
// 		Ts:     tx.Ts.AsTime(),
// 	}
// }

func (e EntryList) Sort() {
	sort.Slice(e, func(i, j int) bool {
		return e[i].Ts.Before(e[j].Ts)
	})
}

func (e EntryList) Print() string {
	out := []string{}
	for _, r := range e {
		out = append(out, fmt.Sprintf("%s | %s (%s left) x %s€ (Fee Total: %s€)", r.Ts, r.Units, r.UnitsLeft, r.UnitCostEur, r.UnitFeeCostEur.Mul(r.Units)))
	}

	return strings.Join(out, "\n")
}

type Asset struct {
	Name    string
	Entries EntryList
	// baseDir baseDir
}

type Fifo struct {
	assets map[string]Asset
}

func NewFifo() *Fifo {
	return &Fifo{
		assets: map[string]Asset{},
	}
}

func (f *Fifo) Print() string {
	out := []string{}
	for asset, a := range f.assets {
		out = append(out, fmt.Sprintf("\nFIFO queue for %s:\n", asset))
		out = append(out, a.Entries.Print())
	}

	return strings.Join(out, "\n")
}

func (f *Fifo) Add(assetName string, e Entry) {
	if _, ok := f.assets[assetName]; !ok {
		f.assets[assetName] = Asset{
			Name:    assetName,
			Entries: EntryList{},
			// baseDir: BASE_DIR_NONE,
		}
	}

	a := f.assets[assetName]
	a.Entries = append(a.Entries, e)
	a.Entries.Sort()
	f.assets[assetName] = a
}

func (f *Fifo) Read(assetName string) Asset {
	return f.assets[assetName]
}

func (f *Fifo) Take(assetName string, units d.Decimal) (EntryList, error) {
	a, ok := f.assets[assetName]

	if !ok {
		return nil, errors.New("no entry for this asset")
	}

	result := EntryList{}
	rest := units.Copy()

	for {
		if len(a.Entries) == 0 {
			if rest.GreaterThan(d.Zero) {
				return result, errors.New("incomplete")
			}
			break
		}

		oldestIdx := f.getOldestAvailableEntry(a.Entries)

		if oldestIdx == -1 {
			return result, fmt.Errorf("insufficient assets in fifo queue to take %s %s", units, assetName)
		}

		oldest := a.Entries[oldestIdx]

		if rest.LessThanOrEqual(oldest.UnitsLeft) {
			r := oldest.Copy()
			r.UnitsLeft = rest
			result = append(result, r)

			oldest.UnitsLeft = oldest.UnitsLeft.Sub(rest)
			a.Entries[oldestIdx] = oldest

			f.assets[assetName] = a
			return result, nil
		} else {
			rest = rest.Sub(oldest.UnitsLeft)
			result = append(result, oldest.Copy())
			a.Entries[oldestIdx].UnitsLeft = d.Zero
			f.assets[assetName] = a
		}
	}

	return result, nil
}

func (f *Fifo) getOldestAvailableEntry(entries EntryList) int {
	for i := range entries {
		if entries[i].UnitsLeft.GreaterThan(d.Zero) {
			return i
		}
	}

	return -1
}
