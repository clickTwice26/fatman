import { describe, expect, it } from "vitest";
import { FONT_FAMILY_ONEST, fatmanTheme } from "./theme";

describe("fatmanTheme", () => {
  it("binds the Onest font CSS variable into MUI typography", () => {
    expect(fatmanTheme.typography.fontFamily).toBe(FONT_FAMILY_ONEST);
    expect(fatmanTheme.typography.fontFamily).toContain("--font-onest");
  });

  it("configures CSS variables for light and dark color schemes", () => {
    expect(fatmanTheme.vars).toBeDefined();
    expect(fatmanTheme.vars?.palette.primary.main).toBeDefined();
  });
});
