import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { OnboardingWizard } from "./OnboardingWizard";

const pushMock = vi.fn();
const setUserMock = vi.fn();
const checkUsernameMock = vi.fn();
const completeOnboardingMock = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: pushMock }),
}));

vi.mock("@/features/auth/AuthProvider", () => ({
  useAuth: () => ({
    accessToken: "jwt-test",
    setUser: setUserMock,
  }),
}));

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>();
  return {
    ...actual,
    checkUsernameAvailable: (...args: unknown[]) => checkUsernameMock(...args),
    completeOnboarding: (...args: unknown[]) => completeOnboardingMock(...args),
  };
});

describe("OnboardingWizard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("walks through all 3 onboarding steps and submits the profile", async () => {
    const user = userEvent.setup();
    checkUsernameMock.mockResolvedValue({ username: "sam_walks", available: true });
    completeOnboardingMock.mockResolvedValueOnce({
      onboarding_completed: true,
      user: {
        id: "u1",
        email: "sam@example.com",
        username: "sam_walks",
        onboarding_completed: true,
        created_at: new Date().toISOString(),
      },
    });

    render(<OnboardingWizard />);

    // Step 1: Profile
    await user.type(screen.getByLabelText(/username/i), "sam_walks");
    await user.type(screen.getByLabelText(/display name/i), "Sam Walker");
    await user.click(screen.getByRole("button", { name: /next: privacy & goals/i }));

    // Step 2: Privacy & goals
    await waitFor(() => {
      expect(screen.getByRole("heading", { name: /your weight privacy/i })).toBeInTheDocument();
    });
    await user.type(screen.getByLabelText(/starting weight/i), "102");
    await user.type(screen.getByLabelText(/goal weight/i), "85");
    await user.click(screen.getByRole("button", { name: /next: quest focus/i }));

    // Step 3: Quest focus
    await waitFor(() => {
      expect(screen.getByRole("heading", { name: /what habits do you want to build/i })).toBeInTheDocument();
    });
    await user.click(screen.getByRole("button", { name: /complete setup/i }));

    await waitFor(() => {
      expect(completeOnboardingMock).toHaveBeenCalledWith(
        expect.objectContaining({
          username: "sam_walks",
          display_name: "Sam Walker",
          unit_pref: "kg",
          start_weight_grams: 102000,
          goal_weight_grams: 85000,
          weight_privacy: "squads_only",
        }),
        "jwt-test",
      );
      expect(pushMock).toHaveBeenCalledWith("/dashboard");
    });
  });
});
