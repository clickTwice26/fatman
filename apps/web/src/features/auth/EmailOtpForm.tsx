"use client";

import { useEffect, useRef, useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import {
  ArrowLeft01Icon,
  ArrowRight01Icon,
  CheckmarkCircle02Icon,
  Clock01Icon,
  Mail01Icon,
  RefreshIcon,
} from "@hugeicons/core-free-icons";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Card from "@mui/material/Card";
import CardContent from "@mui/material/CardContent";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { AppIcon } from "@/components/AppIcon";
import { useAuth } from "@/features/auth/AuthProvider";
import { emailSchema, otpCodeSchema } from "@/features/auth/schemas";
import { ApiError, requestEmailOtp, verifyEmailOtp } from "@/lib/api";

export function EmailOtpForm() {
  const router = useRouter();
  const { setSession } = useAuth();

  const [step, setStep] = useState<"email" | "code">("email");
  const [email, setEmail] = useState("");
  const [emailError, setEmailError] = useState<string | null>(null);
  const [emailTouched, setEmailTouched] = useState(false);

  const [code, setCode] = useState("");
  const [codeError, setCodeError] = useState<string | null>(null);
  const [codeTouched, setCodeTouched] = useState(false);

  const [formError, setFormError] = useState<string | null>(null);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [resendCooldown, setResendCooldown] = useState(0);

  const codeHeadingRef = useRef<HTMLHeadingElement>(null);

  useEffect(() => {
    if (resendCooldown <= 0) return;
    const timer = window.setInterval(() => {
      setResendCooldown((prev) => (prev <= 1 ? 0 : prev - 1));
    }, 1000);
    return () => window.clearInterval(timer);
  }, [resendCooldown]);

  useEffect(() => {
    if (step === "code") {
      codeHeadingRef.current?.focus();
    }
  }, [step]);

  const validateEmailField = (value: string): string | null => {
    const parsed = emailSchema.safeParse(value);
    return parsed.success ? null : (parsed.error.issues[0]?.message ?? "Invalid email.");
  };

  const validateCodeField = (value: string): string | null => {
    const parsed = otpCodeSchema.safeParse(value);
    return parsed.success ? null : (parsed.error.issues[0]?.message ?? "Invalid code.");
  };

  const handleRequestOtp = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setFormError(null);
    setEmailTouched(true);
    const err = validateEmailField(email);
    setEmailError(err);
    if (err) return;

    setSubmitting(true);
    try {
      const normalized = email.trim().toLowerCase();
      const res = await requestEmailOtp(normalized);
      setEmail(normalized);
      setResendCooldown(res.resend_after_seconds);
      setStatusMessage(`We sent a 6-digit verification code to ${normalized}.`);
      setStep("code");
    } catch (error) {
      if (error instanceof ApiError) {
        setFormError(error.message);
      } else {
        setFormError("Unable to reach Fatman API. Please try again.");
      }
    } finally {
      setSubmitting(false);
    }
  };

  const handleVerifyOtp = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setFormError(null);
    setCodeTouched(true);
    const err = validateCodeField(code);
    setCodeError(err);
    if (err) return;

    setSubmitting(true);
    try {
      const res = await verifyEmailOtp(email, code.trim());
      setSession(res.access_token, res.user);
      router.push(res.onboarding_completed ? "/dashboard" : "/onboarding");
    } catch (error) {
      if (error instanceof ApiError) {
        setFormError(error.message);
      } else {
        setFormError("Verification failed. Please try again.");
      }
    } finally {
      setSubmitting(false);
    }
  };

  const handleResend = async () => {
    if (resendCooldown > 0 || submitting) return;
    setFormError(null);
    setSubmitting(true);
    try {
      const res = await requestEmailOtp(email);
      setResendCooldown(res.resend_after_seconds);
      setStatusMessage(`A fresh 6-digit code was sent to ${email}.`);
    } catch (error) {
      if (error instanceof ApiError) {
        setFormError(error.message);
      } else {
        setFormError("Unable to resend code right now.");
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Card sx={{ maxWidth: 460, width: "100%", mx: "auto" }}>
      <CardContent sx={{ p: { xs: 3, sm: 4 } }}>
        {step === "email" ? (
          <Box component="form" method="POST" onSubmit={handleRequestOtp} noValidate>
            <Stack spacing={3}>
              <Stack spacing={1}>
                <Box
                  sx={{
                    width: 48,
                    height: 48,
                    borderRadius: 3,
                    bgcolor: "primary.main",
                    color: "primary.contrastText",
                    display: "inline-flex",
                    alignItems: "center",
                    justifyContent: "center",
                  }}
                >
                  <AppIcon icon={Mail01Icon} size={24} />
                </Box>
                <Typography variant="h4" component="h1">
                  Join or sign in with email
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  No password to remember — we&apos;ll send a one-time 6-digit verification code to your inbox.
                </Typography>
              </Stack>

              {formError && (
                <Alert severity="error" role="alert">
                  {formError}
                </Alert>
              )}

              <TextField
                id="auth-email"
                name="email"
                type="email"
                label="Email address"
                required
                fullWidth
                value={email}
                error={Boolean(emailError)}
                helperText={emailError ?? "Format: you@example.com"}
                slotProps={{
                  htmlInput: {
                    autoComplete: "email",
                    autoCorrect: "off",
                    autoCapitalize: "none",
                    spellCheck: false,
                    enterKeyHint: "send",
                    "aria-invalid": Boolean(emailError),
                  },
                }}
                onChange={(e) => {
                  const next = e.target.value;
                  setEmail(next);
                  if (emailError && validateEmailField(next) === null) {
                    setEmailError(null);
                  }
                }}
                onBlur={() => {
                  if (email.length > 0 || emailTouched) {
                    setEmailTouched(true);
                    setEmailError(validateEmailField(email));
                  }
                }}
              />

              <Button
                type="submit"
                variant="contained"
                size="large"
                fullWidth
                disabled={submitting}
                endIcon={<AppIcon icon={ArrowRight01Icon} size={18} />}
              >
                {submitting ? "Sending code..." : "Send 6-digit code"}
              </Button>
            </Stack>
          </Box>
        ) : (
          <Box component="form" method="POST" onSubmit={handleVerifyOtp} noValidate>
            <Stack spacing={3}>
              <Stack spacing={1}>
                <Typography
                  ref={codeHeadingRef}
                  tabIndex={-1}
                  variant="h4"
                  component="h1"
                  sx={{ outline: "none" }}
                >
                  Check your inbox
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  Enter the 6-digit code sent to <strong>{email}</strong>.
                </Typography>
              </Stack>

              {statusMessage && (
                <Alert
                  severity="success"
                  icon={<AppIcon icon={CheckmarkCircle02Icon} size={20} />}
                  role="status"
                >
                  {statusMessage}
                </Alert>
              )}

              {formError && (
                <Alert severity="error" role="alert">
                  {formError}
                </Alert>
              )}

              <TextField
                id="auth-otp-code"
                name="code"
                type="text"
                label="6-digit verification code"
                required
                fullWidth
                value={code}
                error={Boolean(codeError)}
                helperText={codeError ?? "Paste or type the 6 digits from your email."}
                slotProps={{
                  htmlInput: {
                    inputMode: "numeric",
                    autoComplete: "one-time-code",
                    autoCorrect: "off",
                    autoCapitalize: "none",
                    spellCheck: false,
                    maxLength: 6,
                    pattern: "\\d{6}",
                    enterKeyHint: "done",
                    "aria-invalid": Boolean(codeError),
                    style: {
                      letterSpacing: "0.35em",
                      fontSize: "1.25rem",
                      fontWeight: 600,
                      textAlign: "center",
                    },
                  },
                }}
                onChange={(e) => {
                  const digitsOnly = e.target.value.replace(/\D/g, "").slice(0, 6);
                  setCode(digitsOnly);
                  if (codeError && validateCodeField(digitsOnly) === null) {
                    setCodeError(null);
                  }
                }}
                onBlur={() => {
                  if (code.length > 0 || codeTouched) {
                    setCodeTouched(true);
                    setCodeError(validateCodeField(code));
                  }
                }}
              />

              <Button
                type="submit"
                variant="contained"
                size="large"
                fullWidth
                disabled={submitting}
              >
                {submitting ? "Verifying..." : "Verify & continue"}
              </Button>

              <Stack direction="row" sx={{ justifyContent: "space-between", alignItems: "center" }}>
                <Button
                  type="button"
                  variant="text"
                  size="small"
                  startIcon={<AppIcon icon={ArrowLeft01Icon} size={16} />}
                  onClick={() => {
                    setStep("email");
                    setCode("");
                    setCodeError(null);
                    setFormError(null);
                  }}
                >
                  Change email
                </Button>

                <Button
                  type="button"
                  variant="text"
                  size="small"
                  disabled={resendCooldown > 0 || submitting}
                  startIcon={
                    <AppIcon icon={resendCooldown > 0 ? Clock01Icon : RefreshIcon} size={16} />
                  }
                  onClick={handleResend}
                >
                  {resendCooldown > 0 ? `Resend in ${resendCooldown}s` : "Resend code"}
                </Button>
              </Stack>
            </Stack>
          </Box>
        )}
      </CardContent>
    </Card>
  );
}
