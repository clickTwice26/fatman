"use client";

import { useEffect, useRef, useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import {
  ArrowLeft01Icon,
  ArrowRight01Icon,
  CheckmarkCircle02Icon,
  Shield01Icon,
  SparklesIcon,
  UserIcon,
} from "@hugeicons/core-free-icons";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Card from "@mui/material/Card";
import CardContent from "@mui/material/CardContent";
import Checkbox from "@mui/material/Checkbox";
import FormControl from "@mui/material/FormControl";
import FormControlLabel from "@mui/material/FormControlLabel";
import FormGroup from "@mui/material/FormGroup";
import FormLabel from "@mui/material/FormLabel";
import Radio from "@mui/material/Radio";
import RadioGroup from "@mui/material/RadioGroup";
import Stack from "@mui/material/Stack";
import Step from "@mui/material/Step";
import StepLabel from "@mui/material/StepLabel";
import Stepper from "@mui/material/Stepper";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { AppIcon } from "@/components/AppIcon";
import { useAuth } from "@/features/auth/AuthProvider";
import {
  displayNameSchema,
  JOURNEY_FOCUS_OPTIONS,
  parseWeightToGrams,
  usernameSchema,
} from "@/features/auth/schemas";
import {
  ApiError,
  checkUsernameAvailable,
  completeOnboarding,
} from "@/lib/api";

const STEPS = ["Your profile", "Privacy & goals", "Quest focus"] as const;

export function OnboardingWizard() {
  const router = useRouter();
  const { accessToken, setUser } = useAuth();

  const [activeStep, setActiveStep] = useState(0);
  const [username, setUsername] = useState("");
  const [usernameError, setUsernameError] = useState<string | null>(null);
  const [usernameStatus, setUsernameStatus] = useState<"idle" | "checking" | "available" | "taken">("idle");

  const [displayName, setDisplayName] = useState("");
  const [displayNameError, setDisplayNameError] = useState<string | null>(null);
  const [bio, setBio] = useState("");

  const [unitPref, setUnitPref] = useState<"kg" | "lb">("kg");
  const [weightPrivacy, setWeightPrivacy] = useState<"private" | "squads_only" | "public">("squads_only");
  const [startWeight, setStartWeight] = useState("");
  const [goalWeight, setGoalWeight] = useState("");

  const [journeyFocus, setJourneyFocus] = useState<string[]>(["daily_steps", "accountability_buddies"]);
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const stepHeadingRef = useRef<HTMLHeadingElement>(null);

  useEffect(() => {
    stepHeadingRef.current?.focus();
  }, [activeStep]);

  const verifyUsernameAvailability = async (candidate: string) => {
    const parsed = usernameSchema.safeParse(candidate);
    if (!parsed.success) {
      setUsernameError(parsed.error.issues[0]?.message ?? "Invalid username.");
      setUsernameStatus("idle");
      return false;
    }
    setUsernameError(null);
    if (!accessToken) return true;
    setUsernameStatus("checking");
    try {
      const res = await checkUsernameAvailable(parsed.data, accessToken);
      if (res.available) {
        setUsernameStatus("available");
        return true;
      }
      setUsernameStatus("taken");
      setUsernameError("That username is already taken. Try another one.");
      return false;
    } catch {
      setUsernameStatus("idle");
      return true;
    }
  };

  const handleNextStep = async () => {
    setFormError(null);
    if (activeStep === 0) {
      const uRes = usernameSchema.safeParse(username);
      const dRes = displayNameSchema.safeParse(displayName);
      setUsernameError(uRes.success ? null : (uRes.error.issues[0]?.message ?? "Invalid username."));
      setDisplayNameError(dRes.success ? null : (dRes.error.issues[0]?.message ?? "Invalid display name."));
      if (!uRes.success || !dRes.success) return;

      const available = await verifyUsernameAvailability(uRes.data);
      if (!available) return;
    }
    setActiveStep((s) => Math.min(STEPS.length - 1, s + 1));
  };

  const toggleFocus = (id: string) => {
    setJourneyFocus((prev) =>
      prev.includes(id) ? prev.filter((item) => item !== id) : [...prev, id],
    );
  };

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!accessToken) {
      setFormError("Your session expired. Please sign in again.");
      return;
    }
    setFormError(null);
    setSubmitting(true);
    try {
      const startGrams = parseWeightToGrams(startWeight, unitPref);
      const goalGrams = parseWeightToGrams(goalWeight, unitPref);
      const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";

      const res = await completeOnboarding(
        {
          username: username.trim(),
          display_name: displayName.trim(),
          bio: bio.trim(),
          unit_pref: unitPref,
          start_weight_grams: startGrams,
          current_weight_grams: startGrams,
          goal_weight_grams: goalGrams,
          weight_privacy: weightPrivacy,
          journey_focus: journeyFocus,
          timezone,
        },
        accessToken,
      );
      setUser(res.user);
      router.push("/dashboard");
    } catch (error) {
      if (error instanceof ApiError) {
        setFormError(error.message);
        if (error.code === "USERNAME_TAKEN") {
          setActiveStep(0);
          setUsernameError(error.message);
        }
      } else {
        setFormError("Unable to complete setup. Please try again.");
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Card sx={{ maxWidth: 600, width: "100%", mx: "auto" }}>
      <CardContent sx={{ p: { xs: 3, sm: 4 } }}>
        <Stack spacing={4}>
          <Stepper activeStep={activeStep} alternativeLabel>
            {STEPS.map((label) => (
              <Step key={label}>
                <StepLabel>{label}</StepLabel>
              </Step>
            ))}
          </Stepper>

          {formError && (
            <Alert severity="error" role="alert">
              {formError}
            </Alert>
          )}

          <Box component="form" method="POST" onSubmit={handleSubmit} noValidate>
            {activeStep === 0 && (
              <Stack spacing={3}>
                <Stack direction="row" spacing={1.5} sx={{ alignItems: "center" }}>
                  <AppIcon icon={UserIcon} size={24} />
                  <Typography
                    ref={stepHeadingRef}
                    tabIndex={-1}
                    variant="h5"
                    component="h1"
                    sx={{ outline: "none" }}
                  >
                    Tell the community what to call you
                  </Typography>
                </Stack>

                <TextField
                  id="onboarding-username"
                  name="username"
                  label="Username"
                  required
                  fullWidth
                  value={username}
                  error={Boolean(usernameError)}
                  helperText={
                    usernameError ??
                    (usernameStatus === "available"
                      ? "Looks great — this username is available!"
                      : "3–24 characters using letters, numbers, or underscores.")
                  }
                  slotProps={{
                    htmlInput: {
                      autoComplete: "username",
                      autoCorrect: "off",
                      autoCapitalize: "none",
                      spellCheck: false,
                      enterKeyHint: "next",
                      "aria-invalid": Boolean(usernameError),
                    },
                  }}
                  onChange={(e) => {
                    setUsername(e.target.value);
                    setUsernameStatus("idle");
                    if (usernameError && usernameSchema.safeParse(e.target.value).success) {
                      setUsernameError(null);
                    }
                  }}
                  onBlur={() => {
                    if (username.trim().length > 0) {
                      void verifyUsernameAvailability(username);
                    }
                  }}
                />

                <TextField
                  id="onboarding-display-name"
                  name="display_name"
                  label="Display name"
                  required
                  fullWidth
                  value={displayName}
                  error={Boolean(displayNameError)}
                  helperText={displayNameError ?? "First name or nickname shown on posts & quest squads."}
                  slotProps={{
                    htmlInput: {
                      autoComplete: "name",
                      enterKeyHint: "next",
                      "aria-invalid": Boolean(displayNameError),
                    },
                  }}
                  onChange={(e) => {
                    setDisplayName(e.target.value);
                    if (displayNameError && displayNameSchema.safeParse(e.target.value).success) {
                      setDisplayNameError(null);
                    }
                  }}
                  onBlur={() => {
                    if (displayName.length > 0) {
                      const res = displayNameSchema.safeParse(displayName);
                      setDisplayNameError(res.success ? null : (res.error.issues[0]?.message ?? null));
                    }
                  }}
                />

                <TextField
                  id="onboarding-bio"
                  name="bio"
                  label="Short intro (optional)"
                  multiline
                  minRows={2}
                  fullWidth
                  value={bio}
                  helperText="Share what motivates you or what kind of accountability buddy you're looking for."
                  onChange={(e) => setBio(e.target.value.slice(0, 280))}
                />

                <Stack direction="row" sx={{ justifyContent: "flex-end" }}>
                  <Button
                    type="button"
                    variant="contained"
                    onClick={() => void handleNextStep()}
                    endIcon={<AppIcon icon={ArrowRight01Icon} size={18} />}
                  >
                    Next: Privacy & goals
                  </Button>
                </Stack>
              </Stack>
            )}

            {activeStep === 1 && (
              <Stack spacing={3}>
                <Stack direction="row" spacing={1.5} sx={{ alignItems: "center" }}>
                  <AppIcon icon={Shield01Icon} size={24} />
                  <Typography
                    ref={stepHeadingRef}
                    tabIndex={-1}
                    variant="h5"
                    component="h1"
                    sx={{ outline: "none" }}
                  >
                    Your weight privacy & optional goals
                  </Typography>
                </Stack>

                <FormControl component="fieldset">
                  <FormLabel component="legend">Preferred weight unit</FormLabel>
                  <RadioGroup
                    row
                    name="unit_pref"
                    value={unitPref}
                    onChange={(e) => setUnitPref(e.target.value as "kg" | "lb")}
                  >
                    <FormControlLabel value="kg" control={<Radio />} label="Kilograms (kg)" />
                    <FormControlLabel value="lb" control={<Radio />} label="Pounds (lb)" />
                  </RadioGroup>
                </FormControl>

                <FormControl component="fieldset">
                  <FormLabel component="legend">Who can see your weight numbers?</FormLabel>
                  <RadioGroup
                    name="weight_privacy"
                    value={weightPrivacy}
                    onChange={(e) =>
                      setWeightPrivacy(e.target.value as "private" | "squads_only" | "public")
                    }
                  >
                    <FormControlLabel
                      value="private"
                      control={<Radio />}
                      label="Only me (Private) — share streaks & wins without numbers"
                    />
                    <FormControlLabel
                      value="squads_only"
                      control={<Radio />}
                      label="My quest squads only — visible to teammates you join quests with"
                    />
                    <FormControlLabel
                      value="public"
                      control={<Radio />}
                      label="Fatman community — visible on your member profile"
                    />
                  </RadioGroup>
                </FormControl>

                <Stack direction={{ xs: "column", sm: "row" }} spacing={2}>
                  <TextField
                    id="onboarding-start-weight"
                    name="start_weight"
                    type="text"
                    label={`Starting weight (${unitPref}, optional)`}
                    fullWidth
                    value={startWeight}
                    slotProps={{ htmlInput: { inputMode: "decimal" } }}
                    onChange={(e) => setStartWeight(e.target.value)}
                  />
                  <TextField
                    id="onboarding-goal-weight"
                    name="goal_weight"
                    type="text"
                    label={`Goal weight (${unitPref}, optional)`}
                    fullWidth
                    value={goalWeight}
                    slotProps={{ htmlInput: { inputMode: "decimal" } }}
                    onChange={(e) => setGoalWeight(e.target.value)}
                  />
                </Stack>

                <Stack direction="row" sx={{ justifyContent: "space-between" }}>
                  <Button
                    type="button"
                    variant="outlined"
                    onClick={() => setActiveStep(0)}
                    startIcon={<AppIcon icon={ArrowLeft01Icon} size={18} />}
                  >
                    Back
                  </Button>
                  <Button
                    type="button"
                    variant="contained"
                    onClick={() => void handleNextStep()}
                    endIcon={<AppIcon icon={ArrowRight01Icon} size={18} />}
                  >
                    Next: Quest focus
                  </Button>
                </Stack>
              </Stack>
            )}

            {activeStep === 2 && (
              <Stack spacing={3}>
                <Stack direction="row" spacing={1.5} sx={{ alignItems: "center" }}>
                  <AppIcon icon={SparklesIcon} size={24} />
                  <Typography
                    ref={stepHeadingRef}
                    tabIndex={-1}
                    variant="h5"
                    component="h1"
                    sx={{ outline: "none" }}
                  >
                    What habits do you want to build together?
                  </Typography>
                </Stack>

                <FormControl component="fieldset">
                  <FormLabel component="legend">Choose any areas that interest you</FormLabel>
                  <FormGroup>
                    {JOURNEY_FOCUS_OPTIONS.map((opt) => (
                      <FormControlLabel
                        key={opt.id}
                        control={
                          <Checkbox
                            checked={journeyFocus.includes(opt.id)}
                            onChange={() => toggleFocus(opt.id)}
                          />
                        }
                        label={opt.label}
                      />
                    ))}
                  </FormGroup>
                </FormControl>

                <Stack direction="row" sx={{ justifyContent: "space-between" }}>
                  <Button
                    type="button"
                    variant="outlined"
                    onClick={() => setActiveStep(1)}
                    startIcon={<AppIcon icon={ArrowLeft01Icon} size={18} />}
                  >
                    Back
                  </Button>
                  <Button
                    type="submit"
                    variant="contained"
                    disabled={submitting}
                    startIcon={<AppIcon icon={CheckmarkCircle02Icon} size={18} />}
                  >
                    {submitting ? "Saving profile..." : "Complete setup"}
                  </Button>
                </Stack>
              </Stack>
            )}
          </Box>
        </Stack>
      </CardContent>
    </Card>
  );
}
