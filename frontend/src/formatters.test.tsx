import { describe, expect, it } from "vitest";
import {
  collectFieldNames,
  countdownLabel,
  formatFieldValue,
  getOperationIDFromPath,
} from "./formatters";

describe("formatters", () => {
  it("extracts the operation ID from the approval path", () => {
    expect(getOperationIDFromPath("/approve/op_123")).toBe("op_123");
    expect(getOperationIDFromPath("/")).toBe("");
  });

  it("builds a stable countdown label", () => {
    expect(
      countdownLabel("2026-04-01T12:10:05Z", new Date("2026-04-01T12:09:00Z")),
    ).toBe("1m 05s");
    expect(
      countdownLabel("2026-04-01T12:09:00Z", new Date("2026-04-01T12:09:00Z")),
    ).toBe("expired");
  });

  it("renders a bare date as the same calendar day in timezones west of UTC", () => {
    // Tests run with TZ=America/New_York (see the test script); a UTC-midnight
    // parse would render 2026-07-24 as the 23rd.
    expect(formatFieldValue("2026-07-24")).toContain("24");
  });

  it("collects sorted unique field names from current and requested values", () => {
    expect(
      collectFieldNames(
        { status: "Planning", owner: "Ava" },
        { status: "Done", name: "Website" },
      ),
    ).toEqual(["name", "owner", "status"]);
  });
});
