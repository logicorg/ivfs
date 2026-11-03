import { describe, expect, it } from "vitest"

import { problemMessages } from "./problem"

describe("problemMessages", () => {
  const cases: { name: string; error: unknown; want: string[] }[] = [
    {
      name: "a workflow's error, with its detail",
      error: { title: "Conflict", status: 409, detail: "a Currency with this name already exists" },
      want: ["a Currency with this name already exists"],
    },
    {
      name: "Huma's validation, with each invalid field",
      error: {
        title: "Unprocessable Entity",
        status: 422,
        detail: "validation failed",
        errors: [
          { location: "body.name", message: "expected required property name to be present" },
          { message: "unexpected property" },
          { location: "body.code" },
        ],
      },
      want: ["validation failed", "body.name: expected required property name to be present", "unexpected property"],
    },
    {
      name: "a problem with only a title",
      error: { title: "Not Found", status: 404 },
      want: ["Not Found"],
    },
    {
      name: "a network failure",
      error: new TypeError("Failed to fetch"),
      want: ["The gateway did not answer. Is it running?"],
    },
    {
      name: "a body that is not a problem",
      error: "Bad Gateway",
      want: ["The gateway did not answer. Is it running?"],
    },
  ]

  for (const { name, error, want } of cases) {
    it(name, () => {
      expect(problemMessages(error)).toEqual(want)
    })
  }
})
