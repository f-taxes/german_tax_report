let EURFormatter = new Intl.NumberFormat('de-DE', {
  maximumSignificantDigits: 5,
  minimumSignificantDigits: 3,
});

export const formatEur = val => EURFormatter.format(val);
