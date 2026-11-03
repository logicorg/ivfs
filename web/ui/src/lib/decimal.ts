// addDecimals adds two decimal strings, such as "1200.5" and "-0.25",
// exactly: both are scaled to whole numbers as BigInts, so no digit is lost
// to a JavaScript number.
export function addDecimals(a: string, b: string): string {
  const scale = Math.max(fractionDigits(a), fractionDigits(b))
  const sum = toScaled(a, scale) + toScaled(b, scale)
  return fromScaled(sum, scale)
}

// multiplyDecimals multiplies two decimal strings, such as units by a price,
// exactly, keeping every digit of the product.
export function multiplyDecimals(a: string, b: string): string {
  const scaleA = fractionDigits(a)
  const scaleB = fractionDigits(b)
  return fromScaled(toScaled(a, scaleA) * toScaled(b, scaleB), scaleA + scaleB)
}

// negate is amount with its sign flipped.
export function negate(amount: string): string {
  return amount.startsWith("-") ? amount.slice(1) : `-${amount}`
}

function fractionDigits(amount: string): number {
  const point = amount.indexOf(".")
  return point === -1 ? 0 : amount.length - point - 1
}

function toScaled(amount: string, scale: number): bigint {
  const [whole, fraction = ""] = amount.split(".")
  const sign = whole.startsWith("-") ? -1n : 1n
  const digits = whole.replace(/^[-+]/, "") + fraction.padEnd(scale, "0")
  return sign * BigInt(digits)
}

function fromScaled(value: bigint, scale: number): string {
  const sign = value < 0n ? "-" : ""
  const digits = (value < 0n ? -value : value).toString().padStart(scale + 1, "0")
  if (scale === 0) {
    return sign + digits
  }
  const fraction = digits.slice(-scale).replace(/0+$/, "")
  const whole = digits.slice(0, -scale)
  return sign + whole + (fraction ? `.${fraction}` : "")
}

// compareDecimals orders two decimal strings by value, exactly: negative when
// a is less than b, positive when it is more, and zero when they are equal.
export function compareDecimals(a: string, b: string): number {
  const difference = addDecimals(a, negate(b))
  return difference === "0" ? 0 : difference.startsWith("-") ? -1 : 1
}
