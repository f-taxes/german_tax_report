package reporting

import "github.com/f-taxes/german_tax_report/fifo"

type AccountFifo map[string]*fifo.Fifo

func (a AccountFifo) Get(accountName string) *fifo.Fifo {
	if f, ok := a[accountName]; ok {
		return f
	}

	f := fifo.NewFifo()
	a[accountName] = f
	return f
}
