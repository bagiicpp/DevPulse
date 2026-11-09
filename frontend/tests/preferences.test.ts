import assert from "node:assert/strict"
import { afterEach, beforeEach, test } from "node:test"
import {
  isRepositoryName,
  readDefaultRepository,
  saveDefaultRepository,
} from "../src/lib/preferences.ts"

const original = Object.getOwnPropertyDescriptor(globalThis, "localStorage")
let values: Map<string, string>

beforeEach(() => {
  values = new Map()
  Object.defineProperty(globalThis, "localStorage", {
    configurable: true,
    value: {
      getItem: (key: string) => values.get(key) ?? null,
      setItem: (key: string, value: string) => values.set(key, value),
    },
  })
})

afterEach(() => {
  if (original) Object.defineProperty(globalThis, "localStorage", original)
  else Reflect.deleteProperty(globalThis, "localStorage")
})

test("accepts owner/repository names and rejects URLs, missing names, and oversized input", () => {
  for (const value of ["octocat/Hello-World", "example/project_name.js"]) {
    assert.equal(isRepositoryName(value), true)
  }
  for (const value of ["", "owner", "owner/", "/repo", "https://github.com/owner/repo", "owner/repo?x=1", "owner/" + "a".repeat(101)]) {
    assert.equal(isRepositoryName(value), false)
  }
})

test("saves, reads, and clears the browser's default repository", () => {
  assert.equal(readDefaultRepository(), "")
  saveDefaultRepository("owner/repository")
  assert.equal(readDefaultRepository(), "owner/repository")
  saveDefaultRepository("")
  assert.equal(readDefaultRepository(), "")
})

test("ignores invalid stored values and refuses to overwrite a valid preference with invalid input", () => {
  values.set("devpulse.defaultRepository", "<script>")
  assert.equal(readDefaultRepository(), "")
  saveDefaultRepository("owner/repository")
  assert.throws(() => saveDefaultRepository("not a repository"))
  assert.equal(readDefaultRepository(), "owner/repository")
})

test("allows reading when storage is blocked and surfaces a failed save", () => {
  Object.defineProperty(globalThis, "localStorage", {
    configurable: true,
    get() { throw new Error("Storage blocked") },
  })
  assert.equal(readDefaultRepository(), "")
  assert.throws(() => saveDefaultRepository("owner/repository"), /Storage blocked/)
})
