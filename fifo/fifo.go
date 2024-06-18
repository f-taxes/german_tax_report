package fifo

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	g "github.com/f-taxes/german_tax_report/global"
	d "github.com/shopspring/decimal"
)

const (
	ERR_NO_ENTRY = iota
	ERR_INSUFFICIENT
)

type FifoError struct {
	Code           int
	RequiredAmount string
	MissingAmount  string
	err            string
}

func (f FifoError) Error() string {
	return f.err
}

func NewFifoError(code int, err string) FifoError {
	return FifoError{
		Code: code,
		err:  err,
	}
}

func NewFifoTakeError(code int, err string, reqAmount, missingAmount string) FifoError {
	return FifoError{
		Code:           code,
		RequiredAmount: reqAmount,
		MissingAmount:  missingAmount,
		err:            err,
	}
}

func IsFifoErrorNoEntry(err error, code int) bool {
	if ferr, ok := err.(FifoError); ok {
		return ferr.Code == ERR_NO_ENTRY
	}

	return false
}

func IsFifoErrorInsufficient(err error, code int) bool {
	if ferr, ok := err.(FifoError); ok {
		return ferr.Code == ERR_INSUFFICIENT
	}

	return false
}

func IsFifoError(err error) bool {
	_, ok := err.(FifoError)
	return ok
}

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

func (e EntryList) TotalUnitsLeft() d.Decimal {
	s := d.Zero

	for i := range e {
		s = s.Add(e[i].UnitsLeft)
	}

	return s
}

func (e EntryList) Copy() EntryList {
	n := EntryList{}

	for i := range e {
		n = append(n, e[i].Copy())
	}

	return n
}

func NewEntry(amount, valueC, feeC d.Decimal, ts time.Time) Entry {
	return Entry{
		Units:          amount.Copy(),
		UnitsLeft:      amount.Copy(),
		UnitCostEur:    valueC.Div(amount).Round(4),
		UnitFeeCostEur: feeC.Div(amount).Round(4),
		Ts:             ts,
	}
}

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
	Total   string
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
			Total:   "0",
			Entries: EntryList{},
		}
	}

	a := f.assets[assetName]
	a.Entries = append(a.Entries, e)
	a.Entries.Sort()
	f.assets[assetName] = a
}

// func (f *Fifo) Prepend(assetName string, e Entry) {
// 	if _, ok := f.assets[assetName]; !ok {
// 		f.assets[assetName] = Asset{
// 			Name:    assetName,
// 			Entries: EntryList{},
// 		}
// 	}

// 	a := f.assets[assetName]
// 	a.Entries = append(EntryList{e}, a.Entries...)
// 	a.Entries.Sort()
// 	f.assets[assetName] = a
// }

func (f *Fifo) Read(assetName string) Asset {
	asset := f.assets[assetName]
	return Asset{
		Name:    asset.Name,
		Total:   asset.Entries.TotalUnitsLeft().String(),
		Entries: asset.Entries.Copy(),
	}
}

func (f *Fifo) HasUnits(assetName string) bool {
	a, ok := f.assets[assetName]

	if !ok {
		return false
	}

	if len(a.Entries) == 0 {
		return false
	}

	if f.getOldestAvailableEntry(a.Entries) == -1 {
		return false
	}

	return true
}

func (f *Fifo) CouldTake(assetName string, units d.Decimal, decimals int32) bool {
	_, err := f.take(assetName, units, true, decimals)
	return err == nil
}

func (f *Fifo) Take(assetName string, units d.Decimal, decimals int32) (Asset, error) {
	return f.take(assetName, units, false, decimals)
}

func (f *Fifo) take(assetName string, units d.Decimal, simulated bool, decimals int32) (Asset, error) {
	a, ok := f.assets[assetName]

	if !ok {
		return Asset{}, NewFifoError(ERR_NO_ENTRY, "no entry for this asset")
	}

	units = g.RoundIfFiat(assetName, units)

	entries := a.Entries

	if simulated {
		entries = a.Entries.Copy()
	}

	result := Asset{
		Name:    assetName,
		Total:   "0",
		Entries: EntryList{},
	}
	rest := units.Copy()

	for {
		if len(entries) == 0 {
			if rest.GreaterThan(d.Zero) {
				return result, errors.New("incomplete")
			}
			break
		}

		oldestIdx := f.getOldestAvailableEntry(entries)

		if oldestIdx == -1 {
			return result, NewFifoTakeError(ERR_NO_ENTRY, fmt.Sprintf("insufficient assets in fifo queue to take %s %s (rest=%s)", units, assetName, rest), units.String(), rest.String())
		}

		oldest := entries[oldestIdx]

		// rest = g.FixRoundingIfFiat(assetName, rest, oldest.UnitsLeft)

		if rest.LessThanOrEqual(oldest.UnitsLeft) {
			r := oldest.Copy()
			r.UnitsLeft = rest
			result.Entries = append(result.Entries, r)
			result.Total = result.Entries.TotalUnitsLeft().String()

			// oldest.UnitsLeft = g.RoundIfFiat(assetName, oldest.UnitsLeft.Sub(rest))
			oldest.UnitsLeft = oldest.UnitsLeft.Sub(rest).Round(decimals)
			entries[oldestIdx] = oldest

			return result, nil
		} else {
			rest = rest.Sub(oldest.UnitsLeft).Round(decimals)
			result.Entries = append(result.Entries, oldest.Copy())
			result.Total = result.Entries.TotalUnitsLeft().String()
			entries[oldestIdx].UnitsLeft = d.Zero
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
