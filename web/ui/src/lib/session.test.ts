import { describe, expect, it } from "vitest"

import { signInRedirect } from "./session"

describe("signInRedirect", () => {
  const cases: { name: string; status: number; want: string | undefined }[] = [
    { name: "no session", status: 401, want: "/auth/login" },
    { name: "a success", status: 200, want: undefined },
    { name: "a request from another site", status: 403, want: undefined },
    { name: "a missing resource", status: 404, want: undefined },
    { name: "a gateway error", status: 500, want: undefined },
  ]

  for (const { name, status, want } of cases) {
    it(name, () => {
      expect(signInRedirect(status)).toBe(want)
    })
  }
})
