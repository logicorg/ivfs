import { describe, expect, it } from "vitest"

import { sentTagIds, toggleTag } from "./tags"

describe("toggleTag", () => {
  const cases: { name: string; ids: number[]; id: number; want: number[] }[] = [
    { name: "adds a tag to none", ids: [], id: 3, want: [3] },
    { name: "adds a tag after the others", ids: [5, 1], id: 3, want: [5, 1, 3] },
    { name: "takes out a tag already there", ids: [5, 3, 1], id: 3, want: [5, 1] },
    { name: "takes out the only tag", ids: [3], id: 3, want: [] },
  ]
  for (const { name, ids, id, want } of cases) {
    it(name, () => {
      expect(toggleTag(ids, id)).toEqual(want)
    })
  }
})

describe("sentTagIds", () => {
  const cases: { name: string; ids: number[]; want: number[] | undefined }[] = [
    { name: "none is left out", ids: [], want: undefined },
    { name: "some are sent as they are", ids: [5, 3], want: [5, 3] },
  ]
  for (const { name, ids, want } of cases) {
    it(name, () => {
      expect(sentTagIds(ids)).toEqual(want)
    })
  }
})
