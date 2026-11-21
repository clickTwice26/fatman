"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import Box from "@mui/material/Box";
import CircularProgress from "@mui/material/CircularProgress";
import Container from "@mui/material/Container";
import { useAuth } from "@/features/auth/AuthProvider";
import { OnboardingWizard } from "@/features/auth/OnboardingWizard";

export default function OnboardingPage() {
  const router = useRouter();
  const { user, loading } = useAuth();

  useEffect(() => {
    if (loading) return;
    if (!user) {
      router.replace("/auth");
    } else if (user.onboarding_completed) {
      router.replace("/dashboard");
    }
  }, [loading, user, router]);

  if (loading || !user || user.onboarding_completed) {
    return (
      <Box
        component="main"
        sx={{ minHeight: "100dvh", display: "grid", placeItems: "center" }}
      >
        <CircularProgress aria-label="Loading onboarding" />
      </Box>
    );
  }

  return (
    <Box component="main" sx={{ minHeight: "100dvh", py: { xs: 4, md: 8 } }}>
      <Container maxWidth="md">
        <OnboardingWizard />
      </Container>
    </Box>
  );
}
