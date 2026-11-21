"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import {
  FireIcon,
  Logout01Icon,
  Shield01Icon,
  Target01Icon,
} from "@hugeicons/core-free-icons";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Card from "@mui/material/Card";
import CardContent from "@mui/material/CardContent";
import Chip from "@mui/material/Chip";
import CircularProgress from "@mui/material/CircularProgress";
import Container from "@mui/material/Container";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { AppIcon } from "@/components/AppIcon";
import { useAuth } from "@/features/auth/AuthProvider";

export default function DashboardPage() {
  const router = useRouter();
  const { user, loading, logout } = useAuth();

  useEffect(() => {
    if (loading) return;
    if (!user) {
      router.replace("/auth");
    } else if (!user.onboarding_completed) {
      router.replace("/onboarding");
    }
  }, [loading, user, router]);

  if (loading || !user || !user.onboarding_completed) {
    return (
      <Box
        component="main"
        sx={{ minHeight: "100dvh", display: "grid", placeItems: "center" }}
      >
        <CircularProgress aria-label="Loading your dashboard" />
      </Box>
    );
  }

  const displayName = user.profile?.display_name || user.username || user.email;

  return (
    <Box component="main" sx={{ minHeight: "100dvh", py: { xs: 4, md: 6 } }}>
      <Container maxWidth="lg">
        <Stack spacing={4}>
          <Stack
            direction={{ xs: "column", sm: "row" }}
            spacing={2}
            sx={{ justifyContent: "space-between", alignItems: { sm: "center" } }}
          >
            <Stack spacing={0.5}>
              <Typography variant="h4" component="h1">
                Welcome back, {displayName}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                @{user.username} · {user.email}
              </Typography>
            </Stack>

            <Button
              type="button"
              variant="outlined"
              startIcon={<AppIcon icon={Logout01Icon} size={18} />}
              onClick={async () => {
                await logout();
                router.push("/auth");
              }}
            >
              Sign out
            </Button>
          </Stack>

          <Card>
            <CardContent sx={{ p: 3 }}>
              <Stack spacing={2}>
                <Stack direction="row" spacing={1} sx={{ alignItems: "center" }}>
                  <AppIcon icon={FireIcon} size={22} />
                  <Typography variant="h6">Your Journey Setup</Typography>
                </Stack>

                {user.profile?.bio && (
                  <Typography variant="body1">{user.profile.bio}</Typography>
                )}

                <Stack direction="row" spacing={1} sx={{ flexWrap: "wrap", rowGap: 1 }}>
                  <Chip
                    icon={<AppIcon icon={Shield01Icon} size={16} />}
                    label={`Weight privacy: ${user.profile?.weight_privacy ?? "private"}`}
                    variant="outlined"
                  />
                  <Chip
                    icon={<AppIcon icon={Target01Icon} size={16} />}
                    label={`Unit: ${user.profile?.unit_pref ?? "kg"}`}
                    variant="outlined"
                  />
                  {(user.profile?.journey_focus ?? []).map((focus) => (
                    <Chip key={focus} label={focus.replaceAll("_", " ")} color="primary" size="small" />
                  ))}
                </Stack>
              </Stack>
            </CardContent>
          </Card>
        </Stack>
      </Container>
    </Box>
  );
}
