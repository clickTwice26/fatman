"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import Box from "@mui/material/Box";
import Container from "@mui/material/Container";
import { useAuth } from "@/features/auth/AuthProvider";
import { EmailOtpForm } from "@/features/auth/EmailOtpForm";

export default function AuthPage() {
  const router = useRouter();
  const { user, loading } = useAuth();

  useEffect(() => {
    if (!loading && user) {
      router.replace(user.onboarding_completed ? "/dashboard" : "/onboarding");
    }
  }, [loading, user, router]);

  return (
    <Box
      component="main"
      sx={{
        minHeight: "100dvh",
        display: "flex",
        alignItems: "center",
        py: 6,
      }}
    >
      <Container maxWidth="sm">
        <EmailOtpForm />
      </Container>
    </Box>
  );
}
