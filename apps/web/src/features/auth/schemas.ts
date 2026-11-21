import { z } from "zod";

export const emailSchema = z
  .string()
  .trim()
  .min(1, "Please enter your email address.")
  .email("Enter an email address in the format you@example.com.");

export const otpCodeSchema = z
  .string()
  .trim()
  .regex(/^\d{6}$/, "Enter the 6-digit verification code from your email.");

export const usernameSchema = z
  .string()
  .trim()
  .regex(
    /^[a-zA-Z0-9_]{3,24}$/,
    "Use 3–24 letters, numbers, or underscores.",
  );

export const displayNameSchema = z
  .string()
  .trim()
  .min(2, "Display name must be at least 2 characters.")
  .max(60, "Display name must be 60 characters or fewer.");

export const JOURNEY_FOCUS_OPTIONS = [
  { id: "daily_steps", label: "Daily walking & steps" },
  { id: "hydration", label: "Hydration habit" },
  { id: "mindful_meals", label: "Mindful meals" },
  { id: "strength_movement", label: "Strength & movement" },
  { id: "sleep_recovery", label: "Sleep & recovery" },
  { id: "accountability_buddies", label: "Accountability buddies" },
] as const;

export function parseWeightToGrams(
  raw: string,
  unit: "kg" | "lb",
): number | undefined {
  const trimmed = raw.trim();
  if (!trimmed) return undefined;
  const value = Number(trimmed);
  if (!Number.isFinite(value) || value <= 0) return undefined;
  const grams = unit === "kg" ? Math.round(value * 1000) : Math.round(value * 453.59237);
  if (grams <= 20000 || grams >= 500000) return undefined;
  return grams;
}
