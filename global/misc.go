package global

func RemoveElementUnordered[T any](a []T, i int) []T {
	a[i] = a[len(a)-1]    // Copy last element to index i.
	a[len(a)-1] = *new(T) // Erase last element (write zero value).
	a = a[:len(a)-1]
	return a
}

// Using generics return either the then or else value.
func IfThen[T any](cond bool, thenVal, elseVal T) T {
	if cond {
		return thenVal
	}
	return elseVal
}

func IsFiatCurrency(currency string) bool {
	return FiatCurrencies[currency]
}
