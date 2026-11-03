import { describe, expect, it } from "vitest"

import { addDecimals, compareDecimals, multiplyDecimals, negate } from "./decimal"

describe("addDecimals", () => {
  const cases: { name: string; a: string; b: string; want: string }[] = [
    { name: "whole numbers", a: "1200", b: "34", want: "1234" },
    { name: "different numbers of decimals", a: "1200.5", b: "0.25", want: "1200.75" },
    { name: "trailing zeros dropped", a: "0.75", b: "0.25", want: "1" },
    { name: "a negative result", a: "10", b: "-10.01", want: "-0.01" },
    { name: "a negative fraction", a: "-0.5", b: "-0.5", want: "-1" },
    { name: "zero", a: "-3.5", b: "3.5", want: "0" },
    { name: "more digits than a JavaScript number keeps", a: "12345678901234567.89", b: "0.01", want: "12345678901234567.9" },
  ]

  for (const { name, a, b, want } of cases) {
    it(name, () => {
      expect(addDecimals(a, b)).toBe(want)
    })
  }
})

describe("multiplyDecimals", () => {
  const cases: { name: string; a: string; b: string; want: string }[] = [
    { name: "whole numbers", a: "12", b: "3", want: "36" },
    { name: "a fractional share at a price", a: "12.5", b: "231.45", want: "2893.125" },
    { name: "trailing zeros dropped", a: "0.5", b: "0.2", want: "0.1" },
    { name: "a product under one", a: "0.01", b: "0.01", want: "0.0001" },
    { name: "a negative factor", a: "-2", b: "1.5", want: "-3" },
    { name: "zero", a: "0", b: "231.45", want: "0" },
    { name: "more digits than a JavaScript number keeps", a: "12345678901234567", b: "1.1", want: "13580246791358023.7" },
  ]

  for (const { name, a, b, want } of cases) {
    it(name, () => {
      expect(multiplyDecimals(a, b)).toBe(want)
    })
  }
})

describe("negate", () => {
  const cases: { name: string; amount: string; want: string }[] = [
    { name: "a positive amount", amount: "1.5", want: "-1.5" },
    { name: "a negative amount", amount: "-1.5", want: "1.5" },
  ]

  for (const { name, amount, want } of cases) {
    it(name, () => {
      expect(negate(amount)).toBe(want)
    })
  }
})

describe("compareDecimals", () => {
  const cases: { name: string; a: string; b: string; want: number }[] = [
    { name: "less", a: "9.99", b: "10", want: -1 },
    { name: "more", a: "10.01", b: "10", want: 1 },
    { name: "equal, written differently", a: "10.50", b: "10.5", want: 0 },
    { name: "a negative against zero", a: "-0.01", b: "0", want: -1 },
  ]

  for (const { name, a, b, want } of cases) {
    it(name, () => {
      expect(compareDecimals(a, b)).toBe(want)
    })
  }
})
