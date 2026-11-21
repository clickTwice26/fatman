import {
  ArrowRight01Icon,
  FireIcon,
  Mail01Icon,
  Message01Icon,
  Shield01Icon,
  Target01Icon,
  UserGroupIcon,
} from "@hugeicons/core-free-icons";
import Box from "@mui/material/Box";
import Card from "@mui/material/Card";
import CardContent from "@mui/material/CardContent";
import Chip from "@mui/material/Chip";
import Container from "@mui/material/Container";
import Grid from "@mui/material/Grid";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { AppIcon } from "@/components/AppIcon";
import { LinkButton } from "@/components/LinkButton";

const PILLARS = [
  {
    icon: Target01Icon,
    title: "Collaborative Weight-Loss Quests",
    description:
      "Team up in solo, duo, or squad challenges for daily steps, hydration, mindful meals, and consistent check-ins.",
  },
  {
    icon: Message01Icon,
    title: "Real-Time Squad & Buddy Chat",
    description:
      "Encourage your squad in live rooms or check in 1:1 with an accountability buddy whenever motivation dips.",
  },
  {
    icon: UserGroupIcon,
    title: "Judgment-Free Community Feed",
    description:
      "Celebrate non-scale victories, daily wins, and honest setbacks with supportive reactions — never shaming.",
  },
  {
    icon: Shield01Icon,
    title: "Privacy-First Progress Tracking",
    description:
      "You control whether your weights and milestones are public, shared with your squad, or strictly private.",
  },
] as const;

export default function HomePage() {
  return (
    <Box component="main" sx={{ minHeight: "100dvh", py: { xs: 6, md: 10 } }}>
      <Container maxWidth="lg">
        <Stack spacing={6}>
          <Stack spacing={3} sx={{ alignItems: "flex-start", maxWidth: 720 }}>
            <Chip
              icon={<AppIcon icon={FireIcon} size={16} />}
              label="Accountability-first weight-loss community"
              color="primary"
              variant="outlined"
              sx={{ fontWeight: 600 }}
            />
            <Typography variant="h1" sx={{ fontSize: { xs: "2.25rem", md: "3.5rem" }, lineHeight: 1.1 }}>
              Lose weight together, one daily quest at a time.
            </Typography>
            <Typography variant="h6" color="text.secondary" sx={{ fontWeight: 400, lineHeight: 1.6 }}>
              Fatman connects people on a weight-loss journey for real-time encouragement, shared habit quests,
              and honest progress tracking — starting with simple passwordless email onboarding.
            </Typography>
            <Stack direction={{ xs: "column", sm: "row" }} spacing={2} sx={{ pt: 1 }}>
              <LinkButton
                href="/auth"
                variant="contained"
                size="large"
                startIcon={<AppIcon icon={Mail01Icon} size={20} />}
                endIcon={<AppIcon icon={ArrowRight01Icon} size={18} />}
              >
                Continue with Email
              </LinkButton>
              <LinkButton href="/quests" variant="outlined" size="large">
                Explore Quests
              </LinkButton>
            </Stack>
          </Stack>

          <Grid container spacing={3}>
            {PILLARS.map((pillar) => (
              <Grid key={pillar.title} size={{ xs: 12, sm: 6 }}>
                <Card sx={{ height: "100%" }}>
                  <CardContent sx={{ p: 3 }}>
                    <Stack spacing={1.5}>
                      <Box
                        sx={{
                          width: 44,
                          height: 44,
                          borderRadius: 3,
                          display: "inline-flex",
                          alignItems: "center",
                          justifyContent: "center",
                          bgcolor: "primary.main",
                          color: "primary.contrastText",
                        }}
                      >
                        <AppIcon icon={pillar.icon} size={22} />
                      </Box>
                      <Typography variant="h6">{pillar.title}</Typography>
                      <Typography variant="body2" color="text.secondary" sx={{ lineHeight: 1.65 }}>
                        {pillar.description}
                      </Typography>
                    </Stack>
                  </CardContent>
                </Card>
              </Grid>
            ))}
          </Grid>
        </Stack>
      </Container>
    </Box>
  );
}
