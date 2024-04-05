package reporting

import (
	"fmt"
	"time"

	"github.com/f-taxes/german_tax_report/fifo"
)

type DescriptionMap map[string][]Description

func (d DescriptionMap) Add(accountName string, description Description) {
	if _, ok := d[accountName]; !ok {
		d[accountName] = []Description{}
	}

	d[accountName] = append(d[accountName], description)
}

type Description interface {
	String() string
}

// type add struct {
// 	Asset   string
// 	Entries fifo.EntryList
// 	Ts      time.Time
// }

// func (d add) String() string {
// 	return fmt.Sprintf("%s: Add %s %s", d.Ts, d.Entries, d.Asset)
// }

type move struct {
	Asset       string
	Source      string
	Destination string
	Entries     fifo.EntryList
	Ts          time.Time
}

func (d move) String() string {
	return fmt.Sprintf("%s: Moved %s %s from %s to %s", d.Ts, d.Entries.TotalUnits(), d.Asset, d.Source, d.Destination)
}

type convert struct {
	Asset   string
	Quote   string
	Entries fifo.EntryList
	ValueC  string
	Pnl     string
	Ts      time.Time
}

func (d convert) String() string {
	return fmt.Sprintf("%s: Convert %s %s worth %s€ to %s, generating a pnl of %s", d.Ts, d.Entries.TotalUnits(), d.Asset, d.ValueC, d.Quote, d.Pnl)
}
