package reporting

import "github.com/f-taxes/german_tax_report/fifo"

type AccountFifo map[string]*FifoGroup

type FifoGroup struct {
	G *fifo.Fifo            // The global fifo queue for physical holdings.
	I map[string]*fifo.Fifo // Holds a map of "independent" fifo queues. Used for margin trades for example.
}

// Get the accounts global fifo queue for physical holdings or for a sub category to track margin trades for example.
func (a AccountFifo) Get(accountName string, subCategory ...string) *fifo.Fifo {
	if _, ok := a[accountName]; !ok {
		a[accountName] = &FifoGroup{
			G: fifo.NewFifo(),
			I: map[string]*fifo.Fifo{},
		}
	}

	if len(subCategory) > 0 {
		if _, ok := a[accountName].I[subCategory[0]]; !ok {
			a[accountName].I[subCategory[0]] = fifo.NewFifo()
		}

		return a[accountName].I[subCategory[0]]
	}

	return a[accountName].G
}

func (a AccountFifo) SetIsShortTrade(accountName string, subCategory string, status bool) {
	if _, ok := a[accountName]; !ok {
		a[accountName] = &FifoGroup{
			G: fifo.NewFifo(),
			I: map[string]*fifo.Fifo{},
		}
	}

	if len(subCategory) > 0 {
		if _, ok := a[accountName].I[subCategory]; !ok {
			a[accountName].I[subCategory] = fifo.NewFifo()
		}
	}
}
