// formatAmount shows an exact decimal amount from the API, such as "1200.5",
// as an amount of currencyCode with precision decimal places, in the
// reader's locale unless one is given. Intl formats the decimal string
// itself, so no digit is lost to a JavaScript number. A code Intl does not
// accept as a currency, such as a four-letter token, goes after the number.
export function formatAmount(amount: string, currencyCode: string, precision?: number, locale?: string): string {
  const digits = precision === undefined ? {} : { minimumFractionDigits: precision, maximumFractionDigits: precision }
  const value = amount as Intl.StringNumericLiteral
  try {
    return new Intl.NumberFormat(locale, { style: "currency", currency: currencyCode, ...digits }).format(value)
  } catch {
    return `${new Intl.NumberFormat(locale, digits).format(value)} ${currencyCode}`
  }
}
