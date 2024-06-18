package global

import "github.com/shopspring/decimal"

func D(str string, defValue ...decimal.Decimal) decimal.Decimal {
	d, err := decimal.NewFromString(str)
	if err != nil {
		if len(defValue) > 0 {
			return defValue[0]
		}
		return decimal.Zero
	}

	return d
}

func PercentageDelta(n1, n2 decimal.Decimal) decimal.Decimal {
	return n1.Sub(n2).Abs().Div(n2).Mul(decimal.NewFromInt(100))
}

func RoundIfFiat(currency string, v decimal.Decimal) decimal.Decimal {
	switch currency {
	case "USD", "EUR":
		v = v.Round(4)
		if v.LessThanOrEqual(D("0.0001")) {
			return decimal.Zero
		}
		return v
	default:
		return v
	}
}

func FixRoundingIfFiat(currency string, v, should decimal.Decimal) decimal.Decimal {
	switch currency {
	case "USD", "EUR":
		r := v.Sub(should).Abs()
		if r.LessThanOrEqual(D("0.0001")) {
			return should
		}
		return v
	default:
		return v
	}
}
