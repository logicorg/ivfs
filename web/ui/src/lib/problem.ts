import type { ErrorModel } from "@/api"

// problemMessages turns an error from the API into the messages to show: the
// problem's detail, then each invalid field Huma lists, with where it is.
// Anything that is not a problem, such as the gateway being unreachable, gets
// one generic message.
export function problemMessages(error: unknown): string[] {
  if (!isProblem(error)) {
    return ["The gateway did not answer. Is it running?"]
  }

  const messages = error.detail ? [error.detail] : [error.title ?? "The request failed."]
  for (const field of error.errors ?? []) {
    if (field.message) {
      messages.push(field.location ? `${field.location}: ${field.message}` : field.message)
    }
  }
  return messages
}

function isProblem(error: unknown): error is ErrorModel {
  return (
    typeof error === "object" &&
    error !== null &&
    ("detail" in error || "title" in error) &&
    typeof (error as ErrorModel).status === "number"
  )
}
