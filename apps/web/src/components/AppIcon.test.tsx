import { FireIcon } from "@hugeicons/core-free-icons";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AppIcon } from "./AppIcon";

describe("AppIcon", () => {
  it("renders decorative icons as aria-hidden SVG", () => {
    const { container } = render(<AppIcon icon={FireIcon} size={24} />);
    const svg = container.querySelector("svg");
    expect(svg).toBeInTheDocument();
    expect(svg).toHaveAttribute("aria-hidden", "true");
  });

  it("renders labeled icons with accessible img role", () => {
    render(<AppIcon icon={FireIcon} label="Daily streak flame" />);
    expect(screen.getByRole("img", { name: "Daily streak flame" })).toBeInTheDocument();
  });
});
