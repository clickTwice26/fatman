import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { EmailOtpForm } from "./EmailOtpForm";

const pushMock = vi.fn();
const setSessionMock = vi.fn();
const requestEmailOtpMock = vi.fn();
const verifyEmailOtpMock = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: pushMock }),
}));

vi.mock("@/features/auth/AuthProvider", () => ({
  useAuth: () => ({ setSession: setSessionMock }),
}));

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>();
  return {
    ...actual,
    requestEmailOtp: (...args: unknown[]) => requestEmailOtpMock(...args),
    verifyEmailOtp: (...args: unknown[]) => verifyEmailOtpMock(...args),
  };
});

describe("EmailOtpForm", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("validates email on submit and transitions to single-field OTP verification", async () => {
    const user = userEvent.setup();
    requestEmailOtpMock.mockResolvedValueOnce({
      message: "verification code sent",
      expires_in_seconds: 600,
      resend_after_seconds: 60,
    });
    verifyEmailOtpMock.mockResolvedValueOnce({
      access_token: "jwt-123",
      is_new_user: true,
      onboarding_completed: false,
      user: {
        id: "u1",
        email: "alex@example.com",
        onboarding_completed: false,
        created_at: new Date().toISOString(),
      },
    });

    render(<EmailOtpForm />);

    // Submitting invalid email shows clear validation message
    await user.type(screen.getByLabelText(/email address/i), "not-an-email");
    await user.click(screen.getByRole("button", { name: /send 6-digit code/i }));
    expect(screen.getByText(/you@example\.com/i)).toBeInTheDocument();
    expect(requestEmailOtpMock).not.toHaveBeenCalled();

    // Fix email and request OTP
    await user.clear(screen.getByLabelText(/email address/i));
    await user.type(screen.getByLabelText(/email address/i), "Alex@Example.com");
    await user.click(screen.getByRole("button", { name: /send 6-digit code/i }));

    await waitFor(() => {
      expect(screen.getByRole("heading", { name: /check your inbox/i })).toBeInTheDocument();
    });
    expect(requestEmailOtpMock).toHaveBeenCalledWith("alex@example.com");

    const otpInput = screen.getByLabelText(/6-digit verification code/i);
    expect(otpInput).toHaveAttribute("inputmode", "numeric");
    expect(otpInput).toHaveAttribute("autocomplete", "one-time-code");

    await user.type(otpInput, "482910");
    await user.click(screen.getByRole("button", { name: /verify & continue/i }));

    await waitFor(() => {
      expect(verifyEmailOtpMock).toHaveBeenCalledWith("alex@example.com", "482910");
      expect(setSessionMock).toHaveBeenCalled();
      expect(pushMock).toHaveBeenCalledWith("/onboarding");
    });
  });
});
