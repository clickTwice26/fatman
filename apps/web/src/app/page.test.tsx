import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import HomePage from "./page";

describe("HomePage", () => {
  it("renders the community value proposition and email onboarding CTA", () => {
    render(<HomePage />);

    expect(
      screen.getByRole("heading", { name: /lose weight together, one daily quest at a time/i }),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /continue with email/i })).toHaveAttribute("href", "/auth");
    expect(screen.getByText(/collaborative weight-loss quests/i)).toBeInTheDocument();
    expect(screen.getByText(/privacy-first progress tracking/i)).toBeInTheDocument();
  });
});
