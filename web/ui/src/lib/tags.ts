import * as z from "zod"

// tagIdsField is the tags a form gives its entry, by id.
export const tagIdsField = z.array(z.number())

// toggleTag is ids with id added, or taken out when it is already there. The
// order the person chose them in is kept, as the gateway sorts them anyway.
export function toggleTag(ids: number[], id: number): number[] {
  return ids.includes(id) ? ids.filter((other) => other !== id) : [...ids, id]
}

// sentTagIds is ids as a record form sends them: left out when there are none,
// so a request without tags reads as one.
export function sentTagIds(ids: number[]): number[] | undefined {
  return ids.length > 0 ? ids : undefined
}
